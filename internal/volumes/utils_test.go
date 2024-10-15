package volumes_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/version"
)

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
