package volumes

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hetznercloud/csi-driver/csi"
	"k8s.io/kubernetes/pkg/util/resizefs"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
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
			Exec:      exec.New(),
		}),
	}
}

func (l *LinuxResizeService) Resize(volume *csi.Volume, volumePath string) error {
	level.Debug(l.logger).Log(
		"msg", "resizing volume",
		"volume-name", volume.Name,
		"volume-path", volumePath,
	)
	if _, err := l.resizer.Resize(volume.LinuxDevice, volumePath); err != nil {
		return err
	}
	return nil
}
