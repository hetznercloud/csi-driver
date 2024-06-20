package integration

import (
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/go-kit/log"

	"github.com/hetznercloud/csi-driver/internal/volumes"
)

func TestVolumePublishUnpublish(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	tests := []struct {
		name          string
		mountOpts     volumes.MountOpts
		prepare       func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error
		expectedError error
	}{
		// Block volume not formatted
		{
			"plain",
			volumes.MountOpts{},
			nil,
			nil,
		},
		{
			"plain-correct-formatted",
			volumes.MountOpts{},
			func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error {
				return svc.FormatDisk(device, "ext4")
			},
			nil,
		},
		{
			"plain-wrong-formatted",
			volumes.MountOpts{},
			func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error {
				return svc.FormatDisk(device, "xfs")
			},
			fmt.Errorf("requested ext4 volume, but disk /dev-fake-plain-wrong-formatted already is formatted with xfs"),
		},
		{
			"block-volume",
			volumes.MountOpts{BlockVolume: true},
			nil,
			nil,
		},
		{
			"encrypted",
			volumes.MountOpts{EncryptionPassphrase: "passphrase"},
			nil,
			nil,
		},
		{
			"encrypted-correct-formatted-1",
			volumes.MountOpts{EncryptionPassphrase: "passphrase"},
			func(svc volumes.MountService, cs *volumes.CryptSetup, device string) error {
				return cs.Format(device, "passphrase")
			},
			nil,
		},
		{
			"encrypted-correct-formatted-2",
			volumes.MountOpts{EncryptionPassphrase: "passphrase"},
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

				return svc.FormatDisk(luksDevicePath, "ext4")
			},
			nil,
		},
		{
			"encrypted-wrong-formatted-1",
			volumes.MountOpts{EncryptionPassphrase: "passphrase"},
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

			targetPath, err := os.MkdirTemp(os.TempDir(), "csi-driver")
			if err != nil {
				t.Fatal()
			}
			// Make sure that target path is non-existent
			// Required as FS volumes require target dir, but block volumes require
			// target file
			targetPath = path.Join(targetPath, "target-path")

			publishErr := mountService.Publish(targetPath, device, test.mountOpts)
			if test.expectedError != nil {
				// We expected an error
				if publishErr == nil {
					t.Fatalf("expected error %q but got no error", test.expectedError.Error())
				} else if test.expectedError.Error() != publishErr.Error() {
					t.Fatal(fmt.Errorf("expected error %q but got %q", test.expectedError.Error(), publishErr.Error()))
				}

				// Makes no sense to continue verification if we got the error that we expected
				_ = mountService.Unpublish(targetPath)
				return
			}
			if err != nil {
				t.Fatal(publishErr)
			}
			defer func() {
				err := mountService.Unpublish(targetPath)
				if err != nil {
					t.Fatal(err)
				}
			}()

			// Verify target exists and is of expected type
			fileInfo, err := os.Stat(targetPath)
			if err != nil {
				t.Fatal(err)
			}
			isDir := fileInfo.IsDir()

			if test.mountOpts.BlockVolume && isDir {
				t.Fatal("targetPath expected to be a file for block volumes, but is a directory")
			}

			if !test.mountOpts.BlockVolume && !isDir {
				t.Fatal("targetPath expected to be a directory for fs volumes, but is a file")
			}

			if test.mountOpts.EncryptionPassphrase == "" {
				// Verify device has expected fs type
				// Encrypted volumes always have "crypto_LUKS"
				fsType, err := mountService.DetectDiskFormat(device)
				if err != nil {
					t.Fatal(err)
				}
				expectedFSType := test.mountOpts.FSType
				if expectedFSType == "" && !test.mountOpts.BlockVolume {
					// ext4 is default fs type
					expectedFSType = "ext4"
				}
				if fsType != expectedFSType {
					t.Fatalf("expected device to have fs type '%s', but device is formatted with '%s'", expectedFSType, fsType)
				}

				if err := mountService.Unpublish(targetPath); err != nil {
					t.Fatal(err)
				}
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
			targetPath, err := os.MkdirTemp(os.TempDir(), "")
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
