package volsrv

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hetznercloud/csi-driver/internal/csi"
	"github.com/hetznercloud/csi-driver/internal/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutil"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

var _ volumes.Service = (*VolumeService)(nil)

func makeTestVolumeService(t *testing.T, requests []mockutil.Request) (*VolumeService, func()) {
	t.Helper()

	testServer := httptest.NewServer(mockutil.Handler(t, requests))

	testClient := hcloud.NewClient(
		hcloud.WithEndpoint(testServer.URL),
		hcloud.WithRetryOpts(hcloud.RetryOpts{BackoffFunc: hcloud.ConstantBackoff(0), MaxRetries: 3}),
		hcloud.WithPollOpts(hcloud.PollOpts{BackoffFunc: hcloud.ConstantBackoff(0)}),
	)

	volumeService := NewVolumeService(slog.New(slog.DiscardHandler), testClient)

	return volumeService, testServer.Close
}

func TestResize(t *testing.T) {
	t.Run("ErrVolumeSizeAlreadyReached", func(t *testing.T) {
		t.Run("happy with larger volume size", func(t *testing.T) {
			volumeService, cleanup := makeTestVolumeService(t, []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON: schema.VolumeGetResponse{
						Volume: schema.Volume{ID: 1, Name: "pvc-123", Size: 10},
					},
				},
				{
					Method: "POST", Path: "/volumes/1/actions/resize",
					Status: 201,
					JSON: schema.VolumeActionResizeVolumeResponse{
						Action: schema.Action{ID: 3, Status: "success"},
					},
				},
			})
			defer cleanup()

			err := volumeService.Resize(context.Background(), &csi.Volume{ID: 1}, 15)
			assert.NoError(t, err)
		})

		t.Run("with equal volume size", func(t *testing.T) {
			volumeService, cleanup := makeTestVolumeService(t, []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON: schema.VolumeGetResponse{
						Volume: schema.Volume{ID: 1, Name: "pvc-123", Size: 15},
					},
				},
			})
			defer cleanup()

			err := volumeService.Resize(context.Background(), &csi.Volume{ID: 1}, 15)
			assert.Equal(t, volumes.ErrVolumeSizeAlreadyReached, err)
		})

		t.Run("with smaller volume size", func(t *testing.T) {
			volumeService, cleanup := makeTestVolumeService(t, []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON: schema.VolumeGetResponse{
						Volume: schema.Volume{ID: 1, Name: "pvc-123", Size: 15},
					},
				},
			})
			defer cleanup()

			err := volumeService.Resize(context.Background(), &csi.Volume{ID: 1}, 10)
			assert.Equal(t, volumes.ErrVolumeSizeAlreadyReached, err)
		})
	})
}

func TestAll(t *testing.T) {
	t.Run("with volumes", func(t *testing.T) {
		volumeService, cleanup := makeTestVolumeService(t, []mockutil.Request{
			{
				Method: "GET", Path: "/volumes?page=1&per_page=50",
				Status: 200,
				JSON: schema.VolumeListResponse{
					Volumes: []schema.Volume{
						{
							ID: 1, Name: "pvc-123", Size: 10,
							Location:    schema.Location{Name: "hel1"},
							LinuxDevice: "/dev/disk/by-id/scsi-0HC_Volume_1",
							Server:      new(int64(42)),
						},
						{
							ID: 2, Name: "pvc-456", Size: 20,
							Location: schema.Location{Name: "fsn1"},
						},
					},
				},
			},
		})
		defer cleanup()

		vols, err := volumeService.All(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []*csi.Volume{
			{
				ID: 1, Name: "pvc-123", Size: 10,
				Location:    "hel1",
				LinuxDevice: "/dev/disk/by-id/scsi-0HC_Volume_1",
				Server:      &csi.Server{ID: 42},
			},
			{
				ID: 2, Name: "pvc-456", Size: 20,
				Location: "fsn1",
			},
		}, vols)
	})

	t.Run("without volumes", func(t *testing.T) {
		volumeService, cleanup := makeTestVolumeService(t, []mockutil.Request{
			{
				Method: "GET", Path: "/volumes?page=1&per_page=50",
				Status: 200,
				JSON:   schema.VolumeListResponse{},
			},
		})
		defer cleanup()

		vols, err := volumeService.All(context.Background())
		require.NoError(t, err)
		assert.Empty(t, vols)
	})
}
