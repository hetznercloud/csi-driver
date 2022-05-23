package volumes

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/hetznercloud/csi-driver/csi"
)

// IdempotentService wraps a volume service and provides idempotency as required by the CSI spec.
type IdempotentService struct {
	logger        log.Logger
	volumeService Service
}

func NewIdempotentService(logger log.Logger, volumeService Service) *IdempotentService {
	return &IdempotentService{
		logger:        logger,
		volumeService: volumeService,
	}
}

func (s *IdempotentService) Create(ctx context.Context, opts CreateOpts) (*csi.Volume, error) {
	level.Info(s.logger).Log(
		"msg", "creating volume",
		"name", opts.Name,
		"min-size", opts.MinSize,
		"max-size", opts.MaxSize,
		"location", opts.Location,
	)

	volume, err := s.volumeService.Create(ctx, opts)

	if err == nil {
		level.Info(s.logger).Log(
			"msg", "volume created",
			"volume-id", volume.ID,
		)
		return volume, nil
	}

	if err == ErrVolumeAlreadyExists {
		level.Info(s.logger).Log(
			"msg", "another volume with that name does already exist",
			"name", opts.Name,
		)
		existingVolume, err := s.volumeService.GetByName(ctx, opts.Name)
		if err != nil {
			level.Error(s.logger).Log(
				"msg", "failed to get existing volume",
				"name", opts.Name,
				"err", err,
			)
			return nil, err
		}
		if existingVolume == nil {
			level.Error(s.logger).Log(
				"msg", "existing volume disappeared",
				"name", opts.Name,
			)
			return nil, ErrVolumeAlreadyExists
		}
		if existingVolume.Size < opts.MinSize {
			level.Info(s.logger).Log(
				"msg", "existing volume is too small",
				"name", opts.Name,
				"min-size", opts.MinSize,
				"actual-size", existingVolume.Size,
			)
			return nil, ErrVolumeAlreadyExists
		}
		if opts.MaxSize > 0 && existingVolume.Size > opts.MaxSize {
			level.Info(s.logger).Log(
				"msg", "existing volume is too large",
				"name", opts.Name,
				"max-size", opts.MaxSize,
				"actual-size", existingVolume.Size,
			)
			return nil, ErrVolumeAlreadyExists
		}
		if existingVolume.Location != opts.Location {
			level.Info(s.logger).Log(
				"msg", "existing volume is in different location",
				"name", opts.Name,
				"location", opts.Location,
				"actual-location", existingVolume.Location,
			)
			return nil, ErrVolumeAlreadyExists
		}
		return existingVolume, nil
	}

	return nil, err
}

func (s *IdempotentService) All(ctx context.Context) ([]*csi.Volume, error) {
	return s.volumeService.All(ctx)
}

func (s *IdempotentService) GetByID(ctx context.Context, id uint64) (*csi.Volume, error) {
	return s.volumeService.GetByID(ctx, id)
}

func (s *IdempotentService) GetByName(ctx context.Context, name string) (*csi.Volume, error) {
	return s.volumeService.GetByName(ctx, name)
}

func (s *IdempotentService) Delete(ctx context.Context, volume *csi.Volume) error {
	switch err := s.volumeService.Detach(ctx, volume, nil); err {
	case ErrVolumeNotFound, ErrNotAttached, nil:
		break
	default:
		return err
	}

	switch err := s.volumeService.Delete(ctx, volume); err {
	case ErrVolumeNotFound:
		return nil
	case nil:
		return nil
	default:
		return err
	}
}

func (s *IdempotentService) Attach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	attachErr := s.volumeService.Attach(ctx, volume, server)
	if attachErr == nil {
		return nil
	}

	vol, err := s.volumeService.GetByID(ctx, volume.ID)
	if err != nil {
		return err
	}
	if vol.Server != nil && vol.Server.ID == server.ID {
		return nil
	}
	return attachErr
}

func (s *IdempotentService) Detach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	switch err := s.volumeService.Detach(ctx, volume, server); err {
	case ErrNotAttached:
		return nil
	case ErrAttached:
		// Volume is attached to another server
		return nil
	case nil:
		return nil
	default:
		return err
	}
}

func (s *IdempotentService) Resize(ctx context.Context, volume *csi.Volume, size int) error {
	return s.volumeService.Resize(ctx, volume, size)
}
