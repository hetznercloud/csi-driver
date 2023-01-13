package integrationtests

import (
	"fmt"
	"io/ioutil"
	"math"
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
		name          string
		passphrase    string
		prepare       func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error
		expectedError error
	}{
		{
			"plain",
			"",
			nil,
			nil,
		},
		{
			"plain-correct-formatted",
			"",
			func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error {
				return svc.FormatDisk(device, "ext4")
			},
			nil,
		},
		{
			"plain-wrong-formatted",
			"",
			func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error {
				return svc.FormatDisk(device, "xfs")
			},
			fmt.Errorf("requested ext4 volume, but disk /dev-fake-plain-wrong-formatted already is formatted with xfs"),
		},
		{
			"encrypted",
			"passphrase",
			nil,
			nil,
		},
		{
			"encrypted-correct-formatted-1",
			"passphrase",
			func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error {
				if err := cs.Format(device, "passphrase"); err != nil {
					return err
				}
				return nil
			},
			nil,
		},
		{
			"encrypted-correct-formatted-2",
			"passphrase",
			func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error {
				if err := cs.Format(device, "passphrase"); err != nil {
					return err
				}
				luksDeviceName := volumes.GenerateLUKSDeviceName(device)
				if err := cs.Open(device, luksDeviceName, "passphrase"); err != nil {
					return err
				}
				defer cs.Close(luksDeviceName)
				luksDevicePath := volumes.GenerateLUKSDevicePath(luksDeviceName)
				if err := svc.FormatDisk(luksDevicePath, "ext4"); err != nil {
					return err
				}
				return nil
			},
			nil,
		},
		{
			"encrypted-wrong-formatted-1",
			"passphrase",
			func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error {
				return svc.FormatDisk(device, "ext4")
			},
			fmt.Errorf("requested encrypted volume, but disk /dev-fake-encrypted-wrong-formatted-1 already is formatted with ext4"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := log.NewLogfmtLogger(NewTestingWriter(t))
			mountService := volumes.NewLinuxMountService(logger)
			cryptSetup := volumes.NewCryptSetup(logger)
			device, err := createFakeDevice("fake-"+test.name, 512)
			if err != nil {
				t.Fatal(err)
			}

			if test.prepare != nil {
				if err := test.prepare(mountService, cryptSetup, device); err != nil {
					t.Fatal(err)
				}
			}

			targetPath, err := ioutil.TempDir(os.TempDir(), "")
			if err != nil {
				t.Fatal()
			}

			if err := mountService.Publish(targetPath, device, volumes.MountOpts{
				EncryptionPassphrase: test.passphrase,
			}); err != nil {
				if test.expectedError == nil {
					t.Fatal(err)
				} else if test.expectedError.Error() != err.Error() {
					t.Fatal(fmt.Errorf("expected error %q but got %q", test.expectedError.Error(), err.Error()))
				}
			}
			defer mountService.Unpublish(targetPath)

			if _, err := ioutil.ReadDir(targetPath); err != nil {
				t.Fatal(err)
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
			resizeService := volumes.NewLinuxResizeService(logger)
			cryptSetup := volumes.NewCryptSetup(logger)
			deviceName := "fake-" + test.name
			device, err := createFakeDevice(deviceName, 512)
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

			if err := increaseFakeDeviceSize(deviceName, 512); err != nil {
				t.Fatal()
			}

			if err := mountService.Publish(targetPath, device, volumes.MountOpts{
				EncryptionPassphrase: test.passphrase,
			}); err != nil {
				t.Fatal(err)
			}
			defer mountService.Unpublish(targetPath)

			initialSize, err := getFakeDeviceSizeKilobytes(targetPath)
			if err != nil {
				t.Fatal(err)
			}

			if err := resizeService.Resize(targetPath); err != nil {
				t.Fatal(err)
			}

			finalSize, err := getFakeDeviceSizeKilobytes(targetPath)
			if err != nil {
				t.Fatal(err)
			}

			if math.Round(float64(finalSize)/float64(initialSize)) != 2.0 {
				t.Fatal(fmt.Errorf("expected final size to be roughly double of initial size (final size %d KB, initial size %d KB)", finalSize, initialSize))
			}

			if err := mountService.Unpublish(targetPath); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDetectDiskFormat(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	tests := []*struct {
		name           string
		prepare        func(*volumes.LinuxMountService, string) error
		expectedFormat string
	}{
		{
			"empty",
			nil,
			"",
		},
		{
			"ext4",
			func(svc *volumes.LinuxMountService, disk string) error {
				return svc.FormatDisk(disk, "ext4")
			},
			"ext4",
		},
		{
			"xfs",
			func(svc *volumes.LinuxMountService, disk string) error {
				return svc.FormatDisk(disk, "xfs")
			},
			"xfs",
		},
		{
			"crypto_LUKS",
			func(svc *volumes.LinuxMountService, disk string) error {
				logger := log.NewLogfmtLogger(NewTestingWriter(t))
				cryptSetup := volumes.NewCryptSetup(logger)
				err := cryptSetup.Format(disk, "passphrase")
				return err
			},
			"crypto_LUKS",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := log.NewLogfmtLogger(NewTestingWriter(t))
			mountService := volumes.NewLinuxMountService(logger)
			disk, err := createFakeDevice(test.name, 512)
			if err != nil {
				t.Fatal(err)
			}
			if test.prepare != nil {
				if err := test.prepare(mountService, disk); err != nil {
					t.Fatal(err)
				}
			}
			format, err := mountService.DetectDiskFormat(disk)
			if err != nil {
				t.Fatal(err)
			}
			if format != test.expectedFormat {
				t.Error(fmt.Errorf("expected format %q, got %q", test.expectedFormat, format))
			}
		})
	}

}
