package driver

const (
	PluginName    = "csi.hetzner.cloud"
	PluginVersion = "1.1.2"

	MaxVolumesPerNode = 16
	DefaultVolumeSize = 10 // GB

	TopologySegmentLocation = PluginName + "/location"
)
