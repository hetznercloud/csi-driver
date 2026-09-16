package volumes

import (
	"context"
	"errors"

	"github.com/hetznercloud/csi-driver/internal/csi"
)

const SnapshotIDLabel = "source-snapshot-id"

var (
	ErrVolumeNotFound           = errors.New("volume not found")
	ErrVolumeAlreadyExists      = errors.New("volume does already exist")
	ErrServerNotFound           = errors.New("server not found")
	ErrAttached                 = errors.New("volume is attached")
	ErrNotAttached              = errors.New("volume is not attached")
	ErrAttachLimitReached       = errors.New("max number of attachments per server reached")
	ErrLockedServer             = errors.New("server is locked")
	ErrVolumeSizeAlreadyReached = errors.New("volume size is already larger or equal than the requested size")
	ErrSnapshotNotFound         = errors.New("snapshot not found")
	ErrSnapshotAlreadyExists    = errors.New("snapshot does already exist")
)

type Service interface {
	Create(ctx context.Context, opts CreateOpts) (*csi.Volume, error)
	GetByID(ctx context.Context, id int64) (*csi.Volume, error)
	GetByName(ctx context.Context, name string) (*csi.Volume, error)
	Delete(ctx context.Context, volume *csi.Volume) error
	Attach(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	Detach(ctx context.Context, volume *csi.Volume, server *csi.Server) error
	Resize(ctx context.Context, volume *csi.Volume, size int) error
	All(ctx context.Context) ([]*csi.Volume, error)
	CreateSnapshot(ctx context.Context, opts CreateSnapshotOpts) (*csi.Snapshot, error)
	GetSnapshotByID(ctx context.Context, id int64) (*csi.Snapshot, error)
	GetSnapshotByName(ctx context.Context, name string) (*csi.Snapshot, error)
	DeleteSnapshot(ctx context.Context, snapshot *csi.Snapshot) error
	AllSnapshots(ctx context.Context) ([]*csi.Snapshot, error)
}

// CreateOpts specifies the options for creating a volume.
type CreateOpts struct {
	Name     string
	MinSize  int
	MaxSize  int
	Location string
	Labels   map[string]string
	Snapshot *csi.Snapshot
}

// CreateSnapshotOpts specifies the options for a point-in-time volume snapshot.
type CreateSnapshotOpts struct {
	Name     string
	VolumeID int64
	Labels   map[string]string
}
