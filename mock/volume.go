package mock

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"

	"github.com/hetznercloud/csi-driver/csi"
	"github.com/hetznercloud/csi-driver/volumes"
)

type VolumeService struct {
	CreateFunc        func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error)
	GetServerByIDFunc func(ctx context.Context, id int) (*hcloud.Server, error)
	GetByIDFunc       func(ctx context.Context, id uint64) (*csi.Volume, error)
	GetByNameFunc     func(ctx context.Context, name string) (*csi.Volume, error)
	DeleteFunc        func(ctx context.Context, volume *csi.Volume) error
	AttachFunc        func(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	DetachFunc        func(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	ResizeFunc        func(ctx context.Context, volume *csi.Volume, size int) error
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

func (s *VolumeService) Resize(ctx context.Context, volume *csi.Volume, size int) error {
	if s.ResizeFunc == nil {
		panic("not implemented")
	}
	return s.ResizeFunc(ctx, volume, size)
}

type VolumeMountService struct {
	PublishFunc    func(volumeID string, targetPath string, devicePath string, encryptionPassphrase string, opts volumes.MountOpts) error
	UnpublishFunc  func(volumeID string, targetPath string) error
	PathExistsFunc func(path string) (bool, error)
}

func (s *VolumeMountService) Publish(volumeID string, targetPath string, devicePath string, encryptionPassphrase string, opts volumes.MountOpts) error {
	if s.PublishFunc == nil {
		panic("not implemented")
	}
	return s.PublishFunc(volumeID, targetPath, devicePath, encryptionPassphrase, opts)
}

func (s *VolumeMountService) PathExists(path string) (bool, error) {
	if s.PathExistsFunc == nil {
		panic("not implemented")
	}
	return s.PathExistsFunc(path)
}

func (s *VolumeMountService) Unpublish(volumeID string, targetPath string) error {
	if s.UnpublishFunc == nil {
		panic("not implemented")
	}
	return s.UnpublishFunc(volumeID, targetPath)
}

type VolumeResizeService struct {
	ResizeFunc func(volumeID string, volumePath string) error
}

func (s *VolumeResizeService) Resize(volumeID string, volumePath string) error {
	if s.ResizeFunc == nil {
		panic("not implemented")
	}
	return s.ResizeFunc(volumeID, volumePath)
}

type VolumeStatsService struct {
	ByteFilesystemStatsFunc  func(volumePath string) (totalBytes int64, availableBytes int64, usedBytes int64, err error)
	INodeFilesystemStatsFunc func(volumePath string) (total int64, used int64, free int64, err error)
}

func (s *VolumeStatsService) ByteFilesystemStats(volumePath string) (totalBytes int64, availableBytes int64, usedBytes int64, err error) {
	if s.ByteFilesystemStatsFunc == nil {
		panic("not implemented")
	}
	return s.ByteFilesystemStatsFunc(volumePath)
}

func (s *VolumeStatsService) INodeFilesystemStats(volumePath string) (total int64, used int64, free int64, err error) {
	if s.INodeFilesystemStatsFunc == nil {
		panic("not implemented")
	}
	return s.INodeFilesystemStatsFunc(volumePath)
}
