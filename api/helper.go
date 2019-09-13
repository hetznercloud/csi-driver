package api

import (
	"github.com/hetznercloud/hcloud-go/hcloud"
	"hetzner.cloud/csi/csi"
)

func toDomainVolume(hcloudVolume *hcloud.Volume) *csi.Volume {
	return &csi.Volume{
		ID:          uint64(hcloudVolume.ID),
		Name:        hcloudVolume.Name,
		Size:        hcloudVolume.Size,
		Location:    hcloudVolume.Location.Name,
		LinuxDevice: hcloudVolume.LinuxDevice,
		Server:      toDomainServer(hcloudVolume.Server),
	}
}

func toDomainServer(hcloudServer *hcloud.Server) *csi.Server {
	if hcloudServer == nil {
		return nil
	}
	return &csi.Server{
		ID: uint64(hcloudServer.ID),
	}
}
