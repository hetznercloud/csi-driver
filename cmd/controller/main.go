package main

import (
	"os"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hetznercloud/csi-driver/api"
	"github.com/hetznercloud/csi-driver/app"
	"github.com/hetznercloud/csi-driver/driver"
	"github.com/hetznercloud/csi-driver/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
)

var logger log.Logger

func main() {
	logger = app.CreateLogger()

	m := app.CreateMetrics(logger)

	hcloudClient, err := app.CreateHcloudClient(m.Registry(), logger)
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to initialize hcloud client",
			"err", err,
		)
		os.Exit(1)
	}

	var location string

	if s := os.Getenv("HCLOUD_VOLUME_DEFAULT_LOCATION"); s != "" {
		location = s
	} else {
		metadataClient := metadata.NewClient(metadata.WithInstrumentation(m.Registry()))

		server, err := app.GetServer(logger, hcloudClient, metadataClient)
		if err != nil {
			level.Error(logger).Log(
				"msg", "failed to fetch server",
				"err", err,
			)
			os.Exit(1)
		}

		location = server.Datacenter.Location.Name
	}

	volumeService := volumes.NewIdempotentService(
		log.With(logger, "component", "idempotent-volume-service"),
		api.NewVolumeService(
			log.With(logger, "component", "api-volume-service"),
			hcloudClient,
		),
	)
	controllerService := driver.NewControllerService(
		log.With(logger, "component", "driver-controller-service"),
		volumeService,
		location,
	)
	identityService := driver.NewIdentityService(
		log.With(logger, "component", "driver-identity-service"),
	)

	listener, err := app.CreateListener()
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to create listener",
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
		level.Error(logger).Log(
			"msg", "grpc server failed",
			"err", err,
		)
		os.Exit(1)
	}
}
