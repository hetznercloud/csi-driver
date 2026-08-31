//go:build integration

package integration

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync/atomic"
	"testing"

	"k8s.io/mount-utils"

	"github.com/hetznercloud/csi-driver/internal/volumes"
)

const (
	testImageName                = "hcloud-csi-driver-integrationtests"
	testImageEnvironmentVariable = "HCLOUD_CSI_DRIVER_INTEGRATIONTESTS"
)

func TestMain(t *testing.M) {
	if os.Getenv(testImageEnvironmentVariable) != "true" {
		if err := prepareDockerImage(); err != nil {
			log.Fatal(err)
		}
	} else if err := setupDeviceFixtures(); err != nil {
		log.Fatal(err)
	}

	os.Exit(t.Run())
}

func prepareDockerImage() error {
	os.Setenv("GOOS", "linux")
	defer os.Unsetenv("GOOS")
	os.Setenv("GOARCH", "amd64")
	defer os.Unsetenv("GOARCH")
	os.Setenv("CGO_ENABLED", "0")
	defer os.Unsetenv("CGO_ENABLED")
	if output, err := runCmd("go", "test", "-tags", "integration", "-c", "-o", "integration.tests"); err != nil {
		return fmt.Errorf("error compiling test binary: %w\n%s", err, output)
	}

	if output, err := DockerBuild(testImageName, "."); err != nil {
		return fmt.Errorf("error building docker image: %w\n%s", err, output)
	}

	return nil
}

func runTestInDockerImage(t *testing.T, privileged bool) bool { //nolint:unparam
	if os.Getenv(testImageEnvironmentVariable) == "true" {
		return true
	}

	if output, err := DockerRun(testImageName, []string{testImageEnvironmentVariable + "=true"}, []string{"-test.v", "-test.run", t.Name()}, privileged); err != nil {
		err := fmt.Errorf("error running test in docker image: %w\n%s", err, output)
		t.Fatal(err)
	} else {
		t.Log(output)
	}

	return false
}

const deviceFixtureVolumeIDBase = 100000000

var (
	deviceFixtureSysClassBlock string
	deviceFixtureVolumeIDs     atomic.Int64
)

// setupDeviceFixtures points the device lookup at directories the tests can write. A fake
// device is a file: it has no SCSI serial for the driver to find it by, and sysfs, where
// the serial would be, is not writable. The tests report one per fake device instead, so
// that publishing takes the same path it takes on a node.
func setupDeviceFixtures() error {
	root, err := os.MkdirTemp(os.TempDir(), "csi-driver-devices")
	if err != nil {
		return err
	}

	deviceFixtureSysClassBlock = filepath.Join(root, "sys", "class", "block")
	if err := os.MkdirAll(deviceFixtureSysClassBlock, 0o750); err != nil {
		return err
	}

	// The fake devices are files at the root of the container filesystem, so that is
	// where the driver has to look for the device the fixture sysfs reports.
	volumes.SetDeviceIdentityPaths("/", deviceFixtureSysClassBlock)

	return nil
}

// createFakeDevice creates a file standing in for a block device, and the sysfs entry
// reporting the serial of the volume it holds. It returns the device path and that volume
// ID: the driver is asked for the volume and finds the device by its serial, the same way
// it does on a node.
func createFakeDevice(name string, megabytes int) (devicePath string, volumeID string, err error) {
	devicePath = "/dev-" + name
	if _, err := os.Create(devicePath); err != nil {
		return "", "", err
	}
	if _, err := runCmd("dd", "if=/dev/zero", "of="+devicePath, "bs=1M", "count="+strconv.Itoa(megabytes)); err != nil {
		return "", "", err
	}

	volumeID, err = reportFakeDeviceSerial(filepath.Base(devicePath))
	if err != nil {
		return "", "", err
	}

	return devicePath, volumeID, nil
}

// reportFakeDeviceSerial makes the fake device report the serial of a volume, the way the
// kernel does for a SCSI disk. device is the name of the device file, as it appears in
// sysfs.
func reportFakeDeviceSerial(device string) (string, error) {
	volumeID := strconv.FormatInt(deviceFixtureVolumeIDBase+deviceFixtureVolumeIDs.Add(1), 10)

	vpdDir := filepath.Join(deviceFixtureSysClassBlock, device, "device")
	if err := os.MkdirAll(vpdDir, 0o750); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(vpdDir, "vpd_pg80"), vpdPage(volumeID), 0o600); err != nil {
		return "", err
	}

	return volumeID, nil
}

// vpdPage lays out the bytes the kernel exposes as vpd_pg80 for a SCSI disk: a 4 byte
// header whose last two bytes hold the length of the serial that follows.
func vpdPage(serial string) []byte {
	page := make([]byte, 4, 4+len(serial))
	page[1] = 0x80
	binary.BigEndian.PutUint16(page[2:4], uint16(len(serial))) //nolint:gosec // a volume ID is 9 digits

	return append(page, serial...)
}

func increaseFakeDeviceSize(name string, megabytesToAdd int) error {
	path := "/dev-" + name
	if _, err := runCmd("dd", "if=/dev/zero", "of="+path, "bs=1M", "count="+strconv.Itoa(megabytesToAdd), "oflag=append", "conv=notrunc"); err != nil {
		return err
	}
	return nil
}

func getFakeDeviceSizeKilobytes(mountPoint string) (int, error) {
	output, err := runCmd("df", "--output=size", "-k", mountPoint)
	if err != nil {
		return -1, err
	}

	regex := regexp.MustCompile(`(?ms)^\s*1K-blocks\s*(\d+)\s*$`)
	match := regex.FindStringSubmatch(output)
	if match == nil {
		return -1, fmt.Errorf("unexpected df command output")
	}
	size, _ := strconv.Atoi(match[1])
	return size, nil
}

type TestingWriter struct {
	t *testing.T
}

func NewTestingWriter(t *testing.T) TestingWriter {
	return TestingWriter{t: t}
}

func (w TestingWriter) Write(p []byte) (n int, err error) {
	if os.Getenv("TEST_DEBUG_MODE") != "" {
		w.t.Log(string(p))
	}
	return len(p), nil
}

func formatDisk(mounter *mount.SafeFormatAndMount, device string, fstype string) error {
	tmppath, err := os.MkdirTemp(os.TempDir(), "csi-driver-prepare")
	if err != nil {
		return err
	}

	// The library we use to format volumes only supports a combined "FormatAndMount"
	// so we call that and immediately unmount and cleanup the temp path afterwards.
	defer os.RemoveAll(tmppath)
	defer mounter.Unmount(tmppath)
	return mounter.FormatAndMount(device, tmppath, fstype, nil)
}
