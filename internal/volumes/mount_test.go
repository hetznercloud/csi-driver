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

func TestWaitDeviceReadyPicksTheDeviceOfTheVolume(t *testing.T) {
	attach, devRoot := fakeNode(t)

	attach("sdc", vpdPage(t, "106478890"))
	attach("sdd", vpdPage(t, "106486781"))

	s := NewLinuxMountService(slog.New(slog.DiscardHandler))

	devicePath, err := s.waitDeviceReady(context.Background(), "106486781")
	if err != nil {
		t.Fatalf("waitDeviceReady() = %v, want the device of volume 106486781", err)
	}
	if want := filepath.Join(devRoot, "sdd"); devicePath != want {
		t.Errorf("waitDeviceReady() = %q, want %q", devicePath, want)
	}
}

func TestPublishRefusesWithoutAnIdentifiedDevice(t *testing.T) {
	attach, _ := fakeNode(t)

	// The device of volume 106478890, holding data blkid does not recognise.
	deviceOfOtherVolume := attach("sdc", vpdPage(t, "106478890"))
	payload := bytes.Repeat([]byte("data-of-volume-106478890."), 40)
	if err := os.WriteFile(deviceOfOtherVolume, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	// A device whose serial can not be read is not an identified device either.
	attach("sdd", nil)

	s := NewLinuxMountService(slog.New(slog.DiscardHandler))

	err := s.Publish(context.Background(), filepath.Join(t.TempDir(), "mount"), "106486781", MountOpts{})
	if !errors.Is(err, errDeviceNotFound) {
		t.Fatalf("Publish() = %v, want %v: it mounted a device after being asked for a volume "+
			"no device reports", err, errDeviceNotFound)
	}

	after, err := os.ReadFile(deviceOfOtherVolume)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, after) {
		t.Fatal("volume 106478890's data was modified while refusing to publish volume 106486781")
	}
}

func TestPublishRefusesAmbiguousDevice(t *testing.T) {
	attach, _ := fakeNode(t)

	attach("sdc", vpdPage(t, "106486781"))
	attach("sdd", vpdPage(t, "106486781"))

	s := NewLinuxMountService(slog.New(slog.DiscardHandler))

	err := s.Publish(context.Background(), filepath.Join(t.TempDir(), "mount"), "106486781", MountOpts{})
	if !errors.Is(err, errAmbiguousDevice) {
		t.Fatalf("Publish() = %v, want %v", err, errAmbiguousDevice)
	}
}

func TestPublishRefusesWhenTheDeviceNodeIsMissing(t *testing.T) {
	_, devRoot := fakeNode(t)

	writeVPD(t, sysClassBlockPath, "sdc", vpdPage(t, "106486781"))

	s := NewLinuxMountService(slog.New(slog.DiscardHandler))

	err := s.Publish(context.Background(), filepath.Join(t.TempDir(), "mount"), "106486781", MountOpts{})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Publish() = %v, want %v", err, os.ErrNotExist)
	}
	if _, err := os.Stat(filepath.Join(devRoot, "sdc")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture created a device node: %v", err)
	}
}

func TestPublishStopsWhenTheContextIsCancelled(t *testing.T) {
	fakeNode(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewLinuxMountService(slog.New(slog.DiscardHandler))

	err := s.Publish(ctx, filepath.Join(t.TempDir(), "mount"), "106486781", MountOpts{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Publish() = %v, want %v", err, context.Canceled)
	}
}
