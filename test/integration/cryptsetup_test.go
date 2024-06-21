package integration

import (
	"os"
	"testing"

	"github.com/go-kit/log"

	"github.com/hetznercloud/csi-driver/internal/volumes"
)

func TestCryptSetup(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	logger := log.NewLogfmtLogger(NewTestingWriter(t))
	cryptSetup := volumes.NewCryptSetup(logger)
	name := "fake"
	device, err := createFakeDevice(name, 32)
	if err != nil {
		t.Fatal(err)
	}
	passphrase := "passphrase"

	if err := cryptSetup.Format(device, passphrase); err != nil {
		t.Fatal(err)
	}
	decryptedName := name + "-decrypted"

	if err := cryptSetup.Open(device, decryptedName, passphrase); err != nil {
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
	if err := cryptSetup.Close(decryptedName); err != nil {
		t.Fatal(err)
	}
}
