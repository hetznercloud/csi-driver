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
	volumeDevicePrefix = "/dev/disk/by-id/scsi-0HC_Volume_"
	devPath            = "/dev"
	sysClassBlockPath  = "/sys/class/block"
)

func SetDeviceIdentityPaths(dev, sysClassBlock string) {
	devPath = dev
	sysClassBlockPath = sysClassBlock
}

var (
	errDeviceNotFound     = errors.New("no block device reports the volume as its serial")
	errAmbiguousDevice    = errors.New("multiple block devices report the volume as their serial")
	errEmptySerial        = errors.New("vpd_pg80: empty serial")
	errAllPadding         = errors.New("vpd_pg80: serial is all padding")
	errShortPage          = errors.New("vpd_pg80: page is shorter than the 4 byte header")
	errWrongPageCode      = errors.New("vpd_pg80: unexpected page code")
	errPageLengthTooLarge = errors.New("vpd_pg80: page length exceeds the bytes read")
)

func VolumeDevicePath(volumeID string) string {
	return volumeDevicePrefix + volumeID
}

func DeviceForVolume(volumeID string) (string, error) {
	entries, err := os.ReadDir(sysClassBlockPath)
	if err != nil {
		return "", fmt.Errorf("list block devices: %w", err)
	}

	var found []string
	for _, entry := range entries {
		serial, err := readDiskSerial(entry.Name())
		if err != nil {
			continue
		}

		if serial == volumeID {
			found = append(found, entry.Name())
		}
	}

	switch len(found) {
	case 1:
		return filepath.Join(devPath, found[0]), nil
	case 0:
		return "", fmt.Errorf("%w: volume %s", errDeviceNotFound, volumeID)
	default:
		// Two devices claiming the same volume means the node can not tell them apart.
		// Refusing is the only safe answer: mounting the wrong one destroys data.
		return "", fmt.Errorf("%w: volume %s: %s", errAmbiguousDevice, volumeID, strings.Join(found, ", "))
	}
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
