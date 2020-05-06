package main

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"google.golang.org/grpc"

	"hetzner.cloud/csi/api"
	"hetzner.cloud/csi/driver"
	"hetzner.cloud/csi/metrics"
	"hetzner.cloud/csi/volumes"
)

var logger log.Logger

func main() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	endpoint := os.Getenv("CSI_ENDPOINT")
	if endpoint == "" {
		level.Error(logger).Log(
			"msg", "you need to specify an endpoint via the CSI_ENDPOINT env var",
		)
		os.Exit(2)
	}
	if !strings.HasPrefix(endpoint, "unix://") {
		level.Error(logger).Log(
			"msg", "endpoint must start with unix://",
		)
		os.Exit(2)
	}
	endpoint = endpoint[7:] // strip unix://

	if err := os.Remove(endpoint); err != nil && !os.IsNotExist(err) {
		level.Error(logger).Log(
			"msg", "failed to remove socket file",
			"path", endpoint,
			"err", err,
		)
		os.Exit(1)
	}

	apiToken := os.Getenv("HCLOUD_TOKEN")
	if apiToken == "" {
		level.Error(logger).Log(
			"msg", "you need to provide an API token via the HCLOUD_TOKEN env var",
		)
		os.Exit(2)
	}

	if len(apiToken) != 64 {
		level.Error(logger).Log(
			"msg", "entered token is invalid (must be exactly 64 characters long)",
		)
		os.Exit(2)
	}

	opts := []hcloud.ClientOption{
		hcloud.WithToken(apiToken),
		hcloud.WithApplication("csi-driver", driver.PluginVersion),
	}

	enableDebug := os.Getenv("HCLOUD_DEBUG")
	if enableDebug == "yes" {
		opts = append(opts, hcloud.WithDebugWriter(os.Stdout))
	}

	customPollingInterval := os.Getenv("HCLOUD_POLLING_INTERVAL")
	pollingInterval := 1
	if customPollingInterval != "" {
		level.Info(logger).Log(
			"msg", "Got custom configuration for polling interval",
		)
		tmp, err := strconv.Atoi(customPollingInterval)
		if err != nil {
			level.Error(logger).Log(
				"msg", "entered polling interval configuration is not a integer",
			)
			os.Exit(2)
		}
		pollingInterval = tmp
	}
	opts = append(opts, hcloud.WithPollInterval(time.Duration(pollingInterval)*time.Second))

	hcloudClient := hcloud.NewClient(opts...)

	hcloudServerID := getServerID()
	level.Debug(logger).Log("msg", "fetching server")
	server, _, err := hcloudClient.Server.GetByID(context.Background(), hcloudServerID)
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to fetch server",
			"err", err,
		)
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "fetched server", "server-name", server.Name)

	volumeService := volumes.NewIdempotentService(
		log.With(logger, "component", "idempotent-volume-service"),
		api.NewVolumeService(
			log.With(logger, "component", "api-volume-service"),
			hcloudClient,
		),
	)
	volumeMountService := volumes.NewLinuxMountService(
		log.With(logger, "component", "linux-mount-service"),
	)
	volumeResizeService := volumes.NewLinuxResizeService(
		log.With(logger, "component", "linux-resize-service"),
	)
	volumeStatsService := volumes.NewLinuxStatsService(
		log.With(logger, "component", "linux-stats-service"),
	)
	controllerService := driver.NewControllerService(
		log.With(logger, "component", "driver-controller-service"),
		volumeService,
		server.Datacenter.Location.Name,
	)
	identityService := driver.NewIdentityService(
		log.With(logger, "component", "driver-identity-service"),
	)
	nodeService := driver.NewNodeService(
		log.With(logger, "component", "driver-node-service"),
		server,
		volumeService,
		volumeMountService,
		volumeResizeService,
		volumeStatsService,
	)

	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to create listener",
			"err", err,
		)
		os.Exit(1)
	}

	metricsEndpoint := os.Getenv("METRICS_ENDPOINT")
	if metricsEndpoint == "" {
		// Use a default endpoint
		metricsEndpoint = ":9189"
	}

	metrics := metrics.New(
		log.With(logger, "component", "metrics-service"),
		metricsEndpoint,
	)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				requestLogger(log.With(logger, "component", "grpc-server")),
				metrics.UnaryServerInterceptor(),
			),
		),
	)

	proto.RegisterControllerServer(grpcServer, controllerService)
	proto.RegisterIdentityServer(grpcServer, identityService)
	proto.RegisterNodeServer(grpcServer, nodeService)

	metrics.InitializeMetrics(grpcServer)
	metrics.Serve()

	identityService.SetReady(true)

	if err := grpcServer.Serve(listener); err != nil {
		level.Error(logger).Log(
			"msg", "grpc server failed",
			"err", err,
		)
		os.Exit(1)
	}
}

func getServerID() int {
	if s := os.Getenv("HCLOUD_SERVER_ID"); s != "" {
		id, err := strconv.Atoi(s)
		if err != nil {
			level.Error(logger).Log(
				"msg", "invalid server id in HCLOUD_SERVER_ID env var",
				"err", err,
			)
			os.Exit(1)
		}
		level.Debug(logger).Log(
			"msg", "using server id from HCLOUD_SERVER_ID env var",
			"server-id", id,
		)
		return id
	}

	level.Debug(logger).Log(
		"msg", "getting instance id from metadata service",
	)
	id, err := getInstanceID()
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to get instance id from metadata service",
			"err", err,
		)
		os.Exit(1)
	}
	return id
}

func getInstanceID() (int, error) {
	resp, err := http.Get("http://169.254.169.254/2009-04-04/meta-data/instance-id")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(body))
}

func requestLogger(logger log.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		level.Debug(logger).Log(
			"msg", "handling request",
			"req", req,
		)
		resp, err := handler(ctx, req)
		if err != nil {
			level.Error(logger).Log(
				"msg", "handler failed",
				"err", err,
			)
		} else {
			level.Debug(logger).Log("msg", "finished handling request")
		}
		return resp, err
	}
}
