package volumes

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"hetzner.cloud/csi/csi"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/util/resizefs"
)

// ResizeService resizes volumes.
type ResizeService interface {
	Resize(volume *csi.Volume, volumePath string) error
}

// LinuxResizeService resizes volumes on a Linux system.
type LinuxResizeService struct {
	logger  log.Logger
	resizer *resizefs.ResizeFs
}

func NewLinuxResizeService(logger log.Logger) *LinuxResizeService {
	return &LinuxResizeService{
		logger: logger,
		resizer: resizefs.NewResizeFs(&mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      mount.NewOsExec(),
		}),
	}
}

func (l *LinuxResizeService) Resize(volume *csi.Volume, volumePath string) error {
	level.Info(l.logger).Log(
		"msg", "resizing volume",
		"volume-name", volume.Name,
		"volume-path", volumePath,
	)
	if _, err := l.resizer.Resize(volume.LinuxDevice, volumePath); err != nil {
		return err
	}
	return nil
}
