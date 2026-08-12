package volumes

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// volumeDevicePrefix is the prefix of the by-id path reported as Volume.LinuxDevice.
	// It is a variable so tests can point it at a fixture directory.
	volumeDevicePrefix = "/dev/disk/by-id/scsi-0HC_Volume_"
	// sysClassBlockPath is a variable so tests can point it at a fixture directory.
	sysClassBlockPath = "/sys/class/block"
)

var (
	errDeviceMismatch     = errors.New("device path resolves to a different volume")
	errNotVolume          = errors.New("device is not a hetzner cloud volume")
	errEmptySerial        = errors.New("vpd_pg80: empty serial")
	errAllPadding         = errors.New("vpd_pg80: serial is all padding")
	errShortPage          = errors.New("vpd_pg80: page is shorter than the 4 byte header")
	errWrongPageCode      = errors.New("vpd_pg80: unexpected page code")
	errPageLengthTooLarge = errors.New("vpd_pg80: page length exceeds the bytes read")
)

// VerifyDeviceIdentity checks that devicePath resolves to the device of the volume named
// in the path, by comparing the SCSI serial of the resolved device against the volume ID.
// The by-id symlinks are maintained asynchronously by udev, so a stale one can resolve to
// another volume's device. An identity that cannot be determined is an error: mounting the
// wrong device destroys data, refusing to mount does not.
func VerifyDeviceIdentity(devicePath string) error {
	volumeID, isHCloudVolume := strings.CutPrefix(devicePath, volumeDevicePrefix)
	if !isHCloudVolume {
		return fmt.Errorf("%w: %s", errNotVolume, devicePath)
	}

	resolved, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return err
	}

	serial, err := readDiskSerial(resolved)
	if err != nil {
		return fmt.Errorf("can't verify serial of volume %s: %w", volumeID, err)
	}

	if serial != volumeID {
		return fmt.Errorf(
			"%w: got serial %s for device %s, expected volume %s",
			errDeviceMismatch,
			serial,
			devicePath,
			volumeID,
		)
	}

	return nil
}

// readDiskSerial reads the SCSI serial of a block device from sysfs. dev is either a
// kernel device name ("sdc") or a path to one ("/dev/sdc").
func readDiskSerial(dev string) (string, error) {
	path := filepath.Join(sysClassBlockPath, filepath.Base(dev), "device", "vpd_pg80")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return parseVPDPG80(b)
}

func parseVPDPG80(b []byte) (string, error) {
	if len(b) < 4 {
		return "", fmt.Errorf("%w: got %d bytes", errShortPage, len(b))
	}

	if b[1] != 0x80 {
		return "", fmt.Errorf("%w: 0x%02x", errWrongPageCode, b[1])
	}

	n := int(binary.BigEndian.Uint16(b[2:4]))
	if n == 0 {
		return "", errEmptySerial
	}

	if payloadLen := len(b) - 4; n > payloadLen {
		return "", fmt.Errorf("%w: got %d bytes", errPageLengthTooLarge, payloadLen)
	}

	serial := strings.Trim(string(b[4:4+n]), " \x00")
	if serial == "" {
		return "", errAllPadding
	}

	return serial, nil
}
