package api

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/hetznercloud/hcloud-go/hcloud"

	"hetzner.cloud/csi"
	"hetzner.cloud/csi/volumes"
)

type VolumeService struct {
	logger log.Logger
	client *hcloud.Client
}

func NewVolumeService(logger log.Logger, client *hcloud.Client) *VolumeService {
	return &VolumeService{
		logger: logger,
		client: client,
	}
}

func (s *VolumeService) Create(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
	result, _, err := s.client.Volume.Create(ctx, hcloud.VolumeCreateOpts{
		Name:     opts.Name,
		Size:     opts.MinSize,
		Location: &hcloud.Location{Name: opts.Location},
	})
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCode("uniqueness_error")) {
			return nil, volumes.ErrVolumeAlreadyExists
		}
		return nil, err
	}

	_, errCh := s.client.Action.WatchProgress(ctx, result.Action)
	if err := <-errCh; err != nil {
		_, _ = s.client.Volume.Delete(ctx, result.Volume) // fire and forget
		return nil, err
	}

	return toDomainVolume(result.Volume), nil
}

func (s *VolumeService) GetByID(ctx context.Context, id uint64) (*csi.Volume, error) {
	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, int(id))
	if err != nil {
		return nil, err
	}
	if hcloudVolume == nil {
		return nil, volumes.ErrVolumeNotFound
	}
	return toDomainVolume(hcloudVolume), nil
}

func (s *VolumeService) GetByName(ctx context.Context, name string) (*csi.Volume, error) {
	hcloudVolume, _, err := s.client.Volume.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if hcloudVolume == nil {
		return nil, volumes.ErrVolumeNotFound
	}
	return toDomainVolume(hcloudVolume), nil
}

func (s *VolumeService) Delete(ctx context.Context, volume *csi.Volume) error {
	hcloudVolume := &hcloud.Volume{ID: int(volume.ID)}
	_, err := s.client.Volume.Delete(ctx, hcloudVolume)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return volumes.ErrVolumeNotFound
		}
		return err
	}
	return nil
}

func (s *VolumeService) Attach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, int(volume.ID))
	if err != nil {
		return err
	}
	if hcloudVolume == nil {
		return volumes.ErrVolumeNotFound
	}

	hcloudServer, _, err := s.client.Server.GetByID(ctx, int(server.ID))
	if err != nil {
		return err
	}
	if hcloudServer == nil {
		return volumes.ErrServerNotFound
	}

	action, _, err := s.client.Volume.Attach(ctx, hcloudVolume, hcloudServer)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCode("limit_exceeded_error")) {
			return volumes.ErrAttachLimitReached
		}
		//TODO: update once hcloud upgraded to v1.12
		if hcloud.IsError(err, "locked"){
			return volumes.ErrLockedServer
		}
		return err
	}

	_, errCh := s.client.Action.WatchProgress(ctx, action)
	if err := <-errCh; err != nil {
		return err
	}
	return nil
}

func (s *VolumeService) Detach(ctx context.Context, volume *csi.Volume) error {
	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, int(volume.ID))
	if err != nil {
		return err
	}
	if hcloudVolume == nil {
		return volumes.ErrVolumeNotFound
	}

	if hcloudVolume.Server == nil {
		return volumes.ErrNotAttached
	}

	action, _, err := s.client.Volume.Detach(ctx, hcloudVolume)
	if err != nil {
		//TODO: update once hcloud upgraded to v1.12
		if hcloud.IsError(err, "locked"){
			return volumes.ErrLockedServer
		}
		return err
	}

	_, errCh := s.client.Action.WatchProgress(ctx, action)
	if err := <-errCh; err != nil {
		return err
	}
	return nil
}
