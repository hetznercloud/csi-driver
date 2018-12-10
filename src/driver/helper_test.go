package driver

import (
	"testing"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
)

const GB = 1024 * 1024 * 1024

func TestVolumeSizeFromRequest(t *testing.T) {
	testCases := []struct {
		Name    string
		CR      *proto.CapacityRange
		MinSize int
		MaxSize int
		OK      bool
	}{
		{
			Name:    "without capacity range",
			CR:      nil,
			MinSize: DefaultVolumeSize,
			MaxSize: 0,
			OK:      true,
		},
		{
			Name:    "empty capacity range",
			CR:      &proto.CapacityRange{},
			MinSize: DefaultVolumeSize,
			MaxSize: 0,
			OK:      true,
		},
		{
			Name: "with required bytes (exactly 5 GB)",
			CR: &proto.CapacityRange{
				RequiredBytes: 5 * GB,
			},
			MinSize: 5,
			MaxSize: 0,
			OK:      true,
		},
		{
			Name: "with required bytes (slightly more than 5 GB)",
			CR: &proto.CapacityRange{
				RequiredBytes: 5*GB + 100,
			},
			MinSize: 6,
			MaxSize: 0,
			OK:      true,
		},
		{
			Name: "with required and limit bytes",
			CR: &proto.CapacityRange{
				RequiredBytes: 5*GB + 100,
				LimitBytes:    10 * GB,
			},
			MinSize: 6,
			MaxSize: 10,
			OK:      true,
		},
		{
			Name: "with required and limit bytes (same value)",
			CR: &proto.CapacityRange{
				RequiredBytes: 10 * GB,
				LimitBytes:    10 * GB,
			},
			MinSize: 10,
			MaxSize: 10,
			OK:      true,
		},
		{
			Name: "with lower limit than required",
			CR: &proto.CapacityRange{
				RequiredBytes: 10 * GB,
				LimitBytes:    1 * GB,
			},
			MinSize: 0,
			MaxSize: 0,
			OK:      false,
		},
		{
			Name: "with invalid required bytes",
			CR: &proto.CapacityRange{
				RequiredBytes: -10 * GB,
				LimitBytes:    1 * GB,
			},
			MinSize: 0,
			MaxSize: 0,
			OK:      false,
		},
		{
			Name: "with invalid limit bytes",
			CR: &proto.CapacityRange{
				RequiredBytes: 1 * GB,
				LimitBytes:    -10 * GB,
			},
			MinSize: 0,
			MaxSize: 0,
			OK:      false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			req := &proto.CreateVolumeRequest{CapacityRange: testCase.CR}
			minSize, maxSize, ok := volumeSizeFromRequest(req)
			if minSize != testCase.MinSize || maxSize != testCase.MaxSize || ok != testCase.OK {
				t.Fatalf("min=%d max=%d ok=%v", minSize, maxSize, ok)
			}
		})
	}
}
