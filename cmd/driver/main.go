package main

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/hetznercloud/hcloud-go/hcloud/metadata"
	"google.golang.org/grpc"

	"github.com/hetznercloud/csi-driver/api"
	"github.com/hetznercloud/csi-driver/driver"
	"github.com/hetznercloud/csi-driver/metrics"
	"github.com/hetznercloud/csi-driver/volumes"
)

var logger log.Logger

func main() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = level.NewFilter(logger, parseLogLevel(os.Getenv("LOG_LEVEL")))
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
	if enableDebug != "" {
		opts = append(opts, hcloud.WithDebugWriter(os.Stdout))
	}

	pollingInterval := 1
	if customPollingInterval := os.Getenv("HCLOUD_POLLING_INTERVAL_SECONDS"); customPollingInterval != "" {
		tmp, err := strconv.Atoi(customPollingInterval)
		if err != nil || tmp < 1 {
			level.Error(logger).Log(
				"msg", "entered polling interval configuration is not a integer that is higher than 1",
			)
			os.Exit(2)
		}
		level.Info(logger).Log(
			"msg", "got custom configuration for polling interval",
			"interval", customPollingInterval,
		)

		pollingInterval = tmp
	}
	opts = append(opts, hcloud.WithPollInterval(time.Duration(pollingInterval)*time.Second))

	metricsEndpoint := os.Getenv("METRICS_ENDPOINT")
	if metricsEndpoint == "" {
		// Use a default endpoint
		metricsEndpoint = ":9189"
	}

	m := metrics.New(
		log.With(logger, "component", "metrics-service"),
		metricsEndpoint,
	)
	opts = append(opts, hcloud.WithInstrumentation(m.Registry()))

	hcloudClient := hcloud.NewClient(opts...)

	metadataClient := metadata.NewClient(metadata.WithInstrumentation(m.Registry()))

	hcloudServerID := getServerID(hcloudClient, metadataClient)
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

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				requestLogger(log.With(logger, "component", "grpc-server")),
				m.UnaryServerInterceptor(),
			),
		),
	)

	proto.RegisterControllerServer(grpcServer, controllerService)
	proto.RegisterIdentityServer(grpcServer, identityService)
	proto.RegisterNodeServer(grpcServer, nodeService)

	m.InitializeMetrics(grpcServer)
	enableMetrics := true // Default to true to keep the old behavior of exporting them always. This is deprecated
	if enableMetricsEnv := os.Getenv("ENABLE_METRICS"); enableMetricsEnv != "" {
		enableMetrics, err = strconv.ParseBool(enableMetricsEnv)
		if err != nil {
			level.Error(logger).Log(
				"msg", "ENABLE_METRICS can only contain a boolean value, true or false",
				"err", err,
			)
			os.Exit(1)
		}
	} else {
		level.Warn(logger).Log(
			"msg", "the environment variable ENABLE_METRICS should be set to true, you can disable metrics by setting this env to false. Not specifying the ENV is deprecated. With v1.9.0 we will change the default to false and in v1.10.0 we will fail on start when the ENABLE_METRICS is not specified.",
		)
	}
	if enableMetrics {
		m.Serve()
	}

	identityService.SetReady(true)

	if err := grpcServer.Serve(listener); err != nil {
		level.Error(logger).Log(
			"msg", "grpc server failed",
			"err", err,
		)
		os.Exit(1)
	}
}

func getServerID(hcloudClient *hcloud.Client, metadataClient *metadata.Client) int {
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

	if s := os.Getenv("KUBE_NODE_NAME"); s != "" {
		server, _, err := hcloudClient.Server.GetByName(context.Background(), s)
		if err != nil {
			level.Debug(logger).Log(
				"msg", "error while getting server through node name",
				"err", err,
			)
		}
		if server != nil {
			level.Debug(logger).Log(
				"msg", "using server name from KUBE_NODE_NAME env var",
				"server-id", server.ID,
			)
			return server.ID
		}
		level.Debug(logger).Log(
			"msg", "server not found by name, fallback to metadata service",
			"err", err,
		)
	}

	level.Debug(logger).Log(
		"msg", "getting instance id from metadata service",
	)
	id, err := metadataClient.InstanceID()
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to get instance id from metadata service",
			"err", err,
		)
		os.Exit(1)
	}
	return id
}

func parseLogLevel(lvl string) level.Option {
	switch lvl {
	case "debug":
		return level.AllowDebug()
	case "info":
		return level.AllowInfo()
	case "warn":
		return level.AllowWarn()
	case "error":
		return level.AllowError()
	default:
		return level.AllowInfo()
	}
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
