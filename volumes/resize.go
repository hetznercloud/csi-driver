package volumes

import (
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
)

// ResizeService resizes volumes.
type ResizeService interface {
	Resize(volumePath string) error
}

// LinuxResizeService resizes volumes on a Linux system.
type LinuxResizeService struct {
	logger     log.Logger
	resizer    *mount.ResizeFs
	cryptSetup *CryptSetup
}

func NewLinuxResizeService(logger log.Logger) *LinuxResizeService {
	return &LinuxResizeService{
		logger: logger,
		resizer: mount.NewResizeFs(mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      exec.New(),
		}.Exec),
		cryptSetup: NewCryptSetup(logger),
	}
}

func (l *LinuxResizeService) Resize(volumePath string) error {
	devicePath, _, err := mount.GetDeviceNameFromMount(mount.New(""), volumePath)
	if err != nil {
		return fmt.Errorf("failed to determine mount path for %s: %s", volumePath, err)
	}

	level.Info(l.logger).Log(
		"msg", "resizing volume",
		"volume-path", volumePath,
		"device-path", devicePath,
	)

	luksDeviceName := GenerateLUKSDeviceName(devicePath)
	active, err := l.cryptSetup.IsActive(luksDeviceName)
	if err != nil {
		return err
	}
	if active {
		luksDevicePath := GenerateLUKSDevicePath(luksDeviceName)
		if err := l.cryptSetup.Resize(luksDeviceName); err != nil {
			return err
		}
		devicePath = luksDevicePath
	}

	if _, err := l.resizer.Resize(devicePath, volumePath); err != nil {
		return err
	}
	return nil
}
