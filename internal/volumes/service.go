package volumes

import (
	"context"
	"errors"

	"github.com/hetznercloud/csi-driver/internal/csi"
)

var (
	ErrVolumeNotFound           = errors.New("volume not found")
	ErrVolumeAlreadyExists      = errors.New("volume does already exist")
	ErrServerNotFound           = errors.New("server not found")
	ErrAttached                 = errors.New("volume is attached")
	ErrNotAttached              = errors.New("volume is not attached")
	ErrAttachLimitReached       = errors.New("max number of attachments per server reached")
	ErrLockedServer             = errors.New("server is locked")
	ErrVolumeSizeAlreadyReached = errors.New("volume size is already larger or equal than the requested size")
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
}

// CreateOpts specifies the options for creating a volume.
type CreateOpts struct {
	Name     string
	MinSize  int
	MaxSize  int
	Location string
}
