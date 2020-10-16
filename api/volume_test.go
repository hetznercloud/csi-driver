package api

import (
	"github.com/hetznercloud/csi-driver/volumes"
)

var _ volumes.Service = (*VolumeService)(nil)
