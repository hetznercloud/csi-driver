package volumes_test

import (
	"testing"

	"github.com/hetznercloud/csi-driver/internal/volumes"
)

func TestParseKernelVersion(t *testing.T) {
	kernelVersions := []string{
		"6.8",
		"6.8.0",
		"6.8.0-45",
		"6.8.0-45-generic",
	}

	correctVersion := volumes.NewKernelVersion(6, 8, 0)

	for _, version := range kernelVersions {
		parsedVersion, err := volumes.ParseKernelVersion(version)
		if err != nil {
			t.Fatal(err)
		}

		if *parsedVersion != *correctVersion {
			t.Fatalf("Parsed version is not correct: %v\n", parsedVersion)
		}
	}
}

func TestIsNewerThan(t *testing.T) {
	newerVersion := volumes.NewKernelVersion(6, 8, 0)
	olderVersion := volumes.NewKernelVersion(6, 1, 0)

	if !newerVersion.IsNewerThan(olderVersion) {
		t.Fatalf("IsNewerThan returned false result for newer version: %v and older version %v\n", newerVersion, olderVersion)
	}
}
