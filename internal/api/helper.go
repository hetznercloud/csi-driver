package api

import (
	"github.com/hetznercloud/csi-driver/internal/csi"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func toDomainVolume(hcloudVolume *hcloud.Volume) *csi.Volume {
	return &csi.Volume{
		ID:          hcloudVolume.ID,
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
		ID: hcloudServer.ID,
	}
}
