package mock

import (
	"context"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/hetznercloud/csi-driver/csi"
	"github.com/hetznercloud/csi-driver/volumes"
)

type VolumeService struct {
	CreateFunc        func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error)
	GetServerByIDFunc func(ctx context.Context, id int) (*hcloud.Server, error)
	AllFunc           func(ctx context.Context) ([]*csi.Volume, error)
	GetByIDFunc       func(ctx context.Context, id int64) (*csi.Volume, error)
	GetByNameFunc     func(ctx context.Context, name string) (*csi.Volume, error)
	DeleteFunc        func(ctx context.Context, volume *csi.Volume) error
	AttachFunc        func(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	DetachFunc        func(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	ResizeFunc        func(ctx context.Context, volume *csi.Volume, size int) error
}

func (s *VolumeService) All(ctx context.Context) ([]*csi.Volume, error) {
	if s.AllFunc == nil {
		panic("not implemented")
	}
	return s.AllFunc(ctx)
}

func (s *VolumeService) Create(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
	if s.CreateFunc == nil {
		panic("not implemented")
	}
	return s.CreateFunc(ctx, opts)
}

func (s *VolumeService) GetByID(ctx context.Context, id int64) (*csi.Volume, error) {
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
	PublishFunc          func(targetPath string, devicePath string, opts volumes.MountOpts) error
	UnpublishFunc        func(targetPath string) error
	PathExistsFunc       func(path string) (bool, error)
	FormatDiskFunc       func(disk string, fstype string) error
	DetectDiskFormatFunc func(disk string) (string, error)
}

func (s *VolumeMountService) Publish(targetPath string, devicePath string, opts volumes.MountOpts) error {
	if s.PublishFunc == nil {
		panic("not implemented")
	}
	return s.PublishFunc(targetPath, devicePath, opts)
}

func (s *VolumeMountService) PathExists(path string) (bool, error) {
	if s.PathExistsFunc == nil {
		panic("not implemented")
	}
	return s.PathExistsFunc(path)
}

func (s *VolumeMountService) Unpublish(targetPath string) error {
	if s.UnpublishFunc == nil {
		panic("not implemented")
	}
	return s.UnpublishFunc(targetPath)
}

func (s *VolumeMountService) FormatDisk(disk string, fstype string) error {
	if s.FormatDiskFunc == nil {
		panic("not implemented")
	}
	return s.FormatDiskFunc(disk, fstype)
}

func (s *VolumeMountService) DetectDiskFormat(disk string) (string, error) {
	if s.DetectDiskFormatFunc == nil {
		panic("not implemented")
	}
	return s.DetectDiskFormatFunc(disk)
}

type VolumeResizeService struct {
	ResizeFunc func(volumePath string) error
}

func (s *VolumeResizeService) Resize(volumePath string) error {
	if s.ResizeFunc == nil {
		panic("not implemented")
	}
	return s.ResizeFunc(volumePath)
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
