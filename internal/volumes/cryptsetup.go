package volumes

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const cryptsetupExecutable = "cryptsetup"

// Device mapper devices are always virtual block devices.
var sysVirtualBlockPath = "/sys/devices/virtual/block"

// Name of the sysfs directory in which the kernel links the backing devices of
// a device mapper device.
const sysfsBackingDevicesDir = "slaves"

type CryptSetup struct {
	logger *slog.Logger
}

func NewCryptSetup(logger *slog.Logger) *CryptSetup {
	return &CryptSetup{logger: logger}
}

func (cs *CryptSetup) Status(luksDeviceName string) (LUKSDeviceStatus, error) {
	dmPath, err := findDMDevice(luksDeviceName)
	if err != nil {
		return LUKSDeviceStatus{}, fmt.Errorf("unable to check LUKS device %s status: %w", luksDeviceName, err)
	}
	if dmPath == "" {
		return LUKSDeviceStatus{Active: false}, nil
	}

	status, err := readLUKSDeviceStatus(dmPath)
	if err != nil {
		return LUKSDeviceStatus{}, fmt.Errorf("unable to check LUKS device %s status: %w", luksDeviceName, err)
	}
	return status, nil
}

func (cs *CryptSetup) Format(ctx context.Context, devicePath string, passphrase string) error {
	cs.logger.Info(
		"formatting LUKS device",
		"devicePath", devicePath,
	)
	output, _, err := cryptsetupWithStdin(ctx, passphrase, "luksFormat", "--type", "luks1", devicePath)
	if err != nil {
		return fmt.Errorf("unable to format device %s with LUKS: %s", devicePath, output)
	}
	return nil
}

func (cs *CryptSetup) Open(ctx context.Context, devicePath string, luksDeviceName string, passphrase string) error {
	status, err := cs.Status(luksDeviceName)
	if err != nil {
		return err
	}
	if status.Active {
		// Check that devicePath resolves to the block device the device mapper maps
		// onto, by comparing its major:minor number with the one the kernel reports
		// for the mapper's backing device in sysfs.
		if status.BackedBy(devicePath) {
			cs.logger.Debug(
				"LUKS device already active",
				"devicePath", devicePath,
				"luksDeviceName", luksDeviceName,
			)
			return nil
		}

		// A volume detached without being unpublished leaves behind a mapper whose backing
		// device is gone. Reusing it would format and mount a dead device.
		cs.logger.Warn(
			"LUKS device is not backed by the expected device, closing it",
			"devicePath", devicePath,
			"luksDeviceName", luksDeviceName,
			"backingDevice", status.Device,
			"backingDeviceNumber", status.DeviceNumber,
		)
		if err := cs.Close(ctx, luksDeviceName); err != nil {
			return fmt.Errorf(
				"LUKS device %s is backed by %q instead of %s: %w",
				luksDeviceName,
				status.Device,
				devicePath,
				err,
			)
		}
	}

	cs.logger.Info(
		"opening LUKS device",
		"devicePath", devicePath,
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := cryptsetupWithStdin(ctx, passphrase, "luksOpen", "--allow-discards", devicePath, luksDeviceName)
	if err != nil {
		return fmt.Errorf("unable to open LUKS device %s: %s", devicePath, output)
	}
	return nil
}

func (cs *CryptSetup) Close(ctx context.Context, luksDeviceName string) error {
	status, err := cs.Status(luksDeviceName)
	if err != nil {
		return err
	}
	if !status.Active {
		return nil
	}
	cs.logger.Info(
		"closing LUKS device",
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := cryptsetup(ctx, "luksClose", luksDeviceName)
	if err != nil {
		return fmt.Errorf("unable to close LUKS device %s: %s", luksDeviceName, output)
	}
	return nil
}

func (cs *CryptSetup) Resize(ctx context.Context, luksDeviceName string) error {
	cs.logger.Info(
		"resizing LUKS device",
		"luksDeviceName", luksDeviceName,
	)
	output, _, err := cryptsetup(ctx, "resize", luksDeviceName)
	if err != nil {
		return fmt.Errorf("unable to resize LUKS device %s: %s", luksDeviceName, output)
	}
	return nil
}

func GenerateLUKSDeviceName(devicePath string) string {
	segments := strings.Split(devicePath, "/")
	return segments[len(segments)-1]
}

func GenerateLUKSDevicePath(luksDeviceName string) string {
	return "/dev/mapper/" + luksDeviceName
}

func cryptsetup(ctx context.Context, args ...string) (string, int, error) {
	return cryptsetupWithStdin(ctx, "", args...)
}

func cryptsetupWithStdin(ctx context.Context, stdin string, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, cryptsetupExecutable, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	if err != nil {
		exitError := &exec.ExitError{}
		if !errors.As(err, &exitError) {
			return output, 0, err
		}
		return output, exitError.ExitCode(), fmt.Errorf("%w\n%s", exitError, output)
	}
	return output, 0, nil
}

type LUKSDeviceStatus struct {
	// Active reports whether a device mapper device with the LUKS device name
	// exists. It says nothing about whether its backing device is still present.
	Active bool
	// Device is the kernel name, e.g. "sdb".
	Device string
	// DeviceNumber is the major:minor of the backing device, e.g. "8:16".
	// Empty if the backing device is gone.
	DeviceNumber string
}

// BackedBy reports whether devicePath is the backing device of the LUKS device.
func (s LUKSDeviceStatus) BackedBy(devicePath string) bool {
	if s.DeviceNumber == "" {
		return false
	}
	number, err := blockDeviceNumber(devicePath)
	return err == nil && number == s.DeviceNumber
}

// findDMDevice returns the sysfs directory of the device mapper device with the
// given name, or an empty string if no such device exists.
func findDMDevice(name string) (string, error) {
	// sysfs only indexes device mapper devices by kernel name (dm-N), so scan
	// all of them for a matching mapper name.
	dmPaths, err := filepath.Glob(filepath.Join(sysVirtualBlockPath, "dm-*"))
	if err != nil {
		return "", err
	}
	for _, dmPath := range dmPaths {
		content, err := os.ReadFile(filepath.Join(dmPath, "dm", "name"))
		// The device may be removed between Glob and ReadFile.
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(string(content)) == name {
			return dmPath, nil
		}
	}
	return "", nil
}

// readLUKSDeviceStatus reads the backing device of an active device mapper
// device. The kernel keeps a backing device entry until the mapping is removed,
// so a hot-unplugged backing device either leaves a dangling entry or none at all.
func readLUKSDeviceStatus(dmPath string) (LUKSDeviceStatus, error) {
	backingPath := filepath.Join(dmPath, sysfsBackingDevicesDir)
	backingDevices, err := os.ReadDir(backingPath)
	if err != nil {
		return LUKSDeviceStatus{}, err
	}

	status := LUKSDeviceStatus{Active: true}
	// dm-crypt maps exactly one device
	if len(backingDevices) != 1 {
		return status, nil
	}
	status.Device = backingDevices[0].Name()

	number, err := os.ReadFile(filepath.Join(backingPath, status.Device, "dev"))
	if errors.Is(err, fs.ErrNotExist) {
		return status, nil
	}
	if err != nil {
		return LUKSDeviceStatus{}, err
	}
	status.DeviceNumber = strings.TrimSpace(string(number))
	return status, nil
}

// blockDeviceNumber returns the major:minor number of a block device, in the
// format used by the dev attribute in sysfs.
func blockDeviceNumber(path string) (string, error) {
	var stat unix.Stat_t
	if err := unix.Stat(path, &stat); err != nil {
		return "", err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFBLK {
		return "", fmt.Errorf("%s is not a block device", path)
	}
	return fmt.Sprintf("%d:%d", unix.Major(stat.Rdev), unix.Minor(stat.Rdev)), nil
}
