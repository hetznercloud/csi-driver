package volumes

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeSysfs struct {
	t           *testing.T
	virtualRoot string
	devicesRoot string
}

func newFakeSysfs(t *testing.T) *fakeSysfs {
	t.Helper()

	root := t.TempDir()
	s := &fakeSysfs{
		t:           t,
		virtualRoot: filepath.Join(root, "devices", "virtual", "block"),
		devicesRoot: filepath.Join(root, "devices", "pci", "block"),
	}
	for _, dir := range []string{s.virtualRoot, s.devicesRoot} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	previous := sysVirtualBlockPath
	sysVirtualBlockPath = s.virtualRoot
	t.Cleanup(func() { sysVirtualBlockPath = previous })

	return s
}

func (s *fakeSysfs) write(path string, content string) {
	s.t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content+"\n"), 0o600); err != nil {
		s.t.Fatal(err)
	}
}

func (s *fakeSysfs) blockDevice(name string, number string) {
	s.t.Helper()

	s.write(filepath.Join(s.devicesRoot, name, "dev"), number)
}

func (s *fakeSysfs) dmDevice(kernelName string, name string, backingDevices ...string) {
	s.t.Helper()

	dmPath := filepath.Join(s.virtualRoot, kernelName)
	s.write(filepath.Join(dmPath, "dm", "name"), name)

	backingPath := filepath.Join(dmPath, sysfsBackingDevicesDir)
	if err := os.MkdirAll(backingPath, 0o750); err != nil {
		s.t.Fatal(err)
	}
	for _, backingDevice := range backingDevices {
		if err := os.Symlink(filepath.Join(s.devicesRoot, backingDevice), filepath.Join(backingPath, backingDevice)); err != nil {
			s.t.Fatal(err)
		}
	}
}

func TestCryptSetupStatus(t *testing.T) {
	testCases := []struct {
		Name  string
		Setup func(s *fakeSysfs)
		Want  LUKSDeviceStatus
	}{
		{
			Name:  "no device mapper devices",
			Setup: func(*fakeSysfs) {},
			Want:  LUKSDeviceStatus{Active: false},
		},
		{
			Name: "other device mapper device",
			Setup: func(s *fakeSysfs) {
				s.blockDevice("sdb", "8:16")
				s.dmDevice("dm-0", "pv-2", "sdb")
			},
			Want: LUKSDeviceStatus{Active: false},
		},
		{
			Name: "healthy device",
			Setup: func(s *fakeSysfs) {
				s.blockDevice("sdb", "8:16")
				s.blockDevice("sdc", "8:32")
				s.dmDevice("dm-0", "pv-2", "sdb")
				s.dmDevice("dm-1", "pv-1", "sdc")
			},
			Want: LUKSDeviceStatus{Active: true, Device: "sdc", DeviceNumber: "8:32"},
		},
		{
			Name: "backing device removed, link dangling",
			Setup: func(s *fakeSysfs) {
				s.dmDevice("dm-0", "pv-1", "sdb")
			},
			Want: LUKSDeviceStatus{Active: true, Device: "sdb"},
		},
		{
			Name: "backing device removed, link removed",
			Setup: func(s *fakeSysfs) {
				s.dmDevice("dm-0", "pv-1")
			},
			Want: LUKSDeviceStatus{Active: true},
		},
		{
			Name: "multiple backing devices",
			Setup: func(s *fakeSysfs) {
				s.blockDevice("sdb", "8:16")
				s.blockDevice("sdc", "8:32")
				s.dmDevice("dm-0", "pv-1", "sdb", "sdc")
			},
			Want: LUKSDeviceStatus{Active: true},
		},
	}

	cryptSetup := NewCryptSetup(slog.New(slog.DiscardHandler))
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			testCase.Setup(newFakeSysfs(t))

			got, err := cryptSetup.Status("pv-1")
			if err != nil {
				t.Fatal(err)
			}
			if got != testCase.Want {
				t.Errorf("Status() = %+v, want %+v", got, testCase.Want)
			}
		})
	}
}

func TestLUKSDeviceStatusBackedBy(t *testing.T) {
	regularFile := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(regularFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		Name       string
		Status     LUKSDeviceStatus
		DevicePath string
	}{
		{
			Name:       "backing device gone",
			Status:     LUKSDeviceStatus{Active: true, Device: "sdb"},
			DevicePath: "/dev/null",
		},
		{
			Name:       "character device",
			Status:     LUKSDeviceStatus{Active: true, Device: "null", DeviceNumber: "1:3"},
			DevicePath: "/dev/null",
		},
		{
			Name:       "regular file",
			Status:     LUKSDeviceStatus{Active: true, Device: "sdb", DeviceNumber: "8:16"},
			DevicePath: regularFile,
		},
		{
			Name:       "missing device",
			Status:     LUKSDeviceStatus{Active: true, Device: "sdb", DeviceNumber: "8:16"},
			DevicePath: filepath.Join(t.TempDir(), "missing"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			if testCase.Status.BackedBy(testCase.DevicePath) {
				t.Errorf("%+v.BackedBy(%q) = true, want false", testCase.Status, testCase.DevicePath)
			}
		})
	}
}

func TestBlockDeviceNumberMatchesSysfs(t *testing.T) {
	entries, err := os.ReadDir("/sys/class/block")
	if err != nil {
		t.Skip(err)
	}
	for _, entry := range entries {
		devicePath := filepath.Join("/dev", entry.Name())
		if _, err := os.Stat(devicePath); err != nil {
			continue
		}
		want, err := os.ReadFile(filepath.Join("/sys/class/block", entry.Name(), "dev"))
		if err != nil {
			t.Fatal(err)
		}

		got, err := blockDeviceNumber(devicePath)
		if err != nil {
			t.Fatal(err)
		}
		if got != strings.TrimSpace(string(want)) {
			t.Errorf("blockDeviceNumber(%q) = %q, want %q", devicePath, got, want)
		}

		status := LUKSDeviceStatus{Active: true, Device: entry.Name(), DeviceNumber: got}
		if !status.BackedBy(devicePath) {
			t.Errorf("%+v.BackedBy(%q) = false, want true", status, devicePath)
		}
		return
	}
	t.Skip("no block device found")
}
