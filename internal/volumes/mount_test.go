package volumes

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

var _ MountService = (*LinuxMountService)(nil)

// writeVPD lays out a fake /sys/block/<device>/device/vpd_pg80 holding serial,
// preceded by the 4 byte VPD page header the kernel emits.
func writeVPD(t *testing.T, root, device, serial string) {
	t.Helper()

	dir := filepath.Join(root, device, "device")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	page := append([]byte{0x00, 0x80, 0x00, byte(len(serial))}, []byte(serial)...)
	if err := os.WriteFile(filepath.Join(dir, "vpd_pg80"), page, 0o644); err != nil {
		t.Fatal(err)
	}
}

// useFakeSysBlock points the serial lookup at a fixture directory.
func useFakeSysBlock(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	previous := sysBlockPath
	sysBlockPath = root
	t.Cleanup(func() { sysBlockPath = previous })

	return root
}

func TestVerifyDeviceSerial(t *testing.T) {
	root := useFakeSysBlock(t)

	writeVPD(t, root, "sdc", "106478890")
	writeVPD(t, root, "sdd", "106486781")
	// A serial padded the way some kernels report it.
	writeVPD(t, root, "sde", "106486782  ")

	tests := []struct {
		name     string
		resolved string
		volumeID string
		wantErr  error
	}{
		{
			name:     "serial matches the requested volume",
			resolved: "/dev/sdc",
			volumeID: "106478890",
		},
		{
			name:     "symlink resolves to another volume",
			resolved: "/dev/sdc",
			volumeID: "106486781",
			wantErr:  ErrDeviceMismatch,
		},
		{
			name:     "padded serial is trimmed before comparing",
			resolved: "/dev/sde",
			volumeID: "106486782",
		},
		{
			name:     "unknown device cannot be verified and is allowed",
			resolved: "/dev/sdz",
			volumeID: "106486781",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyDeviceSerial(HCloudVolumeDevicePrefix+tt.volumeID, tt.resolved, tt.volumeID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyDeviceIdentitySkipsForeignPaths(t *testing.T) {
	useFakeSysBlock(t)

	s := &LinuxMountService{}

	// LUKS mappings and the sanity tests' fake device paths carry no volume ID.
	for _, devicePath := range []string{"devpath", "/dev/mapper/scsi-0HC_Volume_1", "/dev/sdc"} {
		if err := s.verifyDeviceIdentity(devicePath); err != nil {
			t.Errorf("verifyDeviceIdentity(%q) = %v, want nil", devicePath, err)
		}
	}
}

// TestPublishRejectsAliasedDevice covers a stale by-id symlink resolving to another
// volume's device. Without the identity check, Publish mounts it, or formats it when
// blkid finds no filesystem, destroying its data (#1346).
func TestPublishRejectsAliasedDevice(t *testing.T) {
	byID := filepath.Dir(HCloudVolumeDevicePrefix)
	if err := os.MkdirAll(byID, 0o755); err != nil {
		t.Skipf("needs a writable %s: %v", byID, err)
	}

	// The device of volume 106478890, holding data blkid does not recognise.
	deviceOfOtherVolume := "/dev/sdc-aliased"
	payload := bytes.Repeat([]byte("data-of-volume-106478890."), 40)
	if err := os.WriteFile(deviceOfOtherVolume, payload, 0o644); err != nil {
		t.Skipf("needs a writable /dev: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(deviceOfOtherVolume) })

	root := useFakeSysBlock(t)
	writeVPD(t, root, filepath.Base(deviceOfOtherVolume), "106478890")

	requestedPath := HCloudVolumeDevicePrefix + "106486781"
	_ = os.Remove(requestedPath)
	if err := os.Symlink(deviceOfOtherVolume, requestedPath); err != nil {
		t.Skipf("needs a writable /dev: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(requestedPath) })

	s := NewLinuxMountService(slog.Default())

	err := s.Publish(context.Background(), filepath.Join(t.TempDir(), "mount"), requestedPath, MountOpts{})
	if !errors.Is(err, ErrDeviceMismatch) {
		t.Fatalf("Publish(%s) = %v, want %v: it mounted the device holding volume 106478890 "+
			"after being asked for volume 106486781", requestedPath, err, ErrDeviceMismatch)
	}

	after, err := os.ReadFile(deviceOfOtherVolume)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, after) {
		t.Fatalf("volume 106478890's data was modified while refusing to publish volume 106486781")
	}
}
