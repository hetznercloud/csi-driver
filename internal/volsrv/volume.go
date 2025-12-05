package volsrv

import (
	"context"
	"log/slog"
	"time"

	"github.com/hetznercloud/csi-driver/internal/csi"
	"github.com/hetznercloud/csi-driver/internal/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type VolumeService struct {
	logger *slog.Logger
	client *hcloud.Client
}

func NewVolumeService(logger *slog.Logger, client *hcloud.Client) *VolumeService {
	return &VolumeService{
		logger: logger,
		client: client,
	}
}

func (s *VolumeService) All(ctx context.Context) ([]*csi.Volume, error) {
	hcloudVolumes, err := s.client.Volume.All(ctx)
	if err != nil {
		s.logger.Info(
			"failed to get volumes",
			"err", err,
		)
		return nil, err
	}

	volumes := make([]*csi.Volume, 0, len(hcloudVolumes))
	for i, hcloudVolume := range hcloudVolumes {
		volumes[i] = toDomainVolume(hcloudVolume)
	}
	return volumes, nil
}

func (s *VolumeService) Create(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
	s.logger.Info(
		"creating volume",
		"volume-name", opts.Name,
		"volume-size", opts.MinSize,
		"volume-location", opts.Location,
	)

	result, _, err := s.client.Volume.Create(ctx, hcloud.VolumeCreateOpts{
		Name:     opts.Name,
		Size:     opts.MinSize,
		Location: &hcloud.Location{Name: opts.Location},
		Labels:   opts.Labels,
	})

	if err != nil {
		s.logger.Info(
			"failed to create volume",
			"volume-name", opts.Name,
			"err", err,
		)
		if hcloud.IsError(err, hcloud.ErrorCode("uniqueness_error")) {
			return nil, volumes.ErrVolumeAlreadyExists
		}
		return nil, err
	}

	if err := s.client.Action.WaitFor(ctx, result.Action); err != nil {
		s.logger.Info(
			"failed to create volume",
			"volume-name", opts.Name,
			"err", err,
		)
		_, _ = s.client.Volume.Delete(ctx, result.Volume) // fire and forget
		return nil, err
	}

	return toDomainVolume(result.Volume), nil
}

func (s *VolumeService) GetByID(ctx context.Context, id int64) (*csi.Volume, error) {
	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, id)
	if err != nil {
		s.logger.Info(
			"failed to get volume",
			"volume-id", id,
			"err", err,
		)
		return nil, err
	}
	if hcloudVolume == nil {
		s.logger.Info(
			"volume not found",
			"volume-id", id,
		)
		return nil, volumes.ErrVolumeNotFound
	}
	return toDomainVolume(hcloudVolume), nil
}

func (s *VolumeService) GetByName(ctx context.Context, name string) (*csi.Volume, error) {
	hcloudVolume, _, err := s.client.Volume.GetByName(ctx, name)
	if err != nil {
		s.logger.Info(
			"failed to get volume",
			"volume-name", name,
			"err", err,
		)
		return nil, err
	}
	if hcloudVolume == nil {
		s.logger.Info(
			"volume not found",
			"volume-name", name,
		)
		return nil, volumes.ErrVolumeNotFound
	}
	return toDomainVolume(hcloudVolume), nil
}

func (s *VolumeService) Delete(ctx context.Context, volume *csi.Volume) error {
	s.logger.Info(
		"deleting volume",
		"volume-id", volume.ID,
	)

	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, volume.ID)
	if err != nil {
		s.logger.Info(
			"failed to get volume",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	if hcloudVolume == nil {
		s.logger.Info(
			"volume to delete not found",
			"volume-id", volume.ID,
		)
		return volumes.ErrVolumeNotFound
	}
	if hcloudVolume.Server != nil {
		s.logger.Info(
			"volume is attached to a server",
			"volume-id", volume.ID,
			"server-id", hcloudVolume.Server.ID,
		)
		return volumes.ErrAttached
	}

	if _, err := s.client.Volume.Delete(ctx, hcloudVolume); err != nil {
		s.logger.Info(
			"failed to delete volume",
			"volume-id", volume.ID,
			"err", err,
		)
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			return volumes.ErrVolumeNotFound
		}
		return err
	}
	s.logger.Info(
		"volume deleted",
		"volume-id", volume.ID,
	)

	return nil
}

func (s *VolumeService) Attach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	s.logger.Info(
		"attaching volume",
		"volume-id", volume.ID,
		"server-id", server.ID,
	)

	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, volume.ID)
	if err != nil {
		s.logger.Info(
			"failed to get volume",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	if hcloudVolume == nil {
		s.logger.Info(
			"volume to attach not found",
			"volume-id", volume.ID,
		)
		return volumes.ErrVolumeNotFound
	}

	hcloudServer, _, err := s.client.Server.GetByID(ctx, server.ID)
	if err != nil {
		s.logger.Info(
			"failed to get server",
			"volume-id", volume.ID,
			"server-id", server.ID,
			"err", err,
		)
		return err
	}
	if hcloudServer == nil {
		s.logger.Info(
			"server to attach volume to not found",
			"volume-id", volume.ID,
			"server-id", server.ID,
		)
		return volumes.ErrServerNotFound
	}

	if hcloudVolume.Server != nil {
		if hcloudVolume.Server.ID == hcloudServer.ID {
			s.logger.Info(
				"volume is already attached to this server",
				"volume-id", volume.ID,
				"server-id", server.ID,
			)
			return nil
		}
		s.logger.Info(
			"volume is already attached to another server",
			"volume-id", volume.ID,
			"server-id", hcloudVolume.Server.ID,
		)
		return volumes.ErrAttached
	}

	action, _, err := s.client.Volume.Attach(ctx, hcloudVolume, hcloudServer)
	if err != nil {
		s.logger.Info(
			"failed to attach volume",
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
	if err := s.client.Action.WaitFor(ctx, action); err != nil {
		s.logger.Info(
			"failed to attach volume",
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
		s.logger.Info(
			"detaching volume from server",
			"volume-id", volume.ID,
			"server-id", server.ID,
		)
	} else {
		s.logger.Info(
			"detaching volume from any server",
			"volume-id", volume.ID,
		)
	}

	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, volume.ID)
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			s.logger.Info(
				"volume to detach not found",
				"volume-id", volume.ID,
				"err", err,
			)
			return volumes.ErrVolumeNotFound
		}
		s.logger.Info(
			"failed to get volume to detach",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	if hcloudVolume == nil {
		s.logger.Info(
			"volume to detach not found",
			"volume-id", volume.ID,
			"err", err,
		)
		return volumes.ErrVolumeNotFound
	}
	if hcloudVolume.Server == nil {
		s.logger.Info(
			"volume not attached to a server",
			"volume-id", volume.ID,
		)
		return volumes.ErrNotAttached
	}

	// If a server is provided, only detach if the volume is actually attached
	// to that server.
	if server != nil && hcloudVolume.Server.ID != server.ID {
		s.logger.Info(
			"volume not attached to provided server",
			"volume-id", volume.ID,
			"detach-from-server-id", server.ID,
			"attached-to-server-id", hcloudVolume.Server.ID,
		)
		return volumes.ErrAttached
	}

	action, _, err := s.client.Volume.Detach(ctx, hcloudVolume)
	if err != nil {
		s.logger.Info(
			"failed to detach volume",
			"volume-id", volume.ID,
			"err", err,
		)
		if hcloud.IsError(err, hcloud.ErrorCodeLocked) {
			return volumes.ErrLockedServer
		}
		return err
	}

	if err := s.client.Action.WaitFor(ctx, action); err != nil {
		s.logger.Info(
			"failed to detach volume",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	return nil
}

func (s *VolumeService) Resize(ctx context.Context, volume *csi.Volume, size int) error {
	logger := s.logger.With("volume-id", volume.ID, "requested-size", size)

	logger.Info(
		"resize volume",
	)

	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, volume.ID)
	if err != nil {
		logger.Info("failed to get volume", "err", err)
		return err
	}
	if hcloudVolume == nil {
		logger.Info("volume to resize not found")
		return volumes.ErrVolumeNotFound
	}

	logger = logger.With("current-size", hcloudVolume.Size)

	if hcloudVolume.Size >= size {
		logger.Info("volume size is already larger or equal than the requested size")
		return volumes.ErrVolumeSizeAlreadyReached
	}

	action, _, err := s.client.Volume.Resize(ctx, hcloudVolume, size)
	if err != nil {
		logger.Info("failed to resize volume", "err", err)
		return err
	}

	if err = s.client.Action.WaitFor(ctx, action); err != nil {
		logger.Info("failed to resize volume", "err", err)
		return err
	}
	return nil
}
