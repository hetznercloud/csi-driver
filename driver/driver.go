package driver

import (
	apiv1 "k8s.io/api/core/v1"
)

const (
	PluginName    = "csi.hetzner.cloud"
	PluginVersion = "1.6.0"

	MaxVolumesPerNode = 16
	MinVolumeSize     = 10 // GB
	DefaultVolumeSize = MinVolumeSize

	TopologySegmentLocation = apiv1.LabelZoneRegionStable

	// this label will be deprecated in future releases
	TopologySegmentLocationLegacy = PluginName + "/location"
)
