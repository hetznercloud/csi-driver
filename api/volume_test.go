package api

import (
	"hetzner.cloud/csi/volumes"
)

var _ volumes.Service = (*VolumeService)(nil)
