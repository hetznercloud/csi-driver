package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hetznercloud/csi-driver/api"
	"github.com/hetznercloud/csi-driver/app"
	"github.com/hetznercloud/csi-driver/driver"
	"github.com/hetznercloud/csi-driver/volumes"
	"github.com/hetznercloud/hcloud-go/hcloud/metadata"
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

	metadataClient := metadata.NewClient(metadata.WithInstrumentation(m.Registry()))

	server, err := app.GetServer(logger, hcloudClient, metadataClient)
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to fetch server",
			"err", err,
		)
		os.Exit(1)
	}

	// node
	serverID, err := metadataClient.InstanceID()
	if err != nil {
		level.Error(logger).Log("msg", "failed to fetch server ID from metadata service", "err", err)
		os.Exit(1)
	}

	serverAZ, err := metadataClient.AvailabilityZone()
	if err != nil {
		level.Error(logger).Log("msg", "failed to fetch server availability-zone from metadata service", "err", err)
		os.Exit(1)
	}
	parts := strings.Split(serverAZ, "-")
	if len(parts) != 2 {
		level.Error(logger).Log("msg", fmt.Sprintf("unexpected server availability zone: %s", serverAZ), "err", err)
		os.Exit(1)
	}
	serverLocation := parts[0]

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
	nodeService := driver.NewNodeService(
		log.With(logger, "component", "driver-node-service"),
		strconv.Itoa(serverID),
		serverLocation,
		volumeMountService,
		volumeResizeService,
		volumeStatsService,
	)

	// controller
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
		server.Datacenter.Location.Name,
	)

	// common
	identityService := driver.NewIdentityService(
		log.With(logger, "component", "driver-identity-service"),
	)

	// common
	listener, err := app.CreateListener()
	if err != nil {
		level.Error(logger).Log(
			"msg", "failed to create listener",
			"err", err,
		)
		os.Exit(1)
	}

	grpcServer := app.CreateGRPCServer(logger, m.UnaryServerInterceptor())

	// controller
	proto.RegisterControllerServer(grpcServer, controllerService)
	// common
	proto.RegisterIdentityServer(grpcServer, identityService)
	// node
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
