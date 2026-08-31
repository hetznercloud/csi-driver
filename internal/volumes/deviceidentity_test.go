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

func fakeNode(t *testing.T) (attach func(device string, page []byte) string, devRoot string) {
	t.Helper()

	root := t.TempDir()
	devRoot = filepath.Join(root, "dev")
	sysRoot := filepath.Join(root, "sys", "class", "block")

	for _, dir := range []string{devRoot, sysRoot} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	previousDev, previousSys := devPath, sysClassBlockPath
	devPath, sysClassBlockPath = devRoot, sysRoot
	t.Cleanup(func() { devPath, sysClassBlockPath = previousDev, previousSys })

	attach = func(device string, page []byte) string {
		t.Helper()

		devicePath := filepath.Join(devRoot, device)
		if err := os.WriteFile(devicePath, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		writeVPD(t, sysRoot, device, page)

		return devicePath
	}

	return attach, devRoot
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

// writeVPD creates the sysfs entry reporting serial for device. A nil page leaves the
// entry without a vpd_pg80, as a device that is not a SCSI disk has.
func writeVPD(t *testing.T, root, device string, page []byte) {
	t.Helper()

	dir := filepath.Join(root, device, "device")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if page == nil {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "vpd_pg80"), page, 0o600); err != nil {
		t.Fatal(err)
	}
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

	// Callers pass a device path as well as a bare kernel device name.
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

func TestDeviceForVolume(t *testing.T) {
	attach, devRoot := fakeNode(t)

	attach("sdc", vpdPage(t, "106478890"))
	attach("sdd", vpdPage(t, "106486781  "))
	// A device without a readable serial is never a match, even if it is the only one.
	attach("sde", nil)
	attach("sdf", vpdPage(t, ""))
	// Two devices reporting the same volume: the node can not tell them apart.
	attach("sdg", vpdPage(t, "106486799"))
	attach("sdh", vpdPage(t, "106486799"))
	// A partition has no device directory at all.
	if err := os.MkdirAll(filepath.Join(sysClassBlockPath, "sdc1"), 0o750); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		volumeID string
		wantPath string
		wantErr  error
	}{
		{
			name:     "device reporting the volume",
			volumeID: "106478890",
			wantPath: filepath.Join(devRoot, "sdc"),
		},
		{
			name:     "serial is padded",
			volumeID: "106486781",
			wantPath: filepath.Join(devRoot, "sdd"),
		},
		{
			name:     "no device reports the volume",
			volumeID: "106486782",
			wantErr:  errDeviceNotFound,
		},
		{
			name:     "two devices report the volume",
			volumeID: "106486799",
			wantErr:  errAmbiguousDevice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devicePath, err := DeviceForVolume(tt.volumeID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DeviceForVolume(%s) error = %v, want %v", tt.volumeID, err, tt.wantErr)
			}
			if devicePath != tt.wantPath {
				t.Errorf("DeviceForVolume(%s) = %q, want %q", tt.volumeID, devicePath, tt.wantPath)
			}
		})
	}
}

func TestDeviceForVolumeIgnoresByIDLinks(t *testing.T) {
	attach, devRoot := fakeNode(t)

	attach("sdc", vpdPage(t, "106478890"))
	attach("sdd", vpdPage(t, "106486781"))

	// udev points the by-id link of volume 106486781 at the device of volume 106478890,
	// and has not created one for volume 106478890 at all.
	byID := filepath.Join(t.TempDir(), "by-id")
	if err := os.MkdirAll(byID, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(devRoot, "sdc"), filepath.Join(byID, "scsi-0HC_Volume_106486781")); err != nil {
		t.Fatal(err)
	}

	for volumeID, want := range map[string]string{"106478890": "sdc", "106486781": "sdd"} {
		devicePath, err := DeviceForVolume(volumeID)
		if err != nil {
			t.Fatalf("DeviceForVolume(%s) = %v", volumeID, err)
		}
		if devicePath != filepath.Join(devRoot, want) {
			t.Errorf("DeviceForVolume(%s) = %q, want the device %q", volumeID, devicePath, want)
		}
	}
}

func TestDeviceForVolumeNamesTheVolume(t *testing.T) {
	fakeNode(t)

	_, err := DeviceForVolume("106486781")
	if err == nil {
		t.Fatal("DeviceForVolume() = nil, want an error")
	}
	if !strings.Contains(err.Error(), "106486781") {
		t.Errorf("error %q does not mention volume 106486781", err)
	}
}
