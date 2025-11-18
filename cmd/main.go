package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"

	"github.com/hetznercloud/csi-driver/internal/api"
	"github.com/hetznercloud/csi-driver/internal/app"
	"github.com/hetznercloud/csi-driver/internal/driver"
	"github.com/hetznercloud/csi-driver/internal/metrics"
	"github.com/hetznercloud/csi-driver/internal/utils"
	"github.com/hetznercloud/csi-driver/internal/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
)

func main() {
	var controller, node bool

	logger := app.CreateLogger()

	flag.BoolVar(
		&controller,
		"controller",
		false,
		"Run the csi controller. Can be used with `-node` to run as an AIO binary (e.g., for docker swarm).",
	)

	flag.BoolVar(
		&node,
		"node",
		false,
		"Run the csi node driver. Can be used with `-controller` to run as an AIO binary (e.g., for docker swarm).",
	)
	flag.Parse()

	if !controller && !node {
		logger.Error("neither -controller nor -node was specified")
		flag.PrintDefaults()
		os.Exit(1)
	}

	app.SetupCoverageSignalHandler(logger)

	m := app.CreateMetrics(logger)

	metadataClient := metadata.NewClient(metadata.WithInstrumentation(m.Registry()))

	listener, err := app.CreateListener()
	if err != nil {
		logger.Error("failed to create listener", "error", err)
		os.Exit(1)
	}

	grpcServer := app.CreateGRPCServer(
		logger.With("component", "grpc-server"),
		m.UnaryServerInterceptor(),
	)

	if err := setup(logger, controller, node, grpcServer, m, metadataClient); err != nil {
		logger.Error("failed to setup CSI driver", "error", err)
		os.Exit(1)
	}

	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("failed to run CSI driver", "error", err)
		os.Exit(1)
	}
}

func setup(
	logger *slog.Logger,
	controller, node bool,
	grpcServer *grpc.Server,
	m *metrics.Metrics,
	metadataClient *metadata.Client,
) error {
	enableProvidedByTopology := app.GetEnableProvidedByTopology()

	if !metadataClient.IsHcloudServer() {
		logger.Warn("unable to connect to the metadata service")
	}

	if node {
		location, err := app.GetServerLocation(logger, metadataClient, nil, false)
		if err != nil {
			return fmt.Errorf("could not determine default volume location: %w", err)
		}

		serverID, err := metadataClient.InstanceID()
		if err != nil {
			return fmt.Errorf("failed to fetch server ID from metadata service: %w", err)
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
		hcloudClient, err := app.CreateHcloudClient(m.Registry(), logger)
		if err != nil {
			return fmt.Errorf("failed to initialize hcloud client: %w", err)
		}

		location, err := app.GetServerLocation(logger, metadataClient, hcloudClient, true)
		if err != nil {
			return fmt.Errorf("could not determine default volume location: %w", err)
		}

		extraVolumeLabels, err := utils.ConvertLabelsToMap(os.Getenv("HCLOUD_VOLUME_EXTRA_LABELS"))
		if err != nil {
			return fmt.Errorf("could not parse extra labels for volumes: %w", err)
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

	return nil
}
