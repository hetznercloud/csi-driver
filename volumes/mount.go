package volumes

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"

	"github.com/hetznercloud/csi-driver/csi"
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
	Stage(volume *csi.Volume, stagingTargetPath string, opts MountOpts) error
	Unstage(volume *csi.Volume, stagingTargetPath string) error
	Publish(volume *csi.Volume, targetPath string, stagingTargetPath string, opts MountOpts) error
	Unpublish(volume *csi.Volume, targetPath string) error
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

func (s *LinuxMountService) Stage(volume *csi.Volume, stagingTargetPath string, opts MountOpts) error {
	if opts.FSType == "" {
		opts.FSType = DefaultFSType
	}

	level.Debug(s.logger).Log(
		"msg", "staging volume",
		"volume-name", volume.Name,
		"staging-target-path", stagingTargetPath,
		"fs-type", opts.FSType,
	)

	isNotMountPoint, err := s.mounter.IsLikelyNotMountPoint(stagingTargetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(stagingTargetPath, 0750); err != nil {
				return err
			}
			isNotMountPoint = true
		} else {
			return err
		}
	}
	if !isNotMountPoint {
		return nil
	}

	return s.mounter.FormatAndMount(volume.LinuxDevice, stagingTargetPath, opts.FSType, nil)
}

func (s *LinuxMountService) Unstage(volume *csi.Volume, stagingTargetPath string) error {
	level.Debug(s.logger).Log(
		"msg", "unstaging volume",
		"volume-name", volume.Name,
		"staging-target-path", stagingTargetPath,
	)
	return mount.CleanupMountPoint(stagingTargetPath, s.mounter, false)
}

func (s *LinuxMountService) Publish(volume *csi.Volume, targetPath string, stagingTargetPath string, opts MountOpts) error {
	isNotMountPoint, err := mount.IsNotMountPoint(s.mounter, targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			isNotMountPoint = true
		} else {
			return err
		}
	}

	if !isNotMountPoint {
		return nil
	}
	targetPathPermissions := os.FileMode(0750)
	if opts.BlockVolume {
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

	options := []string{"bind"}
	if opts.Readonly {
		options = append(options, "ro")
	}
	options = append(options, opts.Additional...)

	level.Debug(s.logger).Log(
		"msg", "publishing volume",
		"volume-name", volume.Name,
		"target-path", targetPath,
		"staging-target-path", stagingTargetPath,
		"fs-type", opts.FSType,
		"block-volume", opts.BlockVolume,
		"readonly", opts.Readonly,
		"mount-options", strings.Join(options, ", "),
	)

	if err := s.mounter.Mount(stagingTargetPath, targetPath, opts.FSType, options); err != nil {
		return err
	}

	return nil
}

func (s *LinuxMountService) Unpublish(volume *csi.Volume, targetPath string) error {
	level.Debug(s.logger).Log(
		"msg", "unpublishing volume",
		"volume-name", volume.Name,
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
