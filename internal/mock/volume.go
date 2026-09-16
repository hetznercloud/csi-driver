package mock

import (
	"context"

	"github.com/hetznercloud/csi-driver/internal/csi"
	"github.com/hetznercloud/csi-driver/internal/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type VolumeService struct {
	CreateFunc            func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error)
	GetServerByIDFunc     func(ctx context.Context, id int) (*hcloud.Server, error)
	AllFunc               func(ctx context.Context) ([]*csi.Volume, error)
	GetByIDFunc           func(ctx context.Context, id int64) (*csi.Volume, error)
	GetByNameFunc         func(ctx context.Context, name string) (*csi.Volume, error)
	DeleteFunc            func(ctx context.Context, volume *csi.Volume) error
	AttachFunc            func(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	DetachFunc            func(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	ResizeFunc            func(ctx context.Context, volume *csi.Volume, size int) error
	CreateSnapshotFunc    func(ctx context.Context, opts volumes.CreateSnapshotOpts) (*csi.Snapshot, error)
	GetSnapshotByIDFunc   func(ctx context.Context, id int64) (*csi.Snapshot, error)
	GetSnapshotByNameFunc func(ctx context.Context, name string) (*csi.Snapshot, error)
	DeleteSnapshotFunc    func(ctx context.Context, snapshot *csi.Snapshot) error
	AllSnapshotsFunc      func(ctx context.Context) ([]*csi.Snapshot, error)
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

func (s *VolumeService) CreateSnapshot(ctx context.Context, opts volumes.CreateSnapshotOpts) (*csi.Snapshot, error) {
	if s.CreateSnapshotFunc == nil {
		panic("not implemented")
	}
	return s.CreateSnapshotFunc(ctx, opts)
}

func (s *VolumeService) GetSnapshotByID(ctx context.Context, id int64) (*csi.Snapshot, error) {
	if s.GetSnapshotByIDFunc == nil {
		panic("not implemented")
	}
	return s.GetSnapshotByIDFunc(ctx, id)
}

func (s *VolumeService) GetSnapshotByName(ctx context.Context, name string) (*csi.Snapshot, error) {
	if s.GetSnapshotByNameFunc == nil {
		panic("not implemented")
	}
	return s.GetSnapshotByNameFunc(ctx, name)
}

func (s *VolumeService) DeleteSnapshot(ctx context.Context, snapshot *csi.Snapshot) error {
	if s.DeleteSnapshotFunc == nil {
		panic("not implemented")
	}
	return s.DeleteSnapshotFunc(ctx, snapshot)
}

func (s *VolumeService) AllSnapshots(ctx context.Context) ([]*csi.Snapshot, error) {
	if s.AllSnapshotsFunc == nil {
		panic("not implemented")
	}
	return s.AllSnapshotsFunc(ctx)
}

type VolumeMountService struct {
	PublishFunc    func(ctx context.Context, targetPath string, devicePath string, opts volumes.MountOpts) error
	UnpublishFunc  func(ctx context.Context, targetPath string) error
	PathExistsFunc func(path string) (bool, error)
}

func (s *VolumeMountService) Publish(ctx context.Context, targetPath string, devicePath string, opts volumes.MountOpts) error {
	if s.PublishFunc == nil {
		panic("not implemented")
	}
	return s.PublishFunc(ctx, targetPath, devicePath, opts)
}

func (s *VolumeMountService) Unpublish(ctx context.Context, targetPath string) error {
	if s.UnpublishFunc == nil {
		panic("not implemented")
	}
	return s.UnpublishFunc(ctx, targetPath)
}

func (s *VolumeMountService) PathExists(path string) (bool, error) {
	if s.PathExistsFunc == nil {
		panic("not implemented")
	}
	return s.PathExistsFunc(path)
}

type VolumeResizeService struct {
	ResizeFunc func(ctx context.Context, volumePath string) error
}

func (s *VolumeResizeService) Resize(ctx context.Context, volumePath string) error {
	if s.ResizeFunc == nil {
		panic("not implemented")
	}
	return s.ResizeFunc(ctx, volumePath)
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
