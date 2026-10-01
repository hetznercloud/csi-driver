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
	devPath           = "/dev"
	sysClassBlockPath = "/sys/class/block"
)

func SetDeviceIdentityPaths(dev, sysClassBlock string) {
	devPath = dev
	sysClassBlockPath = sysClassBlock
}

var (
	errDeviceNotFound  = errors.New("no block device reports the volume as its serial")
	errAmbiguousDevice = errors.New("multiple block devices report the volume as their serial")
)

func deviceForVolume(volumeID string) (string, error) {
	entries, err := os.ReadDir(sysClassBlockPath)
	if err != nil {
		return "", fmt.Errorf("list block devices: %w", err)
	}

	var found []string
	for _, entry := range entries {
		page, err := os.ReadFile(filepath.Join(sysClassBlockPath, entry.Name(), "device", "vpd_pg80"))
		if err == nil && parseVPDPG80(page) == volumeID {
			found = append(found, entry.Name())
		}
	}

	switch len(found) {
	case 0:
		return "", fmt.Errorf("%w: volume %s", errDeviceNotFound, volumeID)
	case 1:
		return filepath.Join(devPath, found[0]), nil
	default:
		return "", fmt.Errorf("%w: volume %s: %s", errAmbiguousDevice, volumeID, strings.Join(found, ", "))
	}
}

func parseVPDPG80(page []byte) string {
	if len(page) < 4 || page[1] != 0x80 {
		return ""
	}
	n := int(binary.BigEndian.Uint16(page[2:4]))
	if n > len(page)-4 {
		return ""
	}
	return strings.Trim(string(page[4:4+n]), " \x00")
}
