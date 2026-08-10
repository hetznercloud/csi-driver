package volumes

import (
	"bytes"
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
	Publish(ctx context.Context, targetPath string, devicePath string, opts MountOpts) error
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

func (s *LinuxMountService) Publish(ctx context.Context, targetPath string, devicePath string, opts MountOpts) error {
	// Ensure device is ready via stat syscall. Otherwise, `blkid` might return
	// exit code 2, which is the same exit code as for an unformatted device.
	if err := s.waitDeviceReady(ctx, devicePath); err != nil {
		return fmt.Errorf("device %q not ready: %w", devicePath, err)
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
		luksDeviceName := GenerateLUKSDeviceName(devicePath)
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

// HCloudVolumeDevicePrefix is the prefix of the by-id path reported as Volume.LinuxDevice.
const HCloudVolumeDevicePrefix = "/dev/disk/by-id/scsi-0HC_Volume_"

// sysBlockPath is a variable so tests can point it at a fixture directory.
var sysBlockPath = "/sys/block"

var ErrDeviceMismatch = errors.New("device path resolves to a different volume")

// verifyDeviceIdentity checks that devicePath resolves to the device of the volume named
// in the path. The by-id symlinks are maintained asynchronously by udev, so a stale one
// can resolve to another volume's device. Returns nil when the identity is confirmed and
// when it cannot be determined, so an unverifiable mount is never blocked.
func (s *LinuxMountService) verifyDeviceIdentity(devicePath string) error {
	volumeID, isHCloudVolume := strings.CutPrefix(devicePath, HCloudVolumeDevicePrefix)
	if !isHCloudVolume {
		return nil
	}

	resolved, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return nil
	}

	return verifyDeviceSerial(devicePath, resolved, volumeID)
}

// verifyDeviceSerial compares the SCSI serial of the resolved device against the volume ID.
func verifyDeviceSerial(devicePath, resolved, volumeID string) error {
	// VPD page 0x80: a 4 byte header, then the serial, which is the volume ID.
	raw, err := os.ReadFile(filepath.Join(sysBlockPath, filepath.Base(resolved), "device", "vpd_pg80"))
	if err != nil || len(raw) <= 4 {
		return nil
	}

	serial := string(bytes.Trim(raw[4:], "\x00 "))
	if serial != volumeID {
		return fmt.Errorf("%w: %s resolved to %s with serial %q, expected volume %s",
			ErrDeviceMismatch, devicePath, resolved, serial, volumeID)
	}
	return nil
}

// waitDeviceReady ensures the device at devicePath exists and belongs to the volume named
// in the path. Existence is checked via stat, otherwise `blkid` might return exit code 2,
// which is the same exit code as for an unformatted device.
func (s *LinuxMountService) waitDeviceReady(ctx context.Context, devicePath string) error {
	const maxRetries = 7
	backoffFunc := hcloud.ExponentialBackoffWithOpts(hcloud.ExponentialBackoffOpts{
		Base:       time.Millisecond * 50,
		Multiplier: 2.0,
		Cap:        500 * time.Millisecond,
	})

	var err error
	for i := range maxRetries {
		var stat unix.Stat_t
		switch err = unix.Stat(devicePath, &stat); {
		case err == nil:
			if err = s.verifyDeviceIdentity(devicePath); err == nil {
				return nil
			}
			// udev has not finished updating the symlink yet. Retrying is safe,
			// mounting the wrong device is not.
			s.logger.Warn("device not ready yet: path resolves to another volume", "devicePath", devicePath, "error", err)
		case errors.Is(err, unix.ENOENT):
			s.logger.Debug("device not ready yet: stat syscall returned ENOENT", "devicePath", devicePath)
		default:
			return err
		}

		if i == maxRetries-1 {
			break
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for device %s: %w", devicePath, ctx.Err())
		case <-time.After(backoffFunc(i)):
		}
	}

	return err
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
