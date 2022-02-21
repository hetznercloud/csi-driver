package integrationtests

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/hetznercloud/csi-driver/volumes"
)

func TestVolumePublishUnpublish(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	tests := []struct {
		name       string
		passphrase string
	}{
		{"plain", ""},
		{"encrypted", "passphrase"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := log.NewLogfmtLogger(NewTestingWriter(t))
			mountService := volumes.NewLinuxMountService(logger)
			device, err := createFakeDevice("fake-"+test.name, 32)
			if err != nil {
				t.Fatal(err)
			}
			targetPath, err := ioutil.TempDir(os.TempDir(), "")
			if err != nil {
				t.Fatal()
			}

			if err := mountService.Publish(targetPath, device, volumes.MountOpts{
				EncryptionPassphrase: test.passphrase,
			}); err != nil {
				t.Fatal(err)
			}
			defer mountService.Unpublish(targetPath)

			if files, err := ioutil.ReadDir(targetPath); err != nil {
				t.Fatal(err)
			} else {
				if len(files) != 1 || !files[0].IsDir() || files[0].Name() != "lost+found" {
					t.Fatal("expected an fresh ext4 formatted disk with only lost+found directory")
				}
			}

			if err := mountService.Unpublish(targetPath); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestVolumeResize(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("online resizing only works under Linux")
	}

	if !runTestInDockerImage(t, true) {
		return
	}

	tests := []*struct {
		name        string
		passphrase  string
		initialSize int
		finalSize   int
	}{
		{"plain", "", 26609, 57317},
		{"encrypted", "passphrase", 27761, 58597},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := log.NewLogfmtLogger(NewTestingWriter(t))
			mountService := volumes.NewLinuxMountService(logger)
			resizeService := volumes.NewLinuxResizeService(logger)
			cryptSetup := volumes.NewCryptSetup(logger)
			deviceName := "fake-" + test.name
			device, err := createFakeDevice(deviceName, 32)
			if err != nil {
				t.Fatal(err)
			}
			targetPath, err := ioutil.TempDir(os.TempDir(), "")
			if err != nil {
				t.Fatal()
			}

			if test.passphrase == "" {
				if _, err := runCmd("mkfs.ext4", device); err != nil {
					t.Fatal(err)
				}
			} else {
				decryptedName := test.name + "-decrypted"
				if err := cryptSetup.Format(device, test.passphrase); err != nil {
					t.Fatal()
				}
				if err := cryptSetup.Open(device, decryptedName, test.passphrase); err != nil {
					t.Fatal()
				}
				defer cryptSetup.Close(decryptedName)
				decryptedDevice := "/dev/mapper/" + decryptedName
				if _, err := runCmd("mkfs.ext4", decryptedDevice); err != nil {
					t.Fatal(err)
				}
				if err := cryptSetup.Close(decryptedName); err != nil {
					t.Fatal(err)
				}
			}

			if err := increaseFakeDeviceSize(deviceName, 32); err != nil {
				t.Fatal()
			}

			if err := mountService.Publish(targetPath, device, volumes.MountOpts{
				EncryptionPassphrase: test.passphrase,
			}); err != nil {
				t.Fatal(err)
			}
			defer mountService.Unpublish(targetPath)

			if size, err := getFakeDeviceSizeKilobytes(targetPath); err != nil {
				t.Fatal(err)
			} else if size != test.initialSize {
				t.Error(fmt.Errorf("expected initial size of %d KB, got %d KB", test.initialSize, size))
			}

			if err := resizeService.Resize(targetPath); err != nil {
				t.Fatal(err)
			}

			if size, err := getFakeDeviceSizeKilobytes(targetPath); err != nil {
				t.Fatal(err)
			} else if size != test.finalSize {
				t.Fatal(fmt.Errorf("expected final size of %d KB, got %d KB", test.finalSize, size))
			}

			if err := mountService.Unpublish(targetPath); err != nil {
				t.Fatal(err)
			}
		})
	}
}
