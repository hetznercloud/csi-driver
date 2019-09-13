package mock

import (
	"context"

	"hetzner.cloud/csi/csi"
	"hetzner.cloud/csi/volumes"
)

type VolumeService struct {
	CreateFunc    func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error)
	GetByIDFunc   func(ctx context.Context, id uint64) (*csi.Volume, error)
	GetByNameFunc func(ctx context.Context, name string) (*csi.Volume, error)
	DeleteFunc    func(ctx context.Context, volume *csi.Volume) error
	AttachFunc    func(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	DetachFunc    func(ctx context.Context, volume *csi.Volume, server *csi.Server) error
}

func (s *VolumeService) Create(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
	if s.CreateFunc == nil {
		panic("not implemented")
	}
	return s.CreateFunc(ctx, opts)
}

func (s *VolumeService) GetByID(ctx context.Context, id uint64) (*csi.Volume, error) {
	if s.GetByIDFunc == nil {
		panic("not implemented")
	}
	return s.GetByIDFunc(ctx, id)
}

func (s *VolumeService) GetByName(ctx context.Context, name string) (*csi.Volume, error) {
	if s.GetByNameFunc == nil {
		panic("not implemented")
	}
	return s.GetByNameFunc(ctx, name)
}

func (s *VolumeService) Delete(ctx context.Context, volume *csi.Volume) error {
	if s.DeleteFunc == nil {
		panic("not implemented")
	}
	return s.DeleteFunc(ctx, volume)
}

func (s *VolumeService) Attach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	if s.AttachFunc == nil {
		panic("not implemented")
	}
	return s.AttachFunc(ctx, volume, server)
}

func (s *VolumeService) Detach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	if s.DetachFunc == nil {
		panic("not implemented")
	}
	return s.DetachFunc(ctx, volume, server)
}

type VolumeMountService struct {
	StageFunc     func(volume *csi.Volume, stagingTargetPath string, opts volumes.MountOpts) error
	UnstageFunc   func(volume *csi.Volume, stagingTargetPath string) error
	PublishFunc   func(volume *csi.Volume, targetPath string, stagingTargetPath string, opts volumes.MountOpts) error
	UnpublishFunc func(volume *csi.Volume, targetPath string) error
}

func (s *VolumeMountService) Stage(volume *csi.Volume, stagingTargetPath string, opts volumes.MountOpts) error {
	if s.StageFunc == nil {
		panic("not implemented")
	}
	return s.StageFunc(volume, stagingTargetPath, opts)
}

func (s *VolumeMountService) Unstage(volume *csi.Volume, stagingTargetPath string) error {
	if s.UnstageFunc == nil {
		panic("not implemented")
	}
	return s.UnstageFunc(volume, stagingTargetPath)
}

func (s *VolumeMountService) Publish(volume *csi.Volume, targetPath string, stagingTargetPath string, opts volumes.MountOpts) error {
	if s.PublishFunc == nil {
		panic("not implemented")
	}
	return s.PublishFunc(volume, targetPath, stagingTargetPath, opts)
}

func (s *VolumeMountService) Unpublish(volume *csi.Volume, targetPath string) error {
	if s.UnpublishFunc == nil {
		panic("not implemented")
	}
	return s.UnpublishFunc(volume, targetPath)
}
