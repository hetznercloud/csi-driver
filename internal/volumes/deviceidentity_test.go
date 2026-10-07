package volumes

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeNode struct {
	t        *testing.T
	devRoot  string
	byIDRoot string
}

func vpdPage(serial string) []byte {
	page := make([]byte, 4, 4+len(serial))
	page[1] = 0x80
	binary.BigEndian.PutUint16(page[2:4], uint16(len(serial))) //nolint:gosec // test serials fit the length field

	return append(page, serial...)
}

func newFakeNode(t *testing.T) *fakeNode {
	t.Helper()

	root := t.TempDir()
	n := &fakeNode{
		t:        t,
		devRoot:  filepath.Join(root, "dev"),
		byIDRoot: filepath.Join(root, "dev", "disk", "by-id"),
	}
	sysRoot := filepath.Join(root, "sys", "class", "block")

	for _, dir := range []string{n.devRoot, n.byIDRoot, sysRoot} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	previousSys := sysClassBlockPath
	sysClassBlockPath = sysRoot
	t.Cleanup(func() { sysClassBlockPath = previousSys })

	return n
}

func (n *fakeNode) attach(device string, page []byte) string {
	n.t.Helper()

	devicePath := filepath.Join(n.devRoot, device)
	if err := os.WriteFile(devicePath, nil, 0o600); err != nil {
		n.t.Fatal(err)
	}

	vpdDir := filepath.Join(sysClassBlockPath, device, "device")
	if err := os.MkdirAll(vpdDir, 0o750); err != nil {
		n.t.Fatal(err)
	}
	if page != nil {
		if err := os.WriteFile(filepath.Join(vpdDir, "vpd_pg80"), page, 0o600); err != nil {
			n.t.Fatal(err)
		}
	}

	return devicePath
}

func (n *fakeNode) byID(volumeID string) string {
	return filepath.Join(n.byIDRoot, "scsi-0HC_Volume_"+volumeID)
}

func (n *fakeNode) link(volumeID string, devicePath string) string {
	n.t.Helper()

	if err := os.Symlink(devicePath, n.byID(volumeID)); err != nil {
		n.t.Fatal(err)
	}

	return n.byID(volumeID)
}

func TestReadVPDPG80(t *testing.T) {
	tests := []struct {
		name    string
		page    []byte
		want    string
		wantErr bool
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
			name:    "page length exceeds the bytes read",
			page:    []byte{0x00, 0x80, 0x00, 0x09, '1', '0', '6'},
			wantErr: true,
		},
		{
			name:    "shorter than the header",
			page:    []byte{0x00, 0x80, 0x00},
			wantErr: true,
		},
		{
			name:    "another vpd page",
			page:    []byte{0x00, 0x83, 0x00, 0x04, 'a', 'b', 'c', 'd'},
			wantErr: true,
		},
		{
			name:    "device without vpd_pg80",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newFakeNode(t).attach("sdc", tt.page)

			got, err := readVPDPG80("sdc")
			if (err != nil) != tt.wantErr {
				t.Fatalf("readVPDPG80() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("readVPDPG80() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveVolumeDevice(t *testing.T) {
	n := newFakeNode(t)

	sdc := n.attach("sdc", vpdPage("106478890"))
	sdd := n.attach("sdd", vpdPage("106486781  "))
	sde := n.attach("sde", nil)

	tests := []struct {
		name       string
		devicePath string
		volumeID   string
		wantPath   string
		wantErr    error
	}{
		{
			name:       "symlink resolves to the volume",
			devicePath: n.link("106478890", sdc),
			volumeID:   "106478890",
			wantPath:   sdc,
		},
		{
			name:       "serial is padded",
			devicePath: n.link("106486781", sdd),
			volumeID:   "106486781",
			wantPath:   sdd,
		},
		{
			name:       "symlink resolves to another volume",
			devicePath: n.link("106486782", sdc),
			volumeID:   "106486782",
			wantErr:    errDeviceMismatch,
		},
		{
			name:       "symlink does not exist yet",
			devicePath: n.byID("106486783"),
			volumeID:   "106486783",
			wantErr:    fs.ErrNotExist,
		},
		{
			name:       "symlink target is gone",
			devicePath: n.link("106486784", filepath.Join(n.devRoot, "sdz")),
			volumeID:   "106486784",
			wantErr:    fs.ErrNotExist,
		},
		{
			name:       "device without vpd_pg80",
			devicePath: n.link("106486785", sde),
			volumeID:   "106486785",
			wantErr:    fs.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devicePath, err := resolveVolumeDevice(tt.devicePath, tt.volumeID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolveVolumeDevice(%s) error = %v, want %v", tt.volumeID, err, tt.wantErr)
			}
			if devicePath != tt.wantPath {
				t.Errorf("resolveVolumeDevice(%s) = %q, want %q", tt.volumeID, devicePath, tt.wantPath)
			}
		})
	}
}
