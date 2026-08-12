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

// TestPublishRejectsAliasedDevice covers a stale by-id symlink resolving to another
// volume's device. Without the identity check, Publish mounts it, or formats it when
// blkid finds no filesystem, destroying its data (#1346).
func TestPublishRejectsAliasedDevice(t *testing.T) {
	sysRoot := fakeSysClassBlock(t)
	link, byIDPrefix := fakeByID(t)

	// The device of volume 106478890, holding data blkid does not recognise.
	writeVPD(t, sysRoot, "sdc", vpdPage(t, "106478890"))
	payload := bytes.Repeat([]byte("data-of-volume-106478890."), 40)

	// udev has not caught up: the by-id link of volume 106486781 still points at it.
	requestedPath := link("106486781", "sdc")
	deviceOfOtherVolume, err := filepath.EvalSymlinks(requestedPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(deviceOfOtherVolume, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	if requestedPath != byIDPrefix+"106486781" {
		t.Fatalf("fixture path %q does not carry the volume ID", requestedPath)
	}

	s := NewLinuxMountService(slog.New(slog.DiscardHandler))

	err = s.Publish(context.Background(), filepath.Join(t.TempDir(), "mount"), requestedPath, MountOpts{})
	if !errors.Is(err, errDeviceMismatch) {
		t.Fatalf("Publish(%s) = %v, want %v: it mounted the device holding volume 106478890 "+
			"after being asked for volume 106486781", requestedPath, err, errDeviceMismatch)
	}

	after, err := os.ReadFile(deviceOfOtherVolume)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, after) {
		t.Fatal("volume 106478890's data was modified while refusing to publish volume 106486781")
	}
}

// TestPublishRejectsUnverifiableDevice covers a device whose serial cannot be read. The
// identity check is fail-closed, so Publish has to refuse rather than mount blind.
func TestPublishRejectsUnverifiableDevice(t *testing.T) {
	fakeSysClassBlock(t)
	link, _ := fakeByID(t)

	// No vpd_pg80 is written for sdc, so its serial is unknown.
	requestedPath := link("106486781", "sdc")

	s := NewLinuxMountService(slog.New(slog.DiscardHandler))

	err := s.Publish(context.Background(), filepath.Join(t.TempDir(), "mount"), requestedPath, MountOpts{})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Publish(%s) = %v, want %v", requestedPath, err, os.ErrNotExist)
	}
}
