package integrationtests

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
)

const testImageName = "hcloud-csi-driver-integrationtests"
const testImageEnvironmentVariable = "HCLOUD_CSI_DRIVER_INTEGRATIONTESTS"

func TestMain(t *testing.M) {
	if os.Getenv(testImageEnvironmentVariable) != "true" {
		prepareDockerImage()
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
	if output, err := runCmd("go", "test", ".", "-c", "-o", "integrationtests.tests"); err != nil {
		fmt.Printf("Error compiling test binary: %v\n%s\n", err, output)
		os.Exit(1)
	}

	if output, err := DockerBuild(testImageName, "."); err != nil {
		fmt.Printf("Error building docker image: %v\n%s\n", err, output)
		os.Exit(1)
	}

	return nil
}

func runTestInDockerImage(t *testing.T, privileged bool) bool {
	if os.Getenv(testImageEnvironmentVariable) == "true" {
		return true
	}

	if output, err := DockerRun(testImageName, []string{testImageEnvironmentVariable + "=true"}, []string{"-test.v", "-test.run", t.Name()}, privileged); err != nil {
		err := fmt.Errorf("Error running test in docker image: %w\n%s\n", err, output)
		t.Fatal(err)
	} else {
		t.Log(output)
	}

	return false
}

func createFakeDevice(name string, megabytes int) (string, error) {
	path := "/dev-" + name
	if _, err := os.Create(path); err != nil {
		return "", err
	}
	if _, err := runCmd("dd", "if=/dev/zero", "of="+path, "bs=1M", "count="+strconv.Itoa(megabytes)); err != nil {
		return "", err
	}
	return path, nil
}

func increaseFakeDeviceSize(name string, megabytesToAdd int) error {
	path := "/dev-" + name
	if _, err := runCmd("dd", "if=/dev/zero", "of="+path, "bs=1M", "count="+strconv.Itoa(megabytesToAdd), "oflag=append", "conv=notrunc"); err != nil {
		return err
	}
	return nil
}

func getFakeDeviceSizeKilobytes(mountPoint string) (int, error) {
	if output, err := runCmd("df", "--output=size", "-k", mountPoint); err != nil {
		return -1, err
	} else {
		regex := regexp.MustCompile(`(?ms)^\s*1K-blocks\s*(\d+)\s*$`)
		match := regex.FindStringSubmatch(output)
		if match == nil {
			return -1, fmt.Errorf("unexpected df command output")
		}
		size, _ := strconv.Atoi(match[1])
		return size, nil
	}
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
