package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/csi-driver/internal/app"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/kit/randutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/metadata"
)

func TestGetServerLocation(t *testing.T) {
	token := os.Getenv("HCLOUD_TOKEN")
	require.NotEmpty(t, token, "HCLOUD_TOKEN is not set")

	client := hcloud.NewClient(hcloud.WithToken(token))

	metadataService := metadataServiceMux()
	t.Cleanup(metadataService.Close)

	metadataClient := metadata.NewClient(
		metadata.WithHTTPClient(metadataService.Client()),
		metadata.WithEndpoint(fmt.Sprintf("%s/hetzner/v1/metadata", metadataService.URL)),
	)

	serverName := fmt.Sprintf("csi-integration-%s", randutil.GenerateID())

	result, _, err := client.Server.Create(t.Context(), hcloud.ServerCreateOpts{
		Name:       serverName,
		ServerType: &hcloud.ServerType{Name: "cpx22"},
		Location:   &hcloud.Location{Name: "hel1"},
		Image:      &hcloud.Image{Name: "ubuntu-24.04"},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx := context.Background() // t.Context() is already canceled at this point
		delResult, _, err := client.Server.DeleteWithResult(ctx, result.Server)
		require.NoError(t, err)
		require.NoError(t, client.Action.WaitFor(ctx, delResult.Action))
	})

	require.NoError(t, client.Action.WaitFor(t.Context(), result.Action))

	t.Run("location env", func(t *testing.T) {
		t.Setenv("HCLOUD_VOLUME_DEFAULT_LOCATION", "hel1")
		loc, err := app.GetServerLocation(slog.New(slog.DiscardHandler), metadataClient, client, false)
		require.NoError(t, err)
		assert.Equal(t, "hel1", loc)
	})

	t.Run("server ID env", func(t *testing.T) {
		t.Setenv("HCLOUD_SERVER_ID", strconv.FormatInt(result.Server.ID, 10))
		loc, err := app.GetServerLocation(slog.New(slog.DiscardHandler), metadataClient, client, true)
		require.NoError(t, err)
		assert.Equal(t, "hel1", loc)
	})

	t.Run("node name env", func(t *testing.T) {
		t.Setenv("KUBE_NODE_NAME", serverName)
		loc, err := app.GetServerLocation(slog.New(slog.DiscardHandler), metadataClient, client, true)
		require.NoError(t, err)
		assert.Equal(t, "hel1", loc)
	})

	t.Run("metadata service", func(t *testing.T) {
		loc, err := app.GetServerLocation(slog.New(slog.DiscardHandler), metadataClient, client, false)
		require.NoError(t, err)
		assert.Equal(t, "hel1", loc)
	})
}

func metadataServiceMux() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/hetzner/v1/metadata/availability-zone", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "hel1-dc2")
	})

	return httptest.NewServer(mux)
}
