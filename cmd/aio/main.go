package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	proto "github.com/container-storage-interface/spec/lib/go/csi"

	"github.com/hetznercloud/csi-driver/internal/api"
	"github.com/hetznercloud/csi-driver/internal/app"
	"github.com/hetznercloud/csi-driver/internal/driver"
	"github.com/hetznercloud/csi-driver/internal/utils"
	"github.com/hetznercloud/csi-driver/internal/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
)

var logger *slog.Logger

func main() {
	logger = app.CreateLogger()

	m := app.CreateMetrics(logger)

	hcloudClient, err := app.CreateHcloudClient(m.Registry(), logger)
	if err != nil {
		logger.Error(
			"failed to initialize hcloud client",
			"err", err,
		)
		os.Exit(1)
	}

	metadataClient := metadata.NewClient(metadata.WithInstrumentation(m.Registry()))

	if !metadataClient.IsHcloudServer() {
		logger.Warn("unable to connect to metadata service, are you sure this is running on a Hetzner Cloud server?")
	}

	// node
	serverID, err := metadataClient.InstanceID()
	if err != nil {
		logger.Error("failed to fetch server ID from metadata service", "err", err)
		os.Exit(1)
	}

	serverAZ, err := metadataClient.AvailabilityZone()
	if err != nil {
		logger.Error("failed to fetch server availability-zone from metadata service", "err", err)
		os.Exit(1)
	}
	parts := strings.Split(serverAZ, "-")
	if len(parts) != 2 {
		logger.Error(fmt.Sprintf("unexpected server availability zone: %s", serverAZ), "err", err)
		os.Exit(1)
	}
	serverLocation := parts[0]

	logger.Info("Fetched data from metadata service", "id", serverID, "location", serverLocation)

	volumeMountService := volumes.NewLinuxMountService(logger.With("component", "linux-mount-service"))
	volumeResizeService := volumes.NewLinuxResizeService(logger.With("component", "linux-resize-service"))
	volumeStatsService := volumes.NewLinuxStatsService(logger.With("component", "linux-stats-service"))

	enableProvidedByTopology := app.GetEnableProvidedByTopology()

	nodeService := driver.NewNodeService(
		logger.With("component", "driver-node-service"),
		strconv.FormatInt(serverID, 10),
		serverLocation,
		enableProvidedByTopology,
		volumeMountService,
		volumeResizeService,
		volumeStatsService,
	)

	// controller
	volumeService := volumes.NewIdempotentService(
		logger.With("component", "idempotent-volume-service"),
		api.NewVolumeService(
			logger.With("component", "api-volume-service"),
			hcloudClient,
		),
	)

	extraVolumeLabels, err := utils.ConvertLabelsToMap(os.Getenv("HCLOUD_VOLUME_EXTRA_LABELS"))
	if err != nil {
		logger.Error("could not parse extra labels for volumes", "error", err)
		os.Exit(1)
	}
	controllerService := driver.NewControllerService(
		logger.With("component", "driver-controller-service"),
		volumeService,
		serverLocation,
		enableProvidedByTopology,
		extraVolumeLabels,
	)

	// common
	identityService := driver.NewIdentityService(
		logger.With("component", "driver-identity-service"),
	)

	// common
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

	// controller
	proto.RegisterControllerServer(grpcServer, controllerService)
	// common
	proto.RegisterIdentityServer(grpcServer, identityService)
	// node
	proto.RegisterNodeServer(grpcServer, nodeService)

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
