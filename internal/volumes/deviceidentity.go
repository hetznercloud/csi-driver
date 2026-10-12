package volumes

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var sysClassBlockPath = "/sys/class/block"

// Integrations test helper
func SetSysClassBlockPath(sysClassBlock string) {
	sysClassBlockPath = sysClassBlock
}

var errDeviceMismatch = errors.New("device belongs to another volume")

func resolveVolumeDevice(devicePath string, volumeID string) (string, error) {
	resolved, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return "", err
	}

	serial, err := readVPDPG80(filepath.Base(resolved))
	if err != nil {
		return "", fmt.Errorf("volume %s: %w", volumeID, err)
	}

	if serial != volumeID {
		return "", fmt.Errorf(
			"volume %s: %w: %s resolves to %s, which reports serial %s",
			volumeID,
			errDeviceMismatch,
			devicePath,
			resolved,
			serial,
		)
	}

	return resolved, nil
}

// Unit Serial Number VPD page (page code 80h), as defined in T10.
// Linux exposes the raw page, header included, at
// /sys/class/block/<block>/device/vpd_pg80.
//
//	Byte 0     Peripheral qualifier (bits 7-5) and peripheral device type (bits 4-0)
//	Byte 1     Page code, always 0x80
//	Bytes 2-3  Page length: number of bytes after the header (big-endian)
//	Bytes 4-n  Product serial number: vendor-specific ASCII
//
// See utility: https://linux.die.net/man/8/sg_vpd
// See standard: https://www.t10.org/
func readVPDPG80(block string) (string, error) {
	vpdPG80 := filepath.Join(sysClassBlockPath, block, "device", "vpd_pg80")
	page, err := os.ReadFile(vpdPG80)
	if err != nil {
		return "", err
	}

	if len(page) < 4 {
		return "", fmt.Errorf("%s: page too short (%d bytes)", vpdPG80, len(page))
	}

	if page[1] != 0x80 {
		return "", fmt.Errorf("%s: unexpected page code 0x%02x", vpdPG80, page[1])
	}

	n := int(binary.BigEndian.Uint16(page[2:4]))
	if n > len(page)-4 {
		return "", fmt.Errorf("%s: page length %d exceeds %d bytes read", vpdPG80, n, len(page)-4)
	}

	return strings.Trim(string(page[4:4+n]), " \x00"), nil
}
