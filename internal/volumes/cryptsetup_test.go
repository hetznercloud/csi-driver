package volumes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLUKSDeviceStatus(t *testing.T) {
	testCases := []struct {
		Name   string
		Output string
		Want   LUKSDeviceStatus
	}{
		{
			Name: "healthy device",
			Output: `/dev/mapper/pv-1 is active and is in use.
  type:    LUKS1
  cipher:  aes-xts-plain64
  keysize: 512 bits
  device:  /dev/sdb
  sector size:  512
  offset:  4096 sectors
  size:    2093056 sectors
  mode:    read/write`,
			Want: LUKSDeviceStatus{Active: true, Device: "/dev/sdb"},
		},
		{
			Name: "backing device disappeared",
			Output: `/dev/mapper/pv-1 is active.
  type:    n/a
  cipher:  aes-xts-plain64
  keysize: 512 bits
  device:  (null)
  mode:    read/write`,
			Want: LUKSDeviceStatus{Active: true, Device: "(null)"},
		},
		{
			Name: "device line missing",
			Output: `/dev/mapper/pv-1 is active.
  type:    LUKS1
  cipher:  aes-xts-plain64`,
			Want: LUKSDeviceStatus{Active: true},
		},
		{
			Name:   "empty output",
			Output: "",
			Want:   LUKSDeviceStatus{Active: true},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			if got := NewLUKSDeviceStatus(testCase.Output); got != testCase.Want {
				t.Errorf("NewLUKSDeviceStatus() = %+v, want %+v", got, testCase.Want)
			}
		})
	}
}

func TestSameDevice(t *testing.T) {
	regularFile := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(regularFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		Name string
		A    string
		B    string
		Want bool
	}{
		{
			Name: "same device",
			A:    "/dev/null",
			B:    "/dev/null",
			Want: true,
		},
		{
			Name: "different devices",
			A:    "/dev/null",
			B:    "/dev/zero",
			Want: false,
		},
		{
			Name: "backing device disappeared",
			A:    "(null)",
			B:    "/dev/null",
			Want: false,
		},
		{
			Name: "backing device missing",
			A:    "",
			B:    "/dev/null",
			Want: false,
		},
		{
			Name: "regular files",
			A:    regularFile,
			B:    regularFile,
			Want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			if got := sameDevice(testCase.A, testCase.B); got != testCase.Want {
				t.Errorf("sameDevice(%q, %q) = %v, want %v", testCase.A, testCase.B, got, testCase.Want)
			}
		})
	}
}
