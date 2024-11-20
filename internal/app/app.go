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

// GetServerLocation retrieves the hcloud server the application is running on.
func GetServerLocation(logger *slog.Logger, hcloudClient *hcloud.Client, metadataClient *metadata.Client) (string, error) {
	// Option 1: Get from HCLOUD_SERVER_ID env
	// This env would be set explicitly by the user
	// If this is set and location can not be found we do not want a fallback
	isSet, location, err := getLocationByEnvID(logger, hcloudClient)
	if isSet {
		return location, err
	}

	// Option 2: Get from node name and search server list
	// This env is set by default via a fieldRef on spec.nodeName
	// If this is set and server can not be found we fallback to the metadata fallback
	location, err = getLocationByEnvNodeName(logger, hcloudClient)
	if err != nil {
		return "", err
	}
	if location != "" {
		return location, nil
	}

	// Option 3: Metadata service as fallback
	return getLocationFromMetadata(logger, metadataClient)
}

func getLocationByEnvID(logger *slog.Logger, hcloudClient *hcloud.Client) (bool, string, error) {
	envID := os.Getenv("HCLOUD_SERVER_ID")
	if envID == "" {
		return false, "", nil
	}

	id, err := strconv.ParseInt(envID, 10, 64)
	if err != nil {
		return true, "", fmt.Errorf("invalid server id in HCLOUD_SERVER_ID env var: %s", err)
	}

	logger.Debug(
		"using server id from HCLOUD_SERVER_ID env var",
		"server-id", id,
	)

	server, _, err := hcloudClient.Server.GetByID(context.Background(), id)
	if err != nil {
		return true, "", err
	}
	if server == nil {
		return true, "", fmt.Errorf("HCLOUD_SERVER_ID is set to %d, but no server could be found", id)
	}

	return true, server.Datacenter.Location.Name, nil
}

func getLocationByEnvNodeName(logger *slog.Logger, hcloudClient *hcloud.Client) (string, error) {
	nodeName := os.Getenv("KUBE_NODE_NAME")
	if nodeName == "" {
		return "", nil
	}

	server, _, err := hcloudClient.Server.GetByName(context.Background(), nodeName)
	if err != nil {
		return "", fmt.Errorf("error while getting server through node name: %s", err)
	}
	if server != nil {
		logger.Debug(
			"fetched server via server name from KUBE_NODE_NAME env var",
			"server-id", server.ID,
		)
		return server.Datacenter.Location.Name, nil
	}

	logger.Info(
		"KUBE_NODE_NAME is set, but no server could be found",
		"KUBE_NODE_NAME", nodeName,
	)

	return "", nil
}

func getLocationFromMetadata(logger *slog.Logger, metadataClient *metadata.Client) (string, error) {
	logger.Debug("getting location from metadata service")
	availabilityZone, err := metadataClient.AvailabilityZone()
	if err != nil {
		return "", fmt.Errorf("failed to get location from metadata service: %s", err)
	}

	parts := strings.Split(availabilityZone, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("availability zone from metadata service is not in the correct format, got: %s", availabilityZone)
	}

	return parts[0], nil
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
