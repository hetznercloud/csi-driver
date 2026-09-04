package volumes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"golang.org/x/sys/unix"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const (
	DefaultFSType = "ext4"
	// XFSDefaultConfigPath is the oldest Linux Version available from `xfsprogs`. If this becomes unavailable, we need to increase it to the next lowest version and announce the change in the Release Notes.
	XFSDefaultConfigPath = "/usr/share/xfsprogs/mkfs/lts_4.19.conf"
)

// MountOpts specifies options for mounting a volume.
type MountOpts struct {
	BlockVolume          bool
	FSType               string
	Readonly             bool
	Additional           []string // Additional mount options/flags passed to /bin/mount
	EncryptionPassphrase string
	FsFormatOptions      string
}

// MountService mounts volumes.
type MountService interface {
	// Publish mounts the volume with the passed ID at targetPath. The device holding the
	// volume is looked up on the node, see DeviceForVolume.
	Publish(ctx context.Context, targetPath string, volumeID string, opts MountOpts) error
	Unpublish(ctx context.Context, targetPath string) error
	PathExists(path string) (bool, error)
}

// LinuxMountService mounts volumes on a Linux system.
type LinuxMountService struct {
	logger     *slog.Logger
	mounter    *mount.SafeFormatAndMount
	cryptSetup *CryptSetup
}

func NewLinuxMountService(logger *slog.Logger) *LinuxMountService {
	return &LinuxMountService{
		logger: logger,
		mounter: &mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      exec.New(),
		},
		cryptSetup: NewCryptSetup(logger),
	}
}

func (s *LinuxMountService) Publish(ctx context.Context, targetPath string, volumeID string, opts MountOpts) error {
	// Find the device holding the volume, and wait for it in case the attachment has not
	// reached the node yet.
	devicePath, err := s.waitDeviceReady(ctx, volumeID)
	if err != nil {
		return fmt.Errorf("device of volume %s not ready: %w", volumeID, err)
	}

	isMountPoint, err := s.mounter.IsMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			isMountPoint = false
		} else {
			return err
		}
	}
	if isMountPoint {
		return nil
	}

	var mountOptions []string

	targetPathPermissions := os.FileMode(0o750)
	if opts.BlockVolume {
		mountOptions = append(mountOptions, "bind")
		if err := os.MkdirAll(filepath.Dir(targetPath), targetPathPermissions); err != nil {
			return err
		}

		mountFilePermissions := os.FileMode(0o660)
		mountFile, err := os.OpenFile(targetPath, os.O_CREATE, mountFilePermissions)
		if err != nil {
			return err
		}
		err = mountFile.Close()
		if err != nil {
			return err
		}
	} else {
		if opts.FSType == "" {
			// BlockVolume is created without file system, setting a default does not make sense
			opts.FSType = DefaultFSType
		}
		if err := os.MkdirAll(targetPath, targetPathPermissions); err != nil {
			return err
		}
	}

	if opts.Readonly {
		mountOptions = append(mountOptions, "ro")
	}

	if opts.EncryptionPassphrase != "" {
		existingFSType, err := s.mounter.GetDiskFormat(devicePath)
		if err != nil {
			return fmt.Errorf("unable to detect existing disk format of %s: %w", devicePath, err)
		}
		luksDeviceName := GenerateLUKSDeviceName(VolumeDevicePath(volumeID))
		if existingFSType == "" {
			if opts.Readonly {
				return fmt.Errorf("cannot publish unformatted disk %s in read-only mode", devicePath)
			}
			if err = s.cryptSetup.Format(ctx, devicePath, opts.EncryptionPassphrase); err != nil {
				return err
			}
		} else if existingFSType != "crypto_LUKS" {
			return fmt.Errorf("requested encrypted volume, but disk %s already is formatted with %s", devicePath, existingFSType)
		}
		if err := s.cryptSetup.Open(ctx, devicePath, luksDeviceName, opts.EncryptionPassphrase); err != nil {
			return err
		}
		luksDevicePath := GenerateLUKSDevicePath(luksDeviceName)
		devicePath = luksDevicePath
	}

	s.logger.Info(
		"publishing volume",
		"volume-id", volumeID,
		"target-path", targetPath,
		"device-path", devicePath,
		"fs-type", opts.FSType,
		"block-volume", opts.BlockVolume,
		"readonly", opts.Readonly,
		"mount-options", strings.Join(mountOptions, ", "),
		"encrypted", opts.EncryptionPassphrase != "",
	)

	if opts.BlockVolume {
		return s.mounter.MountSensitive(devicePath, targetPath, opts.FSType, mountOptions, opts.Additional)
	}

	formatOptions := make([]string, 0)

	if opts.FsFormatOptions != "" {
		lexer := shell.NewLex('\\')
		formatOptions, err = lexer.ProcessWords(opts.FsFormatOptions, shell.EnvsFromSlice([]string{}))
		if err != nil {
			return err
		}
	} else if opts.FSType == "xfs" {
		formatOptions = append(formatOptions, "-c", fmt.Sprintf("options=%s", XFSDefaultConfigPath))
	}

	return s.mounter.FormatAndMountSensitiveWithFormatOptions(devicePath, targetPath, opts.FSType, mountOptions, opts.Additional, formatOptions)
}

func (s *LinuxMountService) waitDeviceReady(ctx context.Context, volumeID string) (string, error) {
	const maxRetries = 7
	backoffFunc := hcloud.ExponentialBackoffWithOpts(hcloud.ExponentialBackoffOpts{
		Base:       time.Millisecond * 50,
		Multiplier: 2.0,
		Cap:        500 * time.Millisecond,
	})

	var err error
	for attempt := range maxRetries {
		logger := s.logger.With(
			"volume-id", volumeID,
			"attempt", attempt+1,
			"max-attempts", maxRetries,
		)

		var devicePath string
		switch devicePath, err = DeviceForVolume(volumeID); {
		case err == nil:
			var stat unix.Stat_t
			err = unix.Stat(devicePath, &stat)
			if err == nil {
				logger.Debug("found device of volume", "device-path", devicePath)
				return devicePath, nil
			}
			if !errors.Is(err, unix.ENOENT) {
				return "", err
			}

			logger.Debug("device node does not exist yet, waiting for devtmpfs", "device-path", devicePath)
		case errors.Is(err, errDeviceNotFound):
			logger.Debug("no device reports the volume yet, waiting for the volume to attach")
		default:
			// Anything else, an unreadable sysfs or two devices claiming the volume, is
			// not going to resolve itself.
			return "", err
		}

		if attempt == maxRetries-1 {
			break
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for the device of volume %s: %w", volumeID, ctx.Err())
		case <-time.After(backoffFunc(attempt)):
		}
	}

	return "", err
}

func (s *LinuxMountService) Unpublish(ctx context.Context, targetPath string) error {
	devicePath, _, err := mount.GetDeviceNameFromMount(mount.New(""), targetPath)
	if err != nil {
		return fmt.Errorf("failed to determine mount path for %s: %w", targetPath, err)
	}

	s.logger.Info(
		"unpublishing volume",
		"target-path", targetPath,
		"device-path", devicePath,
	)

	if err := mount.CleanupMountPoint(targetPath, s.mounter, true); err != nil {
		return err
	}

	luksDeviceName := GenerateLUKSDeviceName(devicePath)

	return s.cryptSetup.Close(ctx, luksDeviceName)
}

func (s *LinuxMountService) PathExists(path string) (bool, error) {
	return mount.PathExists(path)
}
