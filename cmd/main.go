package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	proto "github.com/container-storage-interface/spec/lib/go/csi"

	"github.com/hetznercloud/csi-driver/internal/api"
	"github.com/hetznercloud/csi-driver/internal/app"
	"github.com/hetznercloud/csi-driver/internal/driver"
	"github.com/hetznercloud/csi-driver/internal/utils"
	"github.com/hetznercloud/csi-driver/internal/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
)

var logger *slog.Logger
var controller, node bool

func init() {
	flag.BoolVar(&controller, "controller", false, "Run the csi controller. Can be used with `-node` to run as an AIO binary (e.g., for docker swarm).")
	flag.BoolVar(&node, "node", false, "Run the csi node driver. Can be used with `-controller` to run as an AIO binary (e.g., for docker swarm).")
	flag.Parse()
}

func main() {
	logger = app.CreateLogger()

	if !controller && !node {
		logger.Error("application must be started with -controller and/or -node.")
		os.Exit(1)
	}

	app.SetupCoverageSignalHandler(logger)

	m := app.CreateMetrics(logger)

	metadataClient := metadata.NewClient(metadata.WithInstrumentation(m.Registry()))

	if !metadataClient.IsHcloudServer() {
		logger.Warn("unable to connect to the metadata service")
	}

	location, err := getLocation(logger, metadataClient)
	if err != nil {
		logger.Error("could not determine default volume location", "error", err)
		os.Exit(1)
	}

	enableProvidedByTopology := app.GetEnableProvidedByTopology()

	listener, err := app.CreateListener()
	if err != nil {
		logger.Error(
			"failed to create listener",
			"err", err,
		)
		os.Exit(1)
	}

	grpcServer := app.CreateGRPCServer(
		logger.With("component", "grpc-server"),
		m.UnaryServerInterceptor(),
	)

	if node {
		serverID, err := metadataClient.InstanceID()
		if err != nil {
			logger.Error("failed to fetch server ID from metadata service", "err", err)
			os.Exit(1)
		}

		volumeMountService := volumes.NewLinuxMountService(logger.With("component", "linux-mount-service"))
		volumeResizeService := volumes.NewLinuxResizeService(logger.With("component", "linux-resize-service"))
		volumeStatsService := volumes.NewLinuxStatsService(logger.With("component", "linux-stats-service"))

		nodeService := driver.NewNodeService(
			logger.With("component", "driver-node-service"),
			strconv.FormatInt(serverID, 10),
			location,
			enableProvidedByTopology,
			volumeMountService,
			volumeResizeService,
			volumeStatsService,
		)

		proto.RegisterNodeServer(grpcServer, nodeService)
	}

	if controller {
		extraVolumeLabels, err := utils.ConvertLabelsToMap(os.Getenv("HCLOUD_VOLUME_EXTRA_LABELS"))
		if err != nil {
			logger.Error("could not parse extra labels for volumes", "error", err)
			os.Exit(1)
		}

		hcloudClient, err := app.CreateHcloudClient(m.Registry(), logger)
		if err != nil {
			logger.Error(
				"failed to initialize hcloud client",
				"err", err,
			)
			os.Exit(1)
		}

		volumeService := volumes.NewIdempotentService(
			logger.With("component", "idempotent-volume-service"),
			api.NewVolumeService(
				logger.With("component", "api-volume-service"),
				hcloudClient,
			),
		)

		controllerService := driver.NewControllerService(
			logger.With("component", "driver-controller-service"),
			volumeService,
			location,
			enableProvidedByTopology,
			extraVolumeLabels,
		)

		proto.RegisterControllerServer(grpcServer, controllerService)
	}

	identityService := driver.NewIdentityService(
		logger.With("component", "driver-identity-service"),
	)

	proto.RegisterIdentityServer(grpcServer, identityService)

	m.InitializeMetrics(grpcServer)

	identityService.SetReady(true)

	if err := grpcServer.Serve(listener); err != nil {
		logger.Error(
			"grpc server failed",
			"err", err,
		)
		os.Exit(1)
	}
}

func getLocation(logger *slog.Logger, metadataClient *metadata.Client) (string, error) {
	if location, ok := os.LookupEnv("HCLOUD_VOLUME_DEFAULT_LOCATION"); ok {
		return location, nil
	}

	if !metadataClient.IsHcloudServer() {
		return "", errors.New("HCLOUD_VOLUME_DEFAULT_LOCATION not set and not running on a cloud server")
	}

	location, err := app.GetLocationFromMetadata(logger, metadataClient)
	if err != nil {
		return "", fmt.Errorf("failed to get location from metadata: %w", err)
	}

	return location, nil
}
