package driver

import (
	"math"
	"strconv"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
)

func parseVolumeID(id string) (uint64, error) { return strconv.ParseUint(id, 10, 64) }
func parseNodeID(id string) (uint64, error)   { return strconv.ParseUint(id, 10, 64) }

func volumeSizeFromCapacityRange(cr *proto.CapacityRange) (int, int, bool) {
	if cr == nil {
		return DefaultVolumeSize, 0, true
	}

	var minSize int
	switch {
	case cr.RequiredBytes == 0:
		minSize = DefaultVolumeSize
	case cr.RequiredBytes < 0:
		return 0, 0, false
	default:
		minSize = int(math.Ceil(float64(cr.RequiredBytes) / 1024 / 1024 / 1024))
		if minSize < MinVolumeSize {
			minSize = MinVolumeSize
		}
	}

	var maxSize int
	switch {
	case cr.LimitBytes == 0:
		break // ignore
	case cr.LimitBytes < 0:
		return 0, 0, false
	default:
		maxSize = int(math.Floor(float64(cr.LimitBytes) / 1024 / 1024 / 1024))
	}

	if maxSize != 0 && minSize > maxSize {
		return 0, 0, false
	}

	return minSize, maxSize, true
}

func isCapabilitySupported(cap *proto.VolumeCapability) bool {
	if cap.AccessMode == nil {
		return false
	}
	switch cap.AccessMode.Mode {
	case proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER:
		return true
	default:
		return false
	}
}
