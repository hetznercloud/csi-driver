package volsrv

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	t.Run("larger volume size", func(t *testing.T) {
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

		err := volumeService.Resize(context.Background(), &hcloud.Volume{ID: 1}, 15)
		assert.NoError(t, err)
	})

	t.Run("equal volume size", func(t *testing.T) {
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

		err := volumeService.Resize(context.Background(), &hcloud.Volume{ID: 1}, 15)
		assert.NoError(t, err)
	})

	t.Run("smaller volume size", func(t *testing.T) {
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

		err := volumeService.Resize(context.Background(), &hcloud.Volume{ID: 1}, 10)
		assert.NoError(t, err)
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
							Assignee:    serverAssignee(42),
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
		assert.Equal(t, []*hcloud.Volume{
			{
				ID: 1, Name: "pvc-123", Size: 10,
				Location:    &hcloud.Location{Name: "hel1"},
				LinuxDevice: "/dev/disk/by-id/scsi-0HC_Volume_1",
				Assignee:    &hcloud.VolumeAssignee{Type: "server", ID: 42},
				Server:      &hcloud.Server{ID: 42}, //nolint:staticcheck
			},
			{
				ID: 2, Name: "pvc-456", Size: 20,
				Location: &hcloud.Location{Name: "fsn1"},
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

func serverAssignee(id int64) *schema.VolumeAssignee {
	return &schema.VolumeAssignee{Type: "server", ID: id}
}

func errorResponse(code hcloud.ErrorCode) schema.ErrorResponse {
	return schema.ErrorResponse{
		Error: schema.Error{Code: string(code), Message: string(code)},
	}
}

func TestCreate(t *testing.T) {
	opts := volumes.CreateOpts{
		Name:     "pvc-123",
		MinSize:  10,
		MaxSize:  20,
		Location: "fsn1",
	}

	t.Run("new volume", func(t *testing.T) {
		volumeService, cleanup := makeTestVolumeService(t, []mockutil.Request{
			{
				Method: "POST", Path: "/volumes",
				Status: 201,
				JSON: schema.VolumeCreateResponse{
					Volume: schema.Volume{ID: 1, Name: "pvc-123", Size: 10, Location: schema.Location{Name: "fsn1"}},
					Action: &schema.Action{ID: 2, Status: "success"},
				},
			},
		})
		defer cleanup()

		volume, err := volumeService.Create(context.Background(), opts)
		require.NoError(t, err)
		assert.Equal(t, int64(1), volume.ID)
	})

	testCases := []struct {
		name     string
		existing []schema.Volume
		wantErr  error
	}{
		{
			name:     "existing volume",
			existing: []schema.Volume{{ID: 1, Name: "pvc-123", Size: 10, Location: schema.Location{Name: "fsn1"}}},
		},
		{
			name:     "existing volume too small",
			existing: []schema.Volume{{ID: 1, Name: "pvc-123", Size: 5, Location: schema.Location{Name: "fsn1"}}},
			wantErr:  volumes.ErrVolumeAlreadyExists,
		},
		{
			name:     "existing volume too large",
			existing: []schema.Volume{{ID: 1, Name: "pvc-123", Size: 30, Location: schema.Location{Name: "fsn1"}}},
			wantErr:  volumes.ErrVolumeAlreadyExists,
		},
		{
			name:     "existing volume in different location",
			existing: []schema.Volume{{ID: 1, Name: "pvc-123", Size: 10, Location: schema.Location{Name: "nbg1"}}},
			wantErr:  volumes.ErrVolumeAlreadyExists,
		},
		{
			name:    "existing volume deleted concurrently",
			wantErr: volumes.ErrVolumeNotFound,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			volumeService, cleanup := makeTestVolumeService(t, []mockutil.Request{
				{
					Method: "POST", Path: "/volumes",
					Status: 409,
					JSON:   errorResponse(hcloud.ErrorCodeUniquenessError),
				},
				{
					Method: "GET", Path: "/volumes?name=pvc-123",
					Status: 200,
					JSON:   schema.VolumeListResponse{Volumes: tt.existing},
				},
			})
			defer cleanup()

			volume, err := volumeService.Create(context.Background(), opts)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int64(1), volume.ID)
		})
	}
}

func TestDelete(t *testing.T) {
	testCases := []struct {
		name     string
		requests []mockutil.Request
		wantErr  error
	}{
		{
			name: "detached volume",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1}},
				},
				{
					Method: "DELETE", Path: "/volumes/1",
					Status: 204,
				},
			},
		},
		{
			name: "attached volume",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: serverAssignee(42)}},
				},
				{
					Method: "POST", Path: "/volumes/1/actions/detach",
					Status: 201,
					JSON:   schema.VolumeActionDetachVolumeResponse{Action: schema.Action{ID: 2, Status: "success"}},
				},
				{
					Method: "DELETE", Path: "/volumes/1",
					Status: 204,
				},
			},
		},
		{
			name: "volume not found",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 404,
					JSON:   errorResponse(hcloud.ErrorCodeNotFound),
				},
			},
		},
		{
			name: "volume deleted concurrently",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1}},
				},
				{
					Method: "DELETE", Path: "/volumes/1",
					Status: 404,
					JSON:   errorResponse(hcloud.ErrorCodeNotFound),
				},
			},
		},
		{
			name: "detach fails",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: serverAssignee(42)}},
				},
				{
					Method: "POST", Path: "/volumes/1/actions/detach",
					Status: 423,
					JSON:   errorResponse(hcloud.ErrorCodeLocked),
				},
			},
			wantErr: volumes.ErrLockedServer,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			volumeService, cleanup := makeTestVolumeService(t, tt.requests)
			defer cleanup()

			err := volumeService.Delete(context.Background(), &hcloud.Volume{ID: 1})
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAttach(t *testing.T) {
	getServer := mockutil.Request{
		Method: "GET", Path: "/servers/42",
		Status: 200,
		JSON:   schema.ServerGetResponse{Server: schema.Server{ID: 42}},
	}

	testCases := []struct {
		name     string
		requests []mockutil.Request
		wantErr  error
	}{
		{
			name: "detached volume",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1}},
				},
				getServer,
				{
					Method: "POST", Path: "/volumes/1/actions/attach",
					Status: 201,
					JSON:   schema.VolumeActionAttachVolumeResponse{Action: schema.Action{ID: 2, Status: "success"}},
				},
			},
		},
		{
			name: "already attached to server",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: serverAssignee(42)}},
				},
				getServer,
			},
		},
		{
			name: "attached to another server",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: serverAssignee(43)}},
				},
				getServer,
			},
			wantErr: volumes.ErrAttached,
		},
		{
			name: "attached to another resource type",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: &schema.VolumeAssignee{Type: "other", ID: 42}}},
				},
				getServer,
			},
			wantErr: volumes.ErrAttached,
		},
		{
			name: "attached to server concurrently",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1}},
				},
				getServer,
				{
					Method: "POST", Path: "/volumes/1/actions/attach",
					Status: 422,
					JSON:   errorResponse(hcloud.ErrorCodeVolumeAlreadyAttached),
				},
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: serverAssignee(42)}},
				},
			},
		},
		{
			name: "attached to another server concurrently",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1}},
				},
				getServer,
				{
					Method: "POST", Path: "/volumes/1/actions/attach",
					Status: 422,
					JSON:   errorResponse(hcloud.ErrorCodeVolumeAlreadyAttached),
				},
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: serverAssignee(43)}},
				},
			},
			wantErr: volumes.ErrAttached,
		},
		{
			name: "attach limit reached",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1}},
				},
				getServer,
				{
					Method: "POST", Path: "/volumes/1/actions/attach",
					Status: 422,
					JSON:   errorResponse(hcloud.ErrorCodeResourceLimitExceeded),
				},
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1}},
				},
			},
			wantErr: volumes.ErrAttachLimitReached,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			volumeService, cleanup := makeTestVolumeService(t, tt.requests)
			defer cleanup()

			err := volumeService.Attach(context.Background(), &hcloud.Volume{ID: 1}, &hcloud.Server{ID: 42})
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestDetach(t *testing.T) {
	testCases := []struct {
		name     string
		requests []mockutil.Request
		wantErr  error
	}{
		{
			name: "attached volume",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: serverAssignee(42)}},
				},
				{
					Method: "POST", Path: "/volumes/1/actions/detach",
					Status: 201,
					JSON:   schema.VolumeActionDetachVolumeResponse{Action: schema.Action{ID: 2, Status: "success"}},
				},
			},
		},
		{
			name: "detached volume",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1}},
				},
			},
		},
		{
			name: "attached to another server",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: serverAssignee(43)}},
				},
			},
		},
		{
			name: "attached to another resource type",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 200,
					JSON:   schema.VolumeGetResponse{Volume: schema.Volume{ID: 1, Assignee: &schema.VolumeAssignee{Type: "other", ID: 42}}},
				},
			},
		},
		{
			name: "volume not found",
			requests: []mockutil.Request{
				{
					Method: "GET", Path: "/volumes/1",
					Status: 404,
					JSON:   errorResponse(hcloud.ErrorCodeNotFound),
				},
			},
			wantErr: volumes.ErrVolumeNotFound,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			volumeService, cleanup := makeTestVolumeService(t, tt.requests)
			defer cleanup()

			err := volumeService.Detach(context.Background(), &hcloud.Volume{ID: 1}, &hcloud.Server{ID: 42})
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
