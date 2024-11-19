package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	"github.com/hetznercloud/csi-driver/internal/driver"
	"github.com/hetznercloud/csi-driver/internal/metrics"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/envutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
)

func parseLogLevel(lvl string) slog.Level {
	switch strings.ToLower(lvl) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// CreateLogger prepares a logger according to LOG_LEVEL environment variable.
func CreateLogger() *slog.Logger {
	logLevel := parseLogLevel(os.Getenv("LOG_LEVEL"))
	options := slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
	}
	th := slog.NewTextHandler(os.Stdout, &options)
	logger := slog.New(th)

	return logger
}

// GetEnableProvidedByTopology parses the ENABLE_PROVIDED_BY_TOPOLOGY environment variable and returns false by default.
func GetEnableProvidedByTopology() bool {
	var enableProvidedByTopology bool
	if featFlag, exists := os.LookupEnv("ENABLE_PROVIDED_BY_TOPOLOGY"); exists {
		enableProvidedByTopology, _ = strconv.ParseBool(featFlag)
	}
	return enableProvidedByTopology
}

// CreateListener creates and binds the unix socket in location specified by the CSI_ENDPOINT environment variable.
func CreateListener() (net.Listener, error) {
	endpoint := os.Getenv("CSI_ENDPOINT")
	if endpoint == "" {
		return nil, errors.New("you need to specify an endpoint via the CSI_ENDPOINT env var")
	}
	if !strings.HasPrefix(endpoint, "unix://") {
		return nil, errors.New("endpoint must start with unix://")
	}
	endpoint = endpoint[7:] // strip unix://

	if err := os.Remove(endpoint); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to remove socket file at %s: %s", endpoint, err)
	}

	return net.Listen("unix", endpoint)
}

// CreateMetrics prepares a metrics client pointing at METRICS_ENDPOINT environment variable (will fallback)
// It will start the metrics HTTP listener depending on the ENABLE_METRICS environment variable.
func CreateMetrics(logger *slog.Logger) *metrics.Metrics {
	metricsEndpoint := os.Getenv("METRICS_ENDPOINT")
	if metricsEndpoint == "" {
		// Use a default endpoint
		metricsEndpoint = ":9189"
	}

	m := metrics.New(
		logger,
		metricsEndpoint,
	)

	enableMetrics := true // Default to true to keep the old behavior of exporting them always. This is deprecated
	if enableMetricsEnv := os.Getenv("ENABLE_METRICS"); enableMetricsEnv != "" {
		var err error
		enableMetrics, err = strconv.ParseBool(enableMetricsEnv)
		if err != nil {
			logger.Error(
				"ENABLE_METRICS can only contain a boolean value, true or false",
				"err", err,
			)
			os.Exit(1)
		}
	} else {
		logger.Warn(
			"the environment variable ENABLE_METRICS should be set to true, you can disable metrics by setting this env to false. Not specifying the ENV is deprecated. With v1.9.0 we will change the default to false and in v1.10.0 we will fail on start when the ENABLE_METRICS is not specified.",
		)
	}
	if enableMetrics {
		m.Serve()
	}

	return m
}

// CreateHcloudClient creates a hcloud.Client using  various environment variables to guide configuration
func CreateHcloudClient(metricsRegistry *prometheus.Registry, logger *slog.Logger) (*hcloud.Client, error) {
	// apiToken can be set via HCLOUD_TOKEN (preferred) or HCLOUD_TOKEN_FILE
	apiToken, err := envutil.LookupEnvWithFile("HCLOUD_TOKEN")
	if err != nil {
		return nil, err
	}
	if apiToken == "" {
		return nil, fmt.Errorf("you need to provide an API token via the HCLOUD_TOKEN or HCLOUD_TOKEN_FILE env var")
	}

	if len(apiToken) != 64 {
		logger.Warn(fmt.Sprintf("unrecognized token format, expected 64 characters, got %d, proceeding anyway", len(apiToken)))
	}

	opts := []hcloud.ClientOption{
		hcloud.WithToken(apiToken),
		hcloud.WithApplication("csi-driver", driver.PluginVersion),
		hcloud.WithInstrumentation(metricsRegistry),
	}
	hcloudEndpoint := os.Getenv("HCLOUD_ENDPOINT")
	if hcloudEndpoint != "" {
		opts = append(opts, hcloud.WithEndpoint(hcloudEndpoint))
	}
	enableDebug := os.Getenv("HCLOUD_DEBUG")
	if enableDebug != "" {
		opts = append(opts, hcloud.WithDebugWriter(os.Stdout))
	}

	pollingInterval := 3
	if customPollingInterval := os.Getenv("HCLOUD_POLLING_INTERVAL_SECONDS"); customPollingInterval != "" {
		tmp, err := strconv.Atoi(customPollingInterval)
		if err != nil || tmp < 1 {
			return nil, errors.New("entered polling interval configuration is not a integer that is higher than 1")
		}
		logger.Info(
			"got custom configuration for polling interval",
			"interval", customPollingInterval,
		)

		pollingInterval = tmp
	}

	opts = append(opts, hcloud.WithPollOpts(hcloud.PollOpts{
		BackoffFunc: hcloud.ExponentialBackoffWithOpts(hcloud.ExponentialBackoffOpts{
			Base:       time.Duration(pollingInterval) * time.Second,
			Multiplier: 2,
			Cap:        10 * time.Second,
		}),
	}))

	return hcloud.NewClient(opts...), nil
}

// GetServer retrieves the hcloud server the application is running on.
func GetServer(logger *slog.Logger, hcloudClient *hcloud.Client, metadataClient *metadata.Client) (*hcloud.Server, error) {
	hcloudServerID, err := getServerID(logger, hcloudClient, metadataClient)
	if err != nil {
		return nil, err
	}
	logger.Debug("fetching server")
	server, _, err := hcloudClient.Server.GetByID(context.Background(), hcloudServerID)
	if err != nil {
		return nil, err
	}

	// Cover potential cases where the server is not found. This results in a
	// nil server object and nil error. If we do not do this, we will panic
	// when trying to log the server.Name.
	if server == nil {
		return nil, errors.New("could not determine server")
	}

	logger.Info("fetched server", "server-name", server.Name)

	return server, nil
}

func getServerID(logger *slog.Logger, hcloudClient *hcloud.Client, metadataClient *metadata.Client) (int64, error) {
	if s := os.Getenv("HCLOUD_SERVER_ID"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid server id in HCLOUD_SERVER_ID env var: %s", err)
		}
		logger.Debug(
			"using server id from HCLOUD_SERVER_ID env var",
			"server-id", id,
		)
		return id, nil
	}

	if s := os.Getenv("KUBE_NODE_NAME"); s != "" {
		server, _, err := hcloudClient.Server.GetByName(context.Background(), s)
		if err != nil {
			return 0, fmt.Errorf("error while getting server through node name: %s", err)
		}
		if server != nil {
			logger.Debug(
				"using server name from KUBE_NODE_NAME env var",
				"server-id", server.ID,
			)
			return server.ID, nil
		}
		logger.Debug(
			"server not found by name, fallback to metadata service",
			"err", err,
		)
	}

	logger.Debug(
		"getting instance id from metadata service",
	)
	id, err := metadataClient.InstanceID()
	if err != nil {
		return 0, fmt.Errorf("failed to get instance id from metadata service: %s", err)
	}
	return id, nil
}

func CreateGRPCServer(logger *slog.Logger, metricsInterceptor grpc.UnaryServerInterceptor) *grpc.Server {
	requestLogger := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		isProbe := info.FullMethod == "/csi.v1.Identity/Probe"

		if !isProbe {
			logger.Debug(
				"handling request",
				"method", info.FullMethod,
				"req", req,
			)
		}
		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error(
				"handler failed",
				"err", err,
			)
		} else if !isProbe {
			logger.Debug("finished handling request", "method", info.FullMethod, "err", err)
		}
		return resp, err
	}

	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			requestLogger,
			metricsInterceptor,
		),
	)
}
