package api

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"

	"github.com/hetznercloud/csi-driver/internal/csi"
	"github.com/hetznercloud/csi-driver/internal/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/mockutils"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
)

var _ volumes.Service = (*VolumeService)(nil)

func makeTestVolumeService(t *testing.T, requests []mockutils.Request) (*VolumeService, func()) {
	t.Helper()

	testServer := httptest.NewServer(mockutils.Handler(t, requests))

	testClient := hcloud.NewClient(
		hcloud.WithEndpoint(testServer.URL),
		hcloud.WithBackoffFunc(func(_ int) time.Duration { return 0 }),
		hcloud.WithPollBackoffFunc(func(_ int) time.Duration { return 0 }),
	)

	volumeService := NewVolumeService(log.NewNopLogger(), testClient)

	return volumeService, testServer.Close
}

func TestResize(t *testing.T) {
	t.Run("ErrVolumeSizeAlreadyReached", func(t *testing.T) {
		t.Run("happy with larger volume size", func(t *testing.T) {
			volumeService, cleanup := makeTestVolumeService(t, []mockutils.Request{
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
			volumeService, cleanup := makeTestVolumeService(t, []mockutils.Request{
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
			volumeService, cleanup := makeTestVolumeService(t, []mockutils.Request{
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
