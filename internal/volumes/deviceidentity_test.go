package volumes

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vpdPage lays out the bytes the kernel exposes as vpd_pg80: a 4 byte header whose last
// two bytes hold the length of the serial that follows.
func vpdPage(t *testing.T, serial string) []byte {
	t.Helper()

	if len(serial) > math.MaxUint16 {
		t.Fatalf("test serial of %d bytes does not fit the vpd page length field", len(serial))
	}

	page := make([]byte, 4, 4+len(serial))
	page[1] = 0x80
	binary.BigEndian.PutUint16(page[2:4], uint16(len(serial))) //nolint:gosec // bounds-checked above

	return append(page, serial...)
}

// fakeSysClassBlock points the serial lookup at a fixture directory and returns it.
// It swaps a package level variable, so its callers must not run in parallel.
func fakeSysClassBlock(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	previous := sysClassBlockPath
	sysClassBlockPath = root
	t.Cleanup(func() { sysClassBlockPath = previous })

	return root
}

// writeVPD creates the sysfs entry reporting serial for device.
func writeVPD(t *testing.T, root, device string, page []byte) {
	t.Helper()

	dir := filepath.Join(root, device, "device")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vpd_pg80"), page, 0o600); err != nil {
		t.Fatal(err)
	}
}

// fakeByID points the by-id prefix at a fixture directory, so the tests need neither a
// writable /dev nor a writable /sys. It returns a function creating a by-id symlink to a
// device file, mirroring what udev maintains on a node. It swaps a package level variable,
// so its callers must not run in parallel.
func fakeByID(t *testing.T) (link func(volumeID, device string) string, byIDPrefix string) {
	t.Helper()

	root := t.TempDir()
	previous := volumeDevicePrefix
	volumeDevicePrefix = filepath.Join(root, "by-id", "scsi-0HC_Volume_")
	t.Cleanup(func() { volumeDevicePrefix = previous })

	if err := os.MkdirAll(filepath.Join(root, "by-id"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "dev"), 0o750); err != nil {
		t.Fatal(err)
	}

	link = func(volumeID, device string) string {
		t.Helper()

		devicePath := filepath.Join(root, "dev", device)
		if err := os.WriteFile(devicePath, nil, 0o600); err != nil {
			t.Fatal(err)
		}

		byIDPath := volumeDevicePrefix + volumeID
		if err := os.Symlink(devicePath, byIDPath); err != nil {
			t.Fatal(err)
		}
		return byIDPath
	}

	return link, volumeDevicePrefix
}

func TestParseVPDPG80(t *testing.T) {
	tests := []struct {
		name       string
		page       []byte
		wantSerial string
		wantErr    error
	}{
		{
			name:       "serial of an hcloud volume",
			page:       vpdPage(t, "106478890"),
			wantSerial: "106478890",
		},
		{
			name:       "trailing spaces are trimmed",
			page:       vpdPage(t, "106478890  "),
			wantSerial: "106478890",
		},
		{
			name:       "trailing NUL bytes are trimmed",
			page:       vpdPage(t, "106478890\x00\x00"),
			wantSerial: "106478890",
		},
		{
			name:       "page length shorter than the payload wins",
			page:       append(vpdPage(t, "106478890"), []byte("trailing junk")...),
			wantSerial: "106478890",
		},
		{
			// A page longer than 255 bytes needs both length bytes to be read.
			name:       "serial padded past a single length byte",
			page:       vpdPage(t, strings.Repeat("\x00", 300)+"106478890"),
			wantSerial: "106478890",
		},
		{
			name:    "empty page",
			page:    nil,
			wantErr: errShortPage,
		},
		{
			name:    "header only, truncated",
			page:    []byte{0x00, 0x80, 0x00},
			wantErr: errShortPage,
		},
		{
			name:    "another vpd page",
			page:    []byte{0x00, 0x83, 0x00, 0x04, 'a', 'b', 'c', 'd'},
			wantErr: errWrongPageCode,
		},
		{
			name:    "no serial reported",
			page:    vpdPage(t, ""),
			wantErr: errEmptySerial,
		},
		{
			name:    "serial is only padding",
			page:    vpdPage(t, " \x00 "),
			wantErr: errAllPadding,
		},
		{
			name:    "page length exceeds the bytes read",
			page:    []byte{0x00, 0x80, 0x00, 0x09, '1', '0', '6'},
			wantErr: errPageLengthTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serial, err := parseVPDPG80(tt.page)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseVPDPG80() error = %v, want %v", err, tt.wantErr)
				}
				if serial != "" {
					t.Errorf("ParseVPDPG80() serial = %q, want empty on error", serial)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseVPDPG80() error = %v, want nil", err)
			}
			if serial != tt.wantSerial {
				t.Errorf("ParseVPDPG80() serial = %q, want %q", serial, tt.wantSerial)
			}
		})
	}
}

func TestReadDiskSerial(t *testing.T) {
	root := fakeSysClassBlock(t)
	writeVPD(t, root, "sdc", vpdPage(t, "106478890"))

	t.Run("kernel device name", func(t *testing.T) {
		serial, err := readDiskSerial("sdc")
		if err != nil {
			t.Fatal(err)
		}
		if serial != "106478890" {
			t.Errorf("ReadDiskSerial() = %q, want %q", serial, "106478890")
		}
	})

	// VerifyDeviceIdentity passes the path EvalSymlinks resolved to, not a bare name.
	t.Run("resolved device path", func(t *testing.T) {
		serial, err := readDiskSerial("/dev/sdc")
		if err != nil {
			t.Fatal(err)
		}
		if serial != "106478890" {
			t.Errorf("ReadDiskSerial() = %q, want %q", serial, "106478890")
		}
	})

	t.Run("device without a vpd_pg80", func(t *testing.T) {
		if _, err := readDiskSerial("sdz"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("ReadDiskSerial() error = %v, want %v", err, os.ErrNotExist)
		}
	})

	t.Run("unparsable page", func(t *testing.T) {
		writeVPD(t, root, "sdd", []byte{0x00, 0x80})
		if _, err := readDiskSerial("sdd"); !errors.Is(err, errShortPage) {
			t.Fatalf("ReadDiskSerial() error = %v, want %v", err, errShortPage)
		}
	})
}

func TestVerifyDeviceIdentity(t *testing.T) {
	sysRoot := fakeSysClassBlock(t)
	link, byIDPrefix := fakeByID(t)

	writeVPD(t, sysRoot, "sdc", vpdPage(t, "106478890"))
	writeVPD(t, sysRoot, "sdd", vpdPage(t, "106486781  "))
	writeVPD(t, sysRoot, "sdf", vpdPage(t, ""))

	// udev pointed the by-id link of volume 106486782 at the device of volume 106478890.
	staleLink := link("106486782", "sdc")
	matching := link("106486781", "sdd")
	noSerial := link("106486783", "sdf")
	noVPD := link("106486784", "sde")

	dangling := byIDPrefix + "106486785"
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone"), dangling); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		devicePath string
		wantErr    error
	}{
		{
			name:       "serial matches the requested volume",
			devicePath: matching,
		},
		{
			name:       "by-id link resolves to another volume's device",
			devicePath: staleLink,
			wantErr:    errDeviceMismatch,
		},
		{
			name:       "device is not an hcloud volume",
			devicePath: "/dev/mapper/scsi-0HC_Volume_106478890",
			wantErr:    errNotVolume,
		},
		{
			name:       "device path carries no volume id",
			devicePath: "devpath",
			wantErr:    errNotVolume,
		},
		{
			name:       "by-id link does not exist",
			devicePath: byIDPrefix + "106486786",
			wantErr:    os.ErrNotExist,
		},
		{
			name:       "by-id link resolves to a missing device",
			devicePath: dangling,
			wantErr:    os.ErrNotExist,
		},
		{
			name:       "device reports no serial",
			devicePath: noSerial,
			wantErr:    errEmptySerial,
		},
		{
			name:       "device exposes no vpd_pg80",
			devicePath: noVPD,
			wantErr:    os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyDeviceIdentity(tt.devicePath)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("VerifyDeviceIdentity(%s) = %v, want %v", tt.devicePath, err, tt.wantErr)
			}
		})
	}
}

// TestVerifyDeviceIdentityNamesTheVolume covers the message an operator sees when a mount
// is refused: it has to name both volumes to be actionable.
func TestVerifyDeviceIdentityNamesTheVolume(t *testing.T) {
	sysRoot := fakeSysClassBlock(t)
	link, _ := fakeByID(t)

	writeVPD(t, sysRoot, "sdc", vpdPage(t, "106478890"))
	staleLink := link("106486781", "sdc")

	err := VerifyDeviceIdentity(staleLink)
	if err == nil {
		t.Fatal("VerifyDeviceIdentity() = nil, want an error")
	}
	for _, want := range []string{"106478890", "106486781"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention volume %s", err, want)
		}
	}
}
