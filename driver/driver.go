package driver

const (
	PluginName    = "csi.hetzner.cloud"
	PluginVersion = "2.2.0"

	MaxVolumesPerNode = 16
	MinVolumeSize     = 10 // GB
	DefaultVolumeSize = MinVolumeSize

	TopologySegmentLocation = PluginName + "/location"
)
