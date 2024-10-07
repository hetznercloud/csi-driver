package driver

import (
	"math"
	"strconv"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
)

func parseVolumeID(id string) (int64, error) { return strconv.ParseInt(id, 10, 64) }
func parseNodeID(id string) (int64, error)   { return strconv.ParseInt(id, 10, 64) }

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
	case proto.VolumeCapability_AccessMode_SINGLE_NODE_MULTI_WRITER:
		return true
	default:
		return false
	}
}

func locationFromTopologyRequirement(tr *proto.TopologyRequirement) *string {
	if tr == nil {
		return nil
	}
	for _, top := range tr.Preferred {
		if location, ok := top.Segments[TopologySegmentLocation]; ok {
			return &location
		}
	}
	for _, top := range tr.Requisite {
		if location, ok := top.Segments[TopologySegmentLocation]; ok {
			return &location
		}
	}
	return nil
}
