package driver

const (
	PluginName    = "csi.hetzner.cloud"
	PluginVersion = "2.16.0" // x-releaser-pleaser-version

	MaxVolumesPerNode = 16
	MinVolumeSize     = 10 // GB
	DefaultVolumeSize = MinVolumeSize

	TopologySegmentLocation = PluginName + "/location"
	ProvidedByLabel         = "instance.hetzner.cloud/provided-by"
)
