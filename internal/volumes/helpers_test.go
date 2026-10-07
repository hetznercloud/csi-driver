package volumes

import (
	"encoding/binary"
	"os"
	"path/filepath"
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
