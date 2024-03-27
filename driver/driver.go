package driver

const (
	PluginName    = "csi.hetzner.cloud"
	PluginVersion = "2.7.0" // x-release-please-version

	MaxVolumesPerNode = 16
	MinVolumeSize     = 10 // GB
	DefaultVolumeSize = MinVolumeSize

	TopologySegmentLocation = PluginName + "/location"
)
