//go:build integration

package integration

import (
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/go-kit/log"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"

	"github.com/hetznercloud/csi-driver/internal/volumes"
)

func TestXFSDefaultConfigFile(t *testing.T) {
	// Testing if the default xfs config exists
	// If this is not the case, then an older kernel version may no longer be supported
	if _, err := os.Stat(volumes.XFSDefaultConfigPath); err != nil {
		t.Fatal(err)
	}
}

func TestVolumePublishUnpublish(t *testing.T) {
	if !runTestInDockerImage(t, true) {
		return
	}

	tests := []struct {
		name          string
		mountOpts     volumes.MountOpts
		prepare       func(mounter *mount.SafeFormatAndMount, cs *volumes.CryptSetup, device string) error
		expectedError error
	}{
		// Block volume not formatted
		{
			name:          "plain",
			mountOpts:     volumes.MountOpts{},
			prepare:       nil,
			expectedError: nil,
		},
		{
			name:      "plain-correct-formatted",
			mountOpts: volumes.MountOpts{},
			prepare: func(mounter *mount.SafeFormatAndMount, cs *volumes.CryptSetup, device string) error {
				return formatDisk(mounter, device, "ext4")
			},
			expectedError: nil,
		},
		{
			name:      "plain-wrong-formatted",
			mountOpts: volumes.MountOpts{},
			prepare: func(mounter *mount.SafeFormatAndMount, cs *volumes.CryptSetup, device string) error {
				return formatDisk(mounter, device, "xfs")
			},
			expectedError: mount.NewMountError(mount.FilesystemMismatch, ""),
		},
		{
			name:          "block-volume",
			mountOpts:     volumes.MountOpts{BlockVolume: true},
			prepare:       nil,
			expectedError: nil,
		},
		{
			name:          "encrypted",
			mountOpts:     volumes.MountOpts{EncryptionPassphrase: "passphrase"},
			prepare:       nil,
			expectedError: nil,
		},
		{
			name:      "encrypted-correct-formatted-1",
			mountOpts: volumes.MountOpts{EncryptionPassphrase: "passphrase"},
			prepare: func(mounter *mount.SafeFormatAndMount, cs *volumes.CryptSetup, device string) error {
				return cs.Format(device, "passphrase")
			},
			expectedError: nil,
		},
		{
			name:      "encrypted-correct-formatted-2",
			mountOpts: volumes.MountOpts{EncryptionPassphrase: "passphrase"},
			prepare: func(mounter *mount.SafeFormatAndMount, cs *volumes.CryptSetup, device string) error {
				if err := cs.Format(device, "passphrase"); err != nil {
					return err
				}

				luksDeviceName := volumes.GenerateLUKSDeviceName(device)
				if err := cs.Open(device, luksDeviceName, "passphrase"); err != nil {
					return err
				}
				defer cs.Close(luksDeviceName)

				luksDevicePath := volumes.GenerateLUKSDevicePath(luksDeviceName)

				return formatDisk(mounter, luksDevicePath, "ext4")
			},
			expectedError: nil,
		},
		{
			name:      "encrypted-wrong-formatted-1",
			mountOpts: volumes.MountOpts{EncryptionPassphrase: "passphrase"},
			prepare: func(mounter *mount.SafeFormatAndMount, cs *volumes.CryptSetup, device string) error {
				return formatDisk(mounter, device, "ext4")
			},
			expectedError: fmt.Errorf("requested encrypted volume, but disk /dev-fake-encrypted-wrong-formatted-1 already is formatted with ext4"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := log.NewLogfmtLogger(NewTestingWriter(t))
			mountService := volumes.NewLinuxMountService(logger)
			mounter := &mount.SafeFormatAndMount{
				Interface: mount.New(""),
				Exec:      exec.New(),
			}
			cryptSetup := volumes.NewCryptSetup(logger)
			device, err := createFakeDevice("fake-"+test.name, 512)
			if err != nil {
				t.Fatal(err)
			}

			if test.prepare != nil {
				if err := test.prepare(mounter, cryptSetup, device); err != nil {
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
			defer func() {
				err := mountService.Unpublish(targetPath)
				if err != nil {
					t.Fatal(err)
				} else {
					t.Logf("Unpublished targetPath %s", targetPath)
				}
			}()

			if test.expectedError != nil {
				if publishErr == nil {
					t.Fatalf("expected error %q but got no error", test.expectedError.Error())
				}

				if got, ok := publishErr.(mount.MountError); ok {
					if expected, ok := test.expectedError.(mount.MountError); ok {
						if got.Type != expected.Type {
							t.Fatalf("Expected Mount Error %s, but got %s", expected.Type, got.Type)
						}
						return
					} else {
						t.Fatalf("Test returned MountError %s, but expected error is not of MountError", got.Type)
					}
				} else if test.expectedError.Error() != publishErr.Error() {
					t.Fatal(fmt.Errorf("expected error %q but got %q", test.expectedError.Error(), publishErr.Error()))
				}
			}

			if err != nil {
				t.Fatal(publishErr)
			}

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
				fsType, err := mounter.GetDiskFormat(device)
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
		prepare        func(*mount.SafeFormatAndMount, string) error
		expectedFormat string
	}{
		{
			name:           "empty",
			prepare:        nil,
			expectedFormat: "",
		},
		{
			name: "ext4",
			prepare: func(mounter *mount.SafeFormatAndMount, device string) error {
				return formatDisk(mounter, device, "ext4")
			},
			expectedFormat: "ext4",
		},
		{
			name: "xfs",
			prepare: func(mounter *mount.SafeFormatAndMount, device string) error {
				return formatDisk(mounter, device, "xfs")
			},
			expectedFormat: "xfs",
		},
		{
			name: "crypto_LUKS",
			prepare: func(mounter *mount.SafeFormatAndMount, device string) error {
				logger := log.NewLogfmtLogger(NewTestingWriter(t))
				cryptSetup := volumes.NewCryptSetup(logger)
				err := cryptSetup.Format(device, "passphrase")
				return err
			},
			expectedFormat: "crypto_LUKS",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mounter := &mount.SafeFormatAndMount{
				Interface: mount.New(""),
				Exec:      exec.New(),
			}
			disk, err := createFakeDevice(test.name, 512)
			if err != nil {
				t.Fatal(err)
			}
			if test.prepare != nil {
				if err := test.prepare(mounter, disk); err != nil {
					t.Fatal(err)
				}
			}
			format, err := mounter.GetDiskFormat(disk)
			if err != nil {
				t.Fatal(err)
			}
			if format != test.expectedFormat {
				t.Error(fmt.Errorf("expected format %q, got %q", test.expectedFormat, format))
			}
		})
	}
}
