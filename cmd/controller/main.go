package main

import (
	"log/slog"
	"os"

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
	app.SetupCoverageSignalHandler(logger)

	m := app.CreateMetrics(logger)

	hcloudClient, err := app.CreateHcloudClient(m.Registry(), logger)
	if err != nil {
		logger.Error(
			"failed to initialize hcloud client",
			"err", err,
		)
		os.Exit(1)
	}

	var location string
	if s := os.Getenv("HCLOUD_VOLUME_DEFAULT_LOCATION"); s != "" {
		location = s
	} else {
		metadataClient := metadata.NewClient(metadata.WithInstrumentation(m.Registry()))

		if !metadataClient.IsHcloudServer() {
			logger.Warn("Unable to connect to metadata service. " +
				"In the current configuration the controller is required to run on a Hetzner Cloud server. " +
				"You can set HCLOUD_VOLUME_DEFAULT_LOCATION if you want to run it somewhere else.")
		}

		location, err = app.GetServerLocation(logger, hcloudClient, metadataClient)
		if err != nil {
			logger.Error(
				"failed to fetch server",
				"err", err,
			)
			os.Exit(1)
		}
	}

	logger.Debug(
		"evaluated default location for volumes",
		"location", location,
	)

	if location == "" {
		logger.Error("could not set a default location for volumes")
		os.Exit(1)
	}

	enableProvidedByTopology := app.GetEnableProvidedByTopology()

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
		location,
		enableProvidedByTopology,
		extraVolumeLabels,
	)

	identityService := driver.NewIdentityService(
		logger.With("component", "driver-identity-service"),
	)

	listener, err := app.CreateListener()
	if err != nil {
		logger.Error(
			"failed to create listener",
			"err", err,
		)
		os.Exit(1)
	}

	grpcServer := app.CreateGRPCServer(logger, m.UnaryServerInterceptor())

	proto.RegisterControllerServer(grpcServer, controllerService)
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
