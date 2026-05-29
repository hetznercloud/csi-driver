package volumes

import "testing"

func TestLUKSDeviceStatusIsZombie(t *testing.T) {
	testCases := []struct {
		Name   string
		Status LUKSDeviceStatus
		Want   bool
	}{
		{
			Name:   "inactive is never a zombie",
			Status: LUKSDeviceStatus{Active: false},
			Want:   false,
		},
		{
			Name:   "inactive with missing fields is never a zombie",
			Status: LUKSDeviceStatus{Active: false, Type: "", Device: ""},
			Want:   false,
		},
		{
			Name:   "healthy active device",
			Status: LUKSDeviceStatus{Active: true, Type: "LUKS1", Device: "/dev/sdb"},
			Want:   false,
		},
		{
			Name:   "active with empty device",
			Status: LUKSDeviceStatus{Active: true, Type: "LUKS1", Device: ""},
			Want:   true,
		},
		{
			Name:   "active with empty type",
			Status: LUKSDeviceStatus{Active: true, Type: "", Device: "/dev/sdb"},
			Want:   true,
		},
		{
			Name:   "active with (null) device",
			Status: LUKSDeviceStatus{Active: true, Type: "LUKS1", Device: "(null)"},
			Want:   true,
		},
		{
			Name:   "active with n/a type",
			Status: LUKSDeviceStatus{Active: true, Type: "n/a", Device: "/dev/sdb"},
			Want:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			if got := testCase.Status.IsZombie(); got != testCase.Want {
				t.Errorf("IsZombie() = %v, want %v", got, testCase.Want)
			}
		})
	}
}

func TestParseLUKSStatus(t *testing.T) {
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
			Want: LUKSDeviceStatus{Active: true, Type: "LUKS1", Device: "/dev/sdb"},
		},
		{
			Name: "stale device with (null) backing",
			Output: `/dev/mapper/pv-1 is active.
  type:    n/a
  cipher:  aes-xts-plain64
  keysize: 512 bits
  device:  (null)
  mode:    read/write`,
			Want: LUKSDeviceStatus{Active: true, Type: "n/a", Device: "(null)"},
		},
		{
			Name: "device line missing entirely",
			Output: `/dev/mapper/pv-1 is active.
  type:    LUKS1
  cipher:  aes-xts-plain64`,
			Want: LUKSDeviceStatus{Active: true, Type: "LUKS1", Device: ""},
		},
		{
			Name:   "empty output",
			Output: "",
			Want:   LUKSDeviceStatus{Active: true},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			got := parseLUKSStatus(testCase.Output)
			if got != testCase.Want {
				t.Errorf("parseLUKSStatus() = %+v, want %+v", got, testCase.Want)
			}
		})
	}
}
