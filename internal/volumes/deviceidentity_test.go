package volumes

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type attachFunc func(device string, page []byte) string

func vpdPage(serial string) []byte {
	page := make([]byte, 4, 4+len(serial))
	page[1] = 0x80
	binary.BigEndian.PutUint16(page[2:4], uint16(len(serial))) //nolint:gosec // test serials fit the length field

	return append(page, serial...)
}

func fakeNode(t *testing.T) attachFunc {
	t.Helper()

	root := t.TempDir()
	devRoot := filepath.Join(root, "dev")
	sysRoot := filepath.Join(root, "sys", "class", "block")

	for _, dir := range []string{devRoot, sysRoot} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	previousDev, previousSys := devPath, sysClassBlockPath
	devPath, sysClassBlockPath = devRoot, sysRoot
	t.Cleanup(func() { devPath, sysClassBlockPath = previousDev, previousSys })

	return func(device string, page []byte) string {
		t.Helper()

		devicePath := filepath.Join(devRoot, device)
		if err := os.WriteFile(devicePath, nil, 0o600); err != nil {
			t.Fatal(err)
		}

		vpdDir := filepath.Join(sysRoot, device, "device")
		if err := os.MkdirAll(vpdDir, 0o750); err != nil {
			t.Fatal(err)
		}
		if page != nil {
			if err := os.WriteFile(filepath.Join(vpdDir, "vpd_pg80"), page, 0o600); err != nil {
				t.Fatal(err)
			}
		}

		return devicePath
	}
}

func TestParseVPDPG80(t *testing.T) {
	tests := []struct {
		name string
		page []byte
		want string
	}{
		{
			name: "serial of an hcloud volume",
			page: vpdPage("106478890"),
			want: "106478890",
		},
		{
			name: "padding is trimmed",
			page: vpdPage(" 106478890 \x00\x00"),
			want: "106478890",
		},
		{
			name: "page length needs both length bytes",
			page: vpdPage(strings.Repeat("\x00", 300) + "106478890"),
			want: "106478890",
		},
		{
			name: "bytes past the page length are ignored",
			page: append(vpdPage("106478890"), "trailing junk"...),
			want: "106478890",
		},
		{
			name: "page length exceeds the bytes read",
			page: []byte{0x00, 0x80, 0x00, 0x09, '1', '0', '6'},
		},
		{
			name: "shorter than the header",
			page: []byte{0x00, 0x80, 0x00},
		},
		{
			name: "another vpd page",
			page: []byte{0x00, 0x83, 0x00, 0x04, 'a', 'b', 'c', 'd'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseVPDPG80(tt.page); got != tt.want {
				t.Errorf("parseVPDPG80() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeviceForVolume(t *testing.T) {
	attach := fakeNode(t)

	sdc := attach("sdc", vpdPage("106478890"))
	sdd := attach("sdd", vpdPage("106486781  "))
	attach("sde", nil)
	attach("sdf", vpdPage(""))
	attach("sdg", vpdPage("106486799"))
	attach("sdh", vpdPage("106486799"))
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
			wantPath: sdc,
		},
		{
			name:     "serial is padded",
			volumeID: "106486781",
			wantPath: sdd,
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
			devicePath, err := deviceForVolume(tt.volumeID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("deviceForVolume(%s) error = %v, want %v", tt.volumeID, err, tt.wantErr)
			}
			if devicePath != tt.wantPath {
				t.Errorf("deviceForVolume(%s) = %q, want %q", tt.volumeID, devicePath, tt.wantPath)
			}
		})
	}
}

func TestDeviceForVolumeNamesTheVolume(t *testing.T) {
	fakeNode(t)

	_, err := deviceForVolume("106486781")
	if err == nil {
		t.Fatal("deviceForVolume() = nil, want an error")
	}
	if !strings.Contains(err.Error(), "106486781") {
		t.Errorf("error %q does not mention volume 106486781", err)
	}
}
