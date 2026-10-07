package volumes

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

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
