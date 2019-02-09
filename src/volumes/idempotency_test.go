package volumes_test

import (
	"context"
	"io"
	"testing"

	"github.com/go-kit/kit/log"

	"hetzner.cloud/csi"
	"hetzner.cloud/csi/mock"
	"hetzner.cloud/csi/volumes"
)

var _ volumes.Service = (*volumes.IdempotentService)(nil)

func TestIdempotentServiceCreateNew(t *testing.T) {
	creatingVolume := &csi.Volume{
		ID:       1,
		Name:     "vol",
		Size:     10,
		Location: "loc",
	}
	creatingOpts := volumes.CreateOpts{
		Name:     "test",
		MinSize:  10,
		MaxSize:  0,
		Location: "loc",
	}

	volumeService := &mock.VolumeService{
		CreateFunc: func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
			if opts != creatingOpts {
				t.Errorf("unexpected options: %v", opts)
			}
			return creatingVolume, nil
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	volume, err := service.Create(context.Background(), creatingOpts)
	if err != nil {
		t.Fatal(err)
	}
	if volume != creatingVolume {
		t.Error("unexpected volume")
	}
}

func TestIdempotentServiceCreateExisting(t *testing.T) {
	existingVolume := &csi.Volume{
		ID:       1,
		Name:     "vol",
		Size:     10,
		Location: "loc",
	}

	volumeService := &mock.VolumeService{
		CreateFunc: func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
			return nil, volumes.ErrVolumeAlreadyExists
		},
		GetByNameFunc: func(ctx context.Context, name string) (*csi.Volume, error) {
			return existingVolume, nil
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	volume, err := service.Create(context.Background(), volumes.CreateOpts{
		Name:     "test",
		MinSize:  10,
		MaxSize:  0,
		Location: "loc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if volume != existingVolume {
		t.Error("unexpected volume")
	}
}

func TestIdempotentServiceCreateExistingError(t *testing.T) {
	volumeService := &mock.VolumeService{
		CreateFunc: func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
			return nil, volumes.ErrVolumeAlreadyExists
		},
		GetByNameFunc: func(ctx context.Context, name string) (*csi.Volume, error) {
			return nil, io.EOF
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	_, err := service.Create(context.Background(), volumes.CreateOpts{
		Name:     "test",
		MinSize:  10,
		MaxSize:  0,
		Location: "loc",
	})
	if err != io.EOF {
		t.Fatal(err)
	}
}

func TestIdempotentServiceCreateExistingNotFitting(t *testing.T) {
	testCases := []struct {
		Name           string
		ExistingVolume *csi.Volume
	}{
		{
			Name: "too small",
			ExistingVolume: &csi.Volume{
				ID:       1,
				Name:     "vol",
				Size:     1,
				Location: "loc",
			},
		},
		{
			Name: "too large",
			ExistingVolume: &csi.Volume{
				ID:       1,
				Name:     "vol",
				Size:     1000,
				Location: "loc",
			},
		},
		{
			Name: "wrong location",
			ExistingVolume: &csi.Volume{
				ID:       1,
				Name:     "vol",
				Size:     10,
				Location: "wrong",
			},
		},
	}

	volumeService := &mock.VolumeService{
		CreateFunc: func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
			return nil, volumes.ErrVolumeAlreadyExists
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			volumeService.GetByNameFunc = func(ctx context.Context, name string) (*csi.Volume, error) {
				return testCase.ExistingVolume, nil
			}

			volume, err := service.Create(context.Background(), volumes.CreateOpts{
				Name:     "test",
				MinSize:  10,
				MaxSize:  20,
				Location: "loc",
			})
			if volume != nil || err == nil {
				t.Fatal("expected to fail")
			}
		})
	}
}

func TestIdempotentServiceDelete(t *testing.T) {
	volumeService := &mock.VolumeService{
		DeleteFunc: func(ctx context.Context, volume *csi.Volume) error {
			return nil
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Delete(context.Background(), &csi.Volume{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdempotentServiceDeleteVolumeNotFound(t *testing.T) {
	volumeService := &mock.VolumeService{
		DeleteFunc: func(ctx context.Context, volume *csi.Volume) error {
			return volumes.ErrVolumeNotFound
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Delete(context.Background(), &csi.Volume{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdempotentServiceDeleteError(t *testing.T) {
	volumeService := &mock.VolumeService{
		DeleteFunc: func(ctx context.Context, volume *csi.Volume) error {
			return io.EOF
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Delete(context.Background(), &csi.Volume{})
	if err != io.EOF {
		t.Fatal(err)
	}
}

func TestIdempotentServiceAttach(t *testing.T) {
	volumeService := &mock.VolumeService{
		AttachFunc: func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
			return nil
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Attach(context.Background(), &csi.Volume{}, &csi.Server{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdempotentServiceAttachAlreadyAttachedSameServer(t *testing.T) {
	volumeService := &mock.VolumeService{
		AttachFunc: func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
			return volumes.ErrAlreadyAttached
		},
		GetByIDFunc: func(ctx context.Context, id uint64) (*csi.Volume, error) {
			return &csi.Volume{
				ID: id,
				Server: &csi.Server{
					ID: 1,
				},
			}, nil
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Attach(context.Background(), &csi.Volume{}, &csi.Server{ID: 1})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdempotentServiceAttachAlreadyAttachedDifferentServer(t *testing.T) {
	volumeService := &mock.VolumeService{
		AttachFunc: func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
			return volumes.ErrAlreadyAttached
		},
		GetByIDFunc: func(ctx context.Context, id uint64) (*csi.Volume, error) {
			return &csi.Volume{
				ID: id,
				Server: &csi.Server{
					ID: 2,
				},
			}, nil
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Attach(context.Background(), &csi.Volume{}, &csi.Server{ID: 1})
	if err != volumes.ErrAlreadyAttached {
		t.Fatal(err)
	}
}

func TestIdempotentServiceDetach(t *testing.T) {
	volumeService := &mock.VolumeService{
		DetachFunc: func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
			return nil
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Detach(context.Background(), &csi.Volume{}, &csi.Server{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdempotentServiceDetachNotAttached(t *testing.T) {
	volumeService := &mock.VolumeService{
		DetachFunc: func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
			return volumes.ErrNotAttached
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Detach(context.Background(), &csi.Volume{}, &csi.Server{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIdempotentServiceDetachAttachedToDifferentServer(t *testing.T) {
	volumeService := &mock.VolumeService{
		DetachFunc: func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
			return volumes.ErrAlreadyAttached
		},
	}

	service := volumes.NewIdempotentService(log.NewNopLogger(), volumeService)

	err := service.Detach(context.Background(), &csi.Volume{}, &csi.Server{})
	if err == nil {
		t.Fatal(err)
	}
}
