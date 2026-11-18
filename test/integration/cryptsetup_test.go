//go:build integration

package integration

import (
	"log/slog"
	"os"
	"slices"
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

	if err := cryptSetup.Open(ctx, device, decryptedName, passphrase, volumes.LUKSOpenOpts{}); err != nil {
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

	if err := cryptSetup.Open(ctx, staleDevice, luksDeviceName, passphrase, volumes.LUKSOpenOpts{}); err != nil {
		t.Fatal(err)
	}
	defer runCmd("cryptsetup", "luksClose", luksDeviceName)

	for range 2 {
		if err := cryptSetup.Open(ctx, currentDevice, luksDeviceName, passphrase, volumes.LUKSOpenOpts{}); err != nil {
			t.Fatal(err)
		}

		status, err := cryptSetup.Status(luksDeviceName)
		if err != nil {
			t.Fatal(err)
		}
		if !status.Active || !status.BackedBy(currentDevice) {
			t.Fatalf("expected %s to be backed by %s, got %+v", luksDeviceName, currentDevice, status)
		}
	}
}

func TestStaleLUKSMapperInUseIsKept(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cryptSetup := volumes.NewCryptSetup(logger)
	passphrase := "passphrase"
	luksDeviceName := "stale-in-use"

	staleDevice := createLoopDevice(t, "stale-in-use-previous")
	currentDevice := createLoopDevice(t, "stale-in-use-current")
	for _, device := range []string{staleDevice, currentDevice} {
		if err := cryptSetup.Format(ctx, device, passphrase); err != nil {
			t.Fatal(err)
		}
	}

	if err := cryptSetup.Open(ctx, staleDevice, luksDeviceName, passphrase, volumes.LUKSOpenOpts{}); err != nil {
		t.Fatal(err)
	}
	defer runCmd("cryptsetup", "luksClose", luksDeviceName)

	luksDevicePath := volumes.GenerateLUKSDevicePath(luksDeviceName)
	if _, err := runCmd("mkfs.ext4", luksDevicePath); err != nil {
		t.Fatal(err)
	}
	mountPath := t.TempDir()
	if _, err := runCmd("mount", "-t", "ext4", luksDevicePath, mountPath); err != nil {
		t.Fatal(err)
	}
	defer runCmd("umount", mountPath)

	err := cryptSetup.Open(ctx, currentDevice, luksDeviceName, passphrase, volumes.LUKSOpenOpts{})
	if err == nil {
		t.Fatalf("expected opening %s on %s to fail while it is mounted", luksDeviceName, currentDevice)
	}
	t.Log(err)

	status, err := cryptSetup.Status(luksDeviceName)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Active || !status.BackedBy(staleDevice) {
		t.Fatalf("expected %s to still be backed by %s, got %+v", luksDeviceName, staleDevice, status)
	}
}

func TestCryptSetupOpenWorkqueueFlags(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	testCases := []struct {
		name string
		opts volumes.LUKSOpenOpts
		want []string
	}{
		{
			name: "default",
			want: []string{"discards"},
		},
		{
			name: "no-read",
			opts: volumes.LUKSOpenOpts{DisableReadWorkqueue: true},
			want: []string{"discards", "no_read_workqueue"},
		},
		{
			name: "no-write",
			opts: volumes.LUKSOpenOpts{DisableWriteWorkqueue: true},
			want: []string{"discards", "no_write_workqueue"},
		},
		{
			name: "no-read-write",
			opts: volumes.LUKSOpenOpts{DisableReadWorkqueue: true, DisableWriteWorkqueue: true},
			want: []string{"discards", "no_read_workqueue", "no_write_workqueue"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			cryptSetup := volumes.NewCryptSetup(slog.New(slog.NewTextHandler(os.Stdout, nil)))
			passphrase := "passphrase"
			luksDeviceName := "workqueue-" + tc.name

			device := createLoopDevice(t, luksDeviceName)
			if err := cryptSetup.Format(ctx, device, passphrase); err != nil {
				t.Fatal(err)
			}
			if err := cryptSetup.Open(ctx, device, luksDeviceName, passphrase, tc.opts); err != nil {
				t.Fatal(err)
			}
			defer runCmd("cryptsetup", "luksClose", luksDeviceName)

			output, err := runCmd("cryptsetup", "status", luksDeviceName)
			if err != nil {
				t.Fatal(err)
			}
			got := activationFlags(output)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("expected flags %v, got %v\n%s", tc.want, got, output)
			}
		})
	}
}

// activationFlags parses the flags line of `cryptsetup status`, e.g.
// "  flags:   discards no_read_workqueue no_write_workqueue".
func activationFlags(status string) []string {
	for line := range strings.Lines(status) {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "flags:"); ok {
			flags := strings.Fields(value)
			slices.Sort(flags)
			return flags
		}
	}
	return nil
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
