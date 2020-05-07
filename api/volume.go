package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hetznercloud/hcloud-go/hcloud"

	"hetzner.cloud/csi/csi"
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
	level.Info(s.logger).Log(
		"msg", "creating volume",
		"volume-name", opts.Name,
		"volume-size", opts.MinSize,
		"volume-location", opts.Location,
	)

	result, _, err := s.client.Volume.Create(ctx, hcloud.VolumeCreateOpts{
		Name:     opts.Name,
		Size:     opts.MinSize,
		Location: &hcloud.Location{Name: opts.Location},
	})
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to create volume",
			"volume-name", opts.Name,
			"err", err,
		)
		if hcloud.IsError(err, hcloud.ErrorCode("uniqueness_error")) {
			return nil, volumes.ErrVolumeAlreadyExists
		}
		return nil, err
	}

	_, errCh := s.client.Action.WatchProgress(ctx, result.Action)
	if err := <-errCh; err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to create volume",
			"volume-name", opts.Name,
			"err", err,
		)
		_, _ = s.client.Volume.Delete(ctx, result.Volume) // fire and forget
		return nil, err
	}

	return toDomainVolume(result.Volume), nil
}

func (s *VolumeService) GetByID(ctx context.Context, id uint64) (*csi.Volume, error) {
	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, int(id))
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to get volume",
			"volume-id", id,
			"err", err,
		)
		return nil, err
	}
	if hcloudVolume == nil {
		level.Info(s.logger).Log(
			"msg", "volume not found",
			"volume-id", id,
		)
		return nil, volumes.ErrVolumeNotFound
	}
	return toDomainVolume(hcloudVolume), nil
}

func (s *VolumeService) GetByName(ctx context.Context, name string) (*csi.Volume, error) {
	hcloudVolume, _, err := s.client.Volume.GetByName(ctx, name)
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to get volume",
			"volume-name", name,
			"err", err,
		)
		return nil, err
	}
	if hcloudVolume == nil {
		level.Info(s.logger).Log(
			"msg", "volume not found",
			"volume-name", name,
		)
		return nil, volumes.ErrVolumeNotFound
	}
	return toDomainVolume(hcloudVolume), nil
}

func (s *VolumeService) Delete(ctx context.Context, volume *csi.Volume) error {
	level.Info(s.logger).Log(
		"msg", "deleting volume",
		"volume-id", volume.ID,
	)

	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, int(volume.ID))
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to get volume",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	if hcloudVolume == nil {
		level.Info(s.logger).Log(
			"msg", "volume to delete not found",
			"volume-id", volume.ID,
		)
		return volumes.ErrVolumeNotFound
	}
	if hcloudVolume.Server != nil {
		level.Info(s.logger).Log(
			"msg", "volume is attached to a server",
			"volume-id", volume.ID,
			"server-id", hcloudVolume.Server.ID,
		)
		return volumes.ErrAttached
	}

	if _, err := s.client.Volume.Delete(ctx, hcloudVolume); err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to delete volume",
			"volume-id", volume.ID,
			"err", err,
		)
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return volumes.ErrVolumeNotFound
		}
		return err
	}

	return nil
}

func (s *VolumeService) Attach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	level.Info(s.logger).Log(
		"msg", "attaching volume",
		"volume-id", volume.ID,
		"server-id", server.ID,
	)

	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, int(volume.ID))
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to get volume",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	if hcloudVolume == nil {
		level.Info(s.logger).Log(
			"msg", "volume to attach not found",
			"volume-id", volume.ID,
		)
		return volumes.ErrVolumeNotFound
	}

	hcloudServer, _, err := s.client.Server.GetByID(ctx, int(server.ID))
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to get server",
			"volume-id", volume.ID,
			"server-id", server.ID,
			"err", err,
		)
		return err
	}
	if hcloudServer == nil {
		level.Info(s.logger).Log(
			"msg", "server to attach volume to not found",
			"volume-id", volume.ID,
			"server-id", server.ID,
		)
		return volumes.ErrServerNotFound
	}

	action, _, err := s.client.Volume.Attach(ctx, hcloudVolume, hcloudServer)
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to attach volume",
			"volume-id", volume.ID,
			"server-id", server.ID,
			"err", err,
		)
		if hcloud.IsError(err, hcloud.ErrorCode("limit_exceeded_error")) {
			return volumes.ErrAttachLimitReached
		}
		if hcloud.IsError(err, hcloud.ErrorCodeLocked) {
			return volumes.ErrLockedServer
		}
		if hcloud.IsError(err, hcloud.ErrorCode("volume_already_attached")) {
			return volumes.ErrAttached
		}
		return err
	}
	time.Sleep(3 * time.Second) // We know that the Attach action will take some time, so we wait 3 seconds before starting polling the action status. Within these 3 seconds the volume attach action may be already finished.
	_, errCh := s.client.Action.WatchProgress(ctx, action)
	if err := <-errCh; err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to attach volume",
			"volume-id", volume.ID,
			"server-id", server.ID,
			"err", err,
		)
		return err
	}
	return nil
}

func (s *VolumeService) Detach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	if server != nil {
		level.Info(s.logger).Log(
			"msg", "detaching volume from server",
			"volume-id", volume.ID,
			"server-id", server.ID,
		)
	} else {
		level.Info(s.logger).Log(
			"msg", "detaching volume from any server",
			"volume-id", volume.ID,
		)
	}

	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, int(volume.ID))
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to get volume to detach",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	if hcloudVolume == nil {
		level.Info(s.logger).Log(
			"msg", "volume to detach not found",
			"volume-id", volume.ID,
			"err", err,
		)
		return volumes.ErrVolumeNotFound
	}
	if hcloudVolume.Server == nil {
		level.Info(s.logger).Log(
			"msg", "volume not attached to a server",
			"volume-id", volume.ID,
		)
		return volumes.ErrNotAttached
	}

	// If a server is provided, only detach if the volume is actually attached
	// to that server.
	if server != nil && hcloudVolume.Server.ID != int(server.ID) {
		level.Info(s.logger).Log(
			"msg", "volume not attached to provided server",
			"volume-id", volume.ID,
			"detach-from-server-id", server.ID,
			"attached-to-server-id", hcloudVolume.Server.ID,
		)
		return volumes.ErrAttached
	}

	action, _, err := s.client.Volume.Detach(ctx, hcloudVolume)
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to detach volume",
			"volume-id", volume.ID,
			"err", err,
		)
		if hcloud.IsError(err, hcloud.ErrorCodeLocked) {
			return volumes.ErrLockedServer
		}
		return err
	}

	_, errCh := s.client.Action.WatchProgress(ctx, action)
	if err := <-errCh; err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to detach volume",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	return nil
}

func (s *VolumeService) Resize(ctx context.Context, volume *csi.Volume, size int) error {
	level.Info(s.logger).Log(
		"msg", "resize volume",
		"volume-id", volume.ID,
		"requested-size", size,
	)

	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, int(volume.ID))
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to get volume",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	if hcloudVolume == nil {
		level.Info(s.logger).Log(
			"msg", "volume to resize not found",
			"volume-id", volume.ID,
		)
		return volumes.ErrVolumeNotFound
	}

	action, _, err := s.client.Volume.Resize(ctx, hcloudVolume, size)
	if err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to resize volume",
			"volume-id", volume.ID,
			"size", size,
			"err", err,
		)
		return err
	}

	_, errCh := s.client.Action.WatchProgress(ctx, action)
	if err := <-errCh; err != nil {
		level.Info(s.logger).Log(
			"msg", "failed to resize volume",
			"volume-id", volume.ID,
			"size", size,
			"err", err,
		)
		return err
	}
	return nil
}
