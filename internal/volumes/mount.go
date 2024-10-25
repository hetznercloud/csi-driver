package volumes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
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
	Publish(targetPath string, devicePath string, opts MountOpts) error
	Unpublish(targetPath string) error
	PathExists(path string) (bool, error)
}

// LinuxMountService mounts volumes on a Linux system.
type LinuxMountService struct {
	logger     log.Logger
	mounter    *mount.SafeFormatAndMount
	cryptSetup *CryptSetup
}

func NewLinuxMountService(logger log.Logger) *LinuxMountService {
	return &LinuxMountService{
		logger: logger,
		mounter: &mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      exec.New(),
		},
		cryptSetup: NewCryptSetup(logger),
	}
}

func (s *LinuxMountService) Publish(targetPath string, devicePath string, opts MountOpts) error {
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

	targetPathPermissions := os.FileMode(0750)
	if opts.BlockVolume {
		mountOptions = append(mountOptions, "bind")
		if err := os.MkdirAll(filepath.Dir(targetPath), targetPathPermissions); err != nil {
			return err
		}

		mountFilePermissions := os.FileMode(0660)
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
			if err = s.cryptSetup.Format(devicePath, opts.EncryptionPassphrase); err != nil {
				return err
			}
		} else if existingFSType != "crypto_LUKS" {
			return fmt.Errorf("requested encrypted volume, but disk %s already is formatted with %s", devicePath, existingFSType)
		}
		if err := s.cryptSetup.Open(devicePath, luksDeviceName, opts.EncryptionPassphrase); err != nil {
			return err
		}
		luksDevicePath := GenerateLUKSDevicePath(luksDeviceName)
		devicePath = luksDevicePath
	}

	level.Info(s.logger).Log(
		"msg", "publishing volume",
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

func (s *LinuxMountService) Unpublish(targetPath string) error {
	devicePath, _, err := mount.GetDeviceNameFromMount(mount.New(""), targetPath)
	if err != nil {
		return fmt.Errorf("failed to determine mount path for %s: %s", targetPath, err)
	}

	level.Info(s.logger).Log(
		"msg", "unpublishing volume",
		"target-path", targetPath,
		"device-path", devicePath,
	)

	if err := mount.CleanupMountPoint(targetPath, s.mounter, true); err != nil {
		return err
	}

	luksDeviceName := GenerateLUKSDeviceName(devicePath)

	return s.cryptSetup.Close(luksDeviceName)
}

func (s *LinuxMountService) PathExists(path string) (bool, error) {
	return mount.PathExists(path)
}
