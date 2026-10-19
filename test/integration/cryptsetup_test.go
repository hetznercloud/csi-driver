//go:build integration

package integration

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/hetznercloud/csi-driver/internal/volumes"
)

func TestCryptSetup(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cryptSetup := volumes.NewCryptSetup(logger)
	name := "fake"
	device, _, err := createFakeDevice(name, 32)
	if err != nil {
		t.Fatal(err)
	}
	passphrase := "passphrase"

	if err := cryptSetup.Format(ctx, device, passphrase); err != nil {
		t.Fatal(err)
	}
	decryptedName := name + "-decrypted"

	if err := cryptSetup.Open(ctx, device, decryptedName, passphrase); err != nil {
		t.Fatal(err)
	}
	decryptedDevice := "/dev/mapper/" + decryptedName
	defer runCmd("cryptsetup", "luksClose", decryptedName)

	if _, err := runCmd("mkfs.ext4", decryptedDevice); err != nil {
		t.Fatal(err)
	}
	decryptedMount := "/mnt/" + name
	if err := os.MkdirAll(decryptedMount, 0o775); err != nil {
		t.Fatal(err)
	}
	if _, err := runCmd("mount", "-t", "ext4", decryptedDevice, decryptedMount); err != nil {
		t.Fatal(err)
	}
	defer runCmd("umount", decryptedMount)

	if _, err := runCmd("umount", decryptedMount); err != nil {
		t.Fatal(err)
	}
	if err := cryptSetup.Close(ctx, decryptedName); err != nil {
		t.Fatal(err)
	}
}

func TestStaleLUKSMapperIsReplaced(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cryptSetup := volumes.NewCryptSetup(logger)
	passphrase := "passphrase"
	luksDeviceName := "stale"

	staleDevice := createLoopDevice(t, "stale-previous")
	currentDevice := createLoopDevice(t, "stale-current")
	for _, device := range []string{staleDevice, currentDevice} {
		if err := cryptSetup.Format(ctx, device, passphrase); err != nil {
			t.Fatal(err)
		}
	}

	if err := cryptSetup.Open(ctx, staleDevice, luksDeviceName, passphrase); err != nil {
		t.Fatal(err)
	}
	defer runCmd("cryptsetup", "luksClose", luksDeviceName)

	for range 2 {
		if err := cryptSetup.Open(ctx, currentDevice, luksDeviceName, passphrase); err != nil {
			t.Fatal(err)
		}

		status, err := cryptSetup.Status(ctx, luksDeviceName)
		if err != nil {
			t.Fatal(err)
		}
		if !status.Active || status.Device != currentDevice {
			t.Fatalf("expected %s to be backed by %s, got %+v", luksDeviceName, currentDevice, status)
		}
	}
}

func createLoopDevice(t *testing.T, name string) string {
	t.Helper()

	file := "/dev-" + name
	if _, err := runCmd("dd", "if=/dev/zero", "of="+file, "bs=1M", "count=32"); err != nil {
		t.Fatal(err)
	}

	output, err := runCmd("losetup", "-f")
	if err != nil {
		t.Fatal(err)
	}
	device := strings.TrimSpace(output)
	if _, err := runCmd("losetup", device, file); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = runCmd("losetup", "-d", device) })

	return device
}
