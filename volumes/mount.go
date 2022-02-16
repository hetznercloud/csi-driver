package volumes

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
)

const DefaultFSType = "ext4"

// MountOpts specifies options for mounting a volume.
type MountOpts struct {
	BlockVolume bool
	FSType      string
	Readonly    bool
	Additional  []string // Additional mount options/flags passed to /bin/mount
}

// MountService mounts volumes.
type MountService interface {
	Publish(targetPath string, devicePath string, opts MountOpts) error
	Unpublish(targetPath string) error
	PathExists(path string) (bool, error)
}

// LinuxMountService mounts volumes on a Linux system.
type LinuxMountService struct {
	logger  log.Logger
	mounter *mount.SafeFormatAndMount
}

func NewLinuxMountService(logger log.Logger) *LinuxMountService {
	return &LinuxMountService{
		logger: logger,
		mounter: &mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      exec.New(),
		},
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

	level.Debug(s.logger).Log(
		"msg", "publishing volume",
		"target-path", targetPath,
		"device-path", devicePath,
		"fs-type", opts.FSType,
		"block-volume", opts.BlockVolume,
		"readonly", opts.Readonly,
		"mount-options", strings.Join(mountOptions, ", "),
	)

	if err := s.mounter.Mount(devicePath, targetPath, opts.FSType, mountOptions); err != nil {
		return err
	}

	return nil
}

func (s *LinuxMountService) Unpublish(targetPath string) error {
	level.Debug(s.logger).Log(
		"msg", "unpublishing volume",
		"target-path", targetPath,
	)
	return mount.CleanupMountPoint(targetPath, s.mounter, true)
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
