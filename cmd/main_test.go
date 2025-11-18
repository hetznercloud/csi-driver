package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/csi-driver/internal/app"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
)

var (
	MetaHostnameRequest = mockutil.Request{
		Method: http.MethodGet, Path: "/hostname",
		Status:  http.StatusOK,
		TextRaw: "foobar",
	}
	MetaAvailabilityZoneRequest = mockutil.Request{
		Method: http.MethodGet, Path: "/availability-zone",
		Status:  http.StatusOK,
		TextRaw: "hel1-dc2",
	}
	MetaInstanceIDRequest = mockutil.Request{
		Method: http.MethodGet, Path: "/instance-id",
		Status:  http.StatusOK,
		TextRaw: "42",
	}
)

func TestSetup(t *testing.T) {
	t.Setenv("ENABLE_METRICS", "false")

	logger := slog.New(slog.DiscardHandler)

	m := app.CreateMetrics(logger)

	t.Run("missing hcloud token", func(t *testing.T) {
		grpcServer := app.CreateGRPCServer(
			logger.With("component", "grpc-server"),
			m.UnaryServerInterceptor(),
		)

		metaServer := mockutil.NewServer(t, []mockutil.Request{
			MetaHostnameRequest,
		})
		metaClient := metadata.NewClient(metadata.WithEndpoint(metaServer.URL))

		t.Setenv("CSI_ENDPOINT", fmt.Sprintf("unix:///%s/csi.sock", t.TempDir()))

		err := setup(logger, true, false, grpcServer, m, metaClient)
		require.EqualError(t, err, "failed to initialize hcloud client: you need to provide an API token via the HCLOUD_TOKEN or HCLOUD_TOKEN_FILE env var")
	})

	t.Run("controller", func(t *testing.T) {
		grpcServer := app.CreateGRPCServer(
			logger.With("component", "grpc-server"),
			m.UnaryServerInterceptor(),
		)

		metaServer := mockutil.NewServer(t, []mockutil.Request{
			MetaHostnameRequest,
			MetaAvailabilityZoneRequest,
		})
		metaClient := metadata.NewClient(metadata.WithEndpoint(metaServer.URL))

		t.Setenv("CSI_ENDPOINT", fmt.Sprintf("unix:///%s/csi.sock", t.TempDir()))
		t.Setenv("HCLOUD_TOKEN", "foobar")

		err := setup(logger, true, false, grpcServer, m, metaClient)
		require.NoError(t, err)
	})

	t.Run("node", func(t *testing.T) {
		grpcServer := app.CreateGRPCServer(
			logger.With("component", "grpc-server"),
			m.UnaryServerInterceptor(),
		)

		metaServer := mockutil.NewServer(t, []mockutil.Request{
			MetaHostnameRequest,
			MetaAvailabilityZoneRequest,
			MetaInstanceIDRequest,
		})
		metaClient := metadata.NewClient(metadata.WithEndpoint(metaServer.URL))

		err := setup(logger, false, true, grpcServer, m, metaClient)
		require.NoError(t, err)
	})

	t.Run("controller and node", func(t *testing.T) {
		grpcServer := app.CreateGRPCServer(
			logger.With("component", "grpc-server"),
			m.UnaryServerInterceptor(),
		)

		metaServer := mockutil.NewServer(t, []mockutil.Request{
			MetaHostnameRequest,
			MetaAvailabilityZoneRequest,
			MetaInstanceIDRequest,
			MetaAvailabilityZoneRequest,
		})
		metaClient := metadata.NewClient(metadata.WithEndpoint(metaServer.URL))

		t.Setenv("CSI_ENDPOINT", fmt.Sprintf("unix:///%s/csi.sock", t.TempDir()))
		t.Setenv("HCLOUD_TOKEN", "foobar")

		err := setup(logger, true, true, grpcServer, m, metaClient)
		require.NoError(t, err)
	})
}
