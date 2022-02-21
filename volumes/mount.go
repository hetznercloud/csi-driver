package volumes

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
)

const DefaultFSType = "ext4"

// MountOpts specifies options for mounting a volume.
type MountOpts struct {
	BlockVolume          bool
	FSType               string
	Readonly             bool
	Additional           []string // Additional mount options/flags passed to /bin/mount
	EncryptionPassphrase string
}

// MountService mounts volumes.
type MountService interface {
	Publish(targetPath string, devicePath string, opts MountOpts) error
	Unpublish(targetPath string) error
	PathExists(path string) (bool, error)
	FormatDisk(disk string, fstype string) error
	DetectDiskFormat(disk string) (string, error)
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
	isNotMountPoint, err := mount.IsNotMountPoint(s.mounter, targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			isNotMountPoint = true
		} else {
			return err
		}
	}

	var mountOptions []string

	if !isNotMountPoint {
		return nil
	}
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
		_ = mountFile.Close()
	} else {
		if opts.FSType == "" {
			opts.FSType = DefaultFSType
		}
		if err := os.MkdirAll(targetPath, targetPathPermissions); err != nil {
			return err
		}
	}

	if opts.Readonly {
		mountOptions = append(mountOptions, "ro")
	}
	mountOptions = append(mountOptions, opts.Additional...)

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

	if opts.EncryptionPassphrase != "" {
		existingFSType, err := s.DetectDiskFormat(devicePath)
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

	existingFSType, err := s.DetectDiskFormat(devicePath)
	if err != nil {
		return fmt.Errorf("unable to detect existing disk format of %s: %w", devicePath, err)
	}
	if existingFSType == "" {
		if opts.Readonly {
			return fmt.Errorf("cannot publish unformatted disk %s in read-only mode", devicePath)
		}
		if err = s.FormatDisk(devicePath, opts.FSType); err != nil {
			return err
		}
	} else if existingFSType != opts.FSType {
		return fmt.Errorf("requested %s volume, but disk %s already is formatted with %s", opts.FSType, devicePath, existingFSType)
	}

	if err := s.mounter.Mount(devicePath, targetPath, opts.FSType, mountOptions); err != nil {
		return err
	}

	return nil
}

func (s *LinuxMountService) Unpublish(targetPath string) error {
	devicePath, _, err := mount.GetDeviceNameFromMount(mount.New(""), targetPath)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to determine mount path for %s: %s", targetPath, err))
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
	if err := s.cryptSetup.Close(luksDeviceName); err != nil {
		return err
	}

	return nil
}

func (s *LinuxMountService) PathExists(path string) (bool, error) {
	level.Debug(s.logger).Log(
		"msg", "checking path existence",
		"path", path,
	)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *LinuxMountService) FormatDisk(disk string, fstype string) error {
	level.Info(s.logger).Log(
		"msg", "formatting disk",
		"disk", disk,
		"fstype", fstype,
	)
	switch fstype {
	case "ext4":
		_, _, err := command("mkfs.ext4", "-F", "-m0", disk)
		return err
	case "xfs":
		_, _, err := command("mkfs.xfs", disk)
		return err
	default:
		return fmt.Errorf("unsupported disk format %s", fstype)
	}
}

// see https://github.com/kubernetes/mount-utils/blob/master/mount_linux.go
func (s *LinuxMountService) DetectDiskFormat(disk string) (string, error) {
	args := []string{"-p", "-s", "TYPE", "-s", "PTTYPE", "-o", "export", disk}
	output, exitCode, err := command("blkid", args...)
	if exitCode == 2 {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	fstypeRegex := regexp.MustCompile(`TYPE=(.*)`)
	pttypeRegex := regexp.MustCompile(`PTTYPE=(.*)`)
	fstype := ""
	pttype := ""
	fstypeMatch := fstypeRegex.FindStringSubmatch(output)
	if fstypeMatch != nil {
		fstype = fstypeMatch[1]
	}
	pttypeMatch := pttypeRegex.FindStringSubmatch(output)
	if pttypeMatch != nil {
		pttype = pttypeMatch[1]
	}

	if pttype != "" {
		return "", fmt.Errorf("disk %s propably contains partitions", disk)
	}

	return fstype, nil
}
