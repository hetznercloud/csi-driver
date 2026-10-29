package volsrv

import (
	"context"
	"log/slog"

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

func (s *VolumeService) All(ctx context.Context) ([]*hcloud.Volume, error) {
	hcloudVolumes, err := s.client.Volume.All(ctx)
	if err != nil {
		s.logger.Info(
			"failed to get volumes",
			"err", err,
		)
		return nil, err
	}
	return hcloudVolumes, nil
}

func (s *VolumeService) Create(ctx context.Context, opts volumes.CreateOpts) (*hcloud.Volume, error) {
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
		if hcloud.IsError(err, hcloud.ErrorCodeUniquenessError) {
			return s.verifyExistingVolume(ctx, opts)
		}
		s.logger.Info(
			"failed to create volume",
			"volume-name", opts.Name,
			"err", err,
		)
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

	return result.Volume, nil
}

func (s *VolumeService) verifyExistingVolume(ctx context.Context, opts volumes.CreateOpts) (*hcloud.Volume, error) {
	s.logger.Info(
		"another volume with that name does already exist",
		"volume-name", opts.Name,
	)

	volume, err := s.GetByName(ctx, opts.Name)
	if err != nil {
		return nil, err
	}
	if volume.Size < opts.MinSize {
		s.logger.Info(
			"existing volume is too small",
			"volume-name", opts.Name,
			"min-size", opts.MinSize,
			"actual-size", volume.Size,
		)
		return nil, volumes.ErrVolumeAlreadyExists
	}
	if opts.MaxSize > 0 && volume.Size > opts.MaxSize {
		s.logger.Info(
			"existing volume is too large",
			"volume-name", opts.Name,
			"max-size", opts.MaxSize,
			"actual-size", volume.Size,
		)
		return nil, volumes.ErrVolumeAlreadyExists
	}
	if volume.Location.Name != opts.Location {
		s.logger.Info(
			"existing volume is in different location",
			"volume-name", opts.Name,
			"volume-location", opts.Location,
			"actual-location", volume.Location.Name,
		)
		return nil, volumes.ErrVolumeAlreadyExists
	}
	return volume, nil
}

func (s *VolumeService) GetByID(ctx context.Context, id int64) (*hcloud.Volume, error) {
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
	return hcloudVolume, nil
}

func (s *VolumeService) GetByName(ctx context.Context, name string) (*hcloud.Volume, error) {
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
	return hcloudVolume, nil
}

func (s *VolumeService) Delete(ctx context.Context, volume *hcloud.Volume) error {
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
		return nil
	}
	if hcloudVolume.Server != nil {
		s.logger.Info(
			"detaching volume before deletion",
			"volume-id", volume.ID,
			"server-id", hcloudVolume.Server.ID,
		)
		if err := s.detach(ctx, hcloudVolume); err != nil {
			return err
		}
	}

	if _, err := s.client.Volume.Delete(ctx, hcloudVolume); err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			s.logger.Info(
				"volume to delete not found",
				"volume-id", volume.ID,
			)
			return nil
		}
		s.logger.Info(
			"failed to delete volume",
			"volume-id", volume.ID,
			"err", err,
		)
		return err
	}
	s.logger.Info(
		"volume deleted",
		"volume-id", volume.ID,
	)

	return nil
}

func (s *VolumeService) Attach(ctx context.Context, volume *hcloud.Volume, server *hcloud.Server) error {
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

	if err := s.attach(ctx, hcloudVolume, hcloudServer); err != nil {
		if s.isAttachedTo(ctx, volume.ID, server.ID) {
			s.logger.Info(
				"volume is attached to this server despite the error",
				"volume-id", volume.ID,
				"server-id", server.ID,
				"err", err,
			)
			return nil
		}
		return err
	}
	return nil
}

func (s *VolumeService) attach(ctx context.Context, volume *hcloud.Volume, server *hcloud.Server) error {
	action, _, err := s.client.Volume.Attach(ctx, volume, server)
	if err != nil {
		s.logger.Info(
			"failed to attach volume",
			"volume-id", volume.ID,
			"server-id", server.ID,
			"err", err,
		)
		if hcloud.IsError(err, hcloud.ErrorCodeResourceLimitExceeded) {
			return volumes.ErrAttachLimitReached
		}
		if hcloud.IsError(err, hcloud.ErrorCodeLocked) {
			return volumes.ErrLockedServer
		}
		if hcloud.IsError(err, hcloud.ErrorCodeVolumeAlreadyAttached) {
			return volumes.ErrAttached
		}
		return err
	}
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

func (s *VolumeService) isAttachedTo(ctx context.Context, volumeID int64, serverID int64) bool {
	hcloudVolume, _, err := s.client.Volume.GetByID(ctx, volumeID)
	if err != nil {
		s.logger.Info(
			"failed to get volume",
			"volume-id", volumeID,
			"err", err,
		)
		return false
	}
	return hcloudVolume != nil && hcloudVolume.Server != nil && hcloudVolume.Server.ID == serverID
}

func (s *VolumeService) Detach(ctx context.Context, volume *hcloud.Volume, server *hcloud.Server) error {
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
		return nil
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
		return nil
	}

	return s.detach(ctx, hcloudVolume)
}

func (s *VolumeService) detach(ctx context.Context, volume *hcloud.Volume) error {
	action, _, err := s.client.Volume.Detach(ctx, volume)
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

func (s *VolumeService) Resize(ctx context.Context, volume *hcloud.Volume, size int) error {
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
		return nil
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
