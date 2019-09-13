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
			Name: "with required bytes less than minimum size",
			CR: &proto.CapacityRange{
				RequiredBytes: (MinVolumeSize - 1) * GB,
			},
			MinSize: MinVolumeSize,
			MaxSize: 0,
			OK:      true,
		},
		{
			Name: "with required bytes exactly minimum size",
			CR: &proto.CapacityRange{
				RequiredBytes: MinVolumeSize * GB,
			},
			MinSize: MinVolumeSize,
			MaxSize: 0,
			OK:      true,
		},
		{
			Name: "with required bytes slightly more than minimum size",
			CR: &proto.CapacityRange{
				RequiredBytes: MinVolumeSize*GB + 100,
			},
			MinSize: MinVolumeSize + 1,
			MaxSize: 0,
			OK:      true,
		},
		{
			Name: "with required and limit bytes",
			CR: &proto.CapacityRange{
				RequiredBytes: MinVolumeSize*GB + 100,
				LimitBytes:    2 * MinVolumeSize * GB,
			},
			MinSize: MinVolumeSize + 1,
			MaxSize: 2 * MinVolumeSize,
			OK:      true,
		},
		{
			Name: "with required and limit bytes (same value)",
			CR: &proto.CapacityRange{
				RequiredBytes: MinVolumeSize * GB,
				LimitBytes:    MinVolumeSize * GB,
			},
			MinSize: MinVolumeSize,
			MaxSize: MinVolumeSize,
			OK:      true,
		},
		{
			Name: "with lower limit than required",
			CR: &proto.CapacityRange{
				RequiredBytes: 2 * MinVolumeSize * GB,
				LimitBytes:    MinVolumeSize * GB,
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
