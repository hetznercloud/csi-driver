package volumes_test

import (
	"os/exec"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/version"

	"github.com/hetznercloud/csi-driver/internal/volumes"
)

func getKernelVersionViaExec() (*version.Version, error) {
	cmd := exec.Command("uname", "-r")
	combinedOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	output := string(combinedOutput)
	output = strings.TrimSpace(output)
	return version.ParseSemantic(output)
}

func TestParseKernelVersion(t *testing.T) {
	kernelVersions := []string{
		"6.8.0",
		"6.8.0-45-generic",
		"6.8.0-rc1",
		"v6.8.0",
		"v6.8.0-45-generic",
		"v6.8.0-rc1",
	}

	correctVersion, err := version.ParseSemantic("6.8.0")
	if err != nil {
		t.Fatal(err)
	}

	for _, kernelVersion := range kernelVersions {
		parsedVersion, err := version.ParseSemantic(kernelVersion)
		if err != nil {
			t.Fatal(err)
		}

		if !correctVersion.AtLeast(parsedVersion) {
			t.Fatalf("Parsed version is not correct: %v\n", parsedVersion)
		}
	}
}

func TestKernelVersion(t *testing.T) {
	unameVersion, err := getKernelVersionViaExec()
	if err != nil {
		t.Fatal(err)
	}

	kernelVersion, err := volumes.GetKernelVersion()
	if err != nil {
		t.Fatal(err)
	}

	if !unameVersion.EqualTo(kernelVersion) {
		t.Fatalf("Versions differ! uname: %s - GetKernelVersion: %s", unameVersion, kernelVersion)
	}
}
