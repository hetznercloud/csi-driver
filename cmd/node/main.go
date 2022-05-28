package main

import (
	"context"
	"os"
	"strconv"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hetznercloud/csi-driver/app"
	"github.com/hetznercloud/csi-driver/driver"
	"github.com/hetznercloud/csi-driver/volumes"
	"github.com/hetznercloud/hcloud-go/hcloud/metadata"
)

var logger log.Logger

func main() {
	logger = app.CreateLogger()

	m := app.CreateMetrics(logger)

	metadataClient := metadata.NewClient(metadata.WithInstrumentation(m.Registry()))

	serverID, err := metadataClient.InstanceID()
	if err != nil {
		level.Error(logger).Log("msg", "failed to fetch server ID from metadata service", "err", err)
		os.Exit(1)
	}

	hcloudClient, err := app.CreateHcloudClient(m.Registry(), logger)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize hcloud client", "err", err)
		os.Exit(1)
	}

	server, _, err := hcloudClient.Server.GetByID(context.Background(), serverID)
	if err != nil {
		level.Error(logger).Log("msg", "failed to fetch server from API", "err", err)
	}
	serverLocation := server.Datacenter.Location.Name

	level.Info(logger).Log("msg", "Fetched data from metadata service", "id", serverID, "location", serverLocation)

	volumeMountService := volumes.NewLinuxMountService(
		log.With(logger, "component", "linux-mount-service"),
	)
	volumeResizeService := volumes.NewLinuxResizeService(
		log.With(logger, "component", "linux-resize-service"),
	)
	volumeStatsService := volumes.NewLinuxStatsService(
		log.With(logger, "component", "linux-stats-service"),
	)
	identityService := driver.NewIdentityService(
		log.With(logger, "component", "driver-identity-service"),
	)
	nodeService := driver.NewNodeService(
		log.With(logger, "component", "driver-node-service"),
		strconv.Itoa(serverID),
		serverLocation,
		volumeMountService,
		volumeResizeService,
		volumeStatsService,
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

	proto.RegisterIdentityServer(grpcServer, identityService)
	proto.RegisterNodeServer(grpcServer, nodeService)

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
