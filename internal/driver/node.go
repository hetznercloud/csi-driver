package driver

import (
	"context"
	"fmt"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hetznercloud/csi-driver/internal/volumes"
)

type NodeService struct {
	proto.UnimplementedNodeServer

	logger              log.Logger
	serverID            string
	serverLocation      string
	volumeMountService  volumes.MountService
	volumeResizeService volumes.ResizeService
	volumeStatsService  volumes.StatsService
}

func NewNodeService(
	logger log.Logger,
	serverID string,
	serverLocation string,
	volumeMountService volumes.MountService,
	volumeResizeService volumes.ResizeService,
	volumeStatsService volumes.StatsService,
) *NodeService {
	return &NodeService{
		logger:              logger,
		serverID:            serverID,
		serverLocation:      serverLocation,
		volumeMountService:  volumeMountService,
		volumeResizeService: volumeResizeService,
		volumeStatsService:  volumeStatsService,
	}
}

const encryptionPassphraseKey = "encryption-passphrase"

func (s *NodeService) NodeStageVolume(_ context.Context, _ *proto.NodeStageVolumeRequest) (*proto.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not supported")
}

func (s *NodeService) NodeUnstageVolume(_ context.Context, _ *proto.NodeUnstageVolumeRequest) (*proto.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not supported")
}

func (s *NodeService) NodePublishVolume(_ context.Context, req *proto.NodePublishVolumeRequest) (*proto.NodePublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing target path")
	}

	devicePath := req.GetPublishContext()["devicePath"]
	if devicePath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing device path")
	}

	var opts volumes.MountOpts
	switch {
	case req.VolumeCapability.GetBlock() != nil:
		opts = volumes.MountOpts{
			BlockVolume:          true,
			EncryptionPassphrase: req.Secrets[encryptionPassphraseKey],
		}
	case req.VolumeCapability.GetMount() != nil:
		mount := req.VolumeCapability.GetMount()
		opts = volumes.MountOpts{
			FSType:               mount.FsType,
			Readonly:             req.Readonly,
			Additional:           mount.MountFlags,
			EncryptionPassphrase: req.Secrets[encryptionPassphraseKey],
			FsFormatOptions:      req.VolumeContext["fsFormatOptions"],
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "publish volume: unsupported volume capability")
	}

	if err := s.volumeMountService.Publish(req.TargetPath, devicePath, opts); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to publish volume: %s", err))
	}
	return &proto.NodePublishVolumeResponse{}, nil
}

func (s *NodeService) NodeUnpublishVolume(_ context.Context, req *proto.NodeUnpublishVolumeRequest) (*proto.NodeUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing target path")
	}

	if err := s.volumeMountService.Unpublish(req.TargetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unpublish volume: %s", err))
	}

	resp := &proto.NodeUnpublishVolumeResponse{}
	return resp, nil
}

func (s *NodeService) NodeGetVolumeStats(_ context.Context, req *proto.NodeGetVolumeStatsRequest) (*proto.NodeGetVolumeStatsResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.VolumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume path")
	}

	volumeExists, err := s.volumeMountService.PathExists(req.VolumePath)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to check for volume existence: %s", err))
	}
	if !volumeExists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("volume %s is not available on this node", req.VolumePath))
	}

	totalBytes, availableBytes, usedBytes, err := s.volumeStatsService.ByteFilesystemStats(req.VolumePath)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get volume byte stats: %s", err))
	}

	totalINodes, usedINodes, freeINodes, err := s.volumeStatsService.INodeFilesystemStats(req.VolumePath)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get volume inode stats: %s", err))
	}

	return &proto.NodeGetVolumeStatsResponse{
		Usage: []*proto.VolumeUsage{
			{
				Unit:      proto.VolumeUsage_BYTES,
				Available: availableBytes,
				Total:     totalBytes,
				Used:      usedBytes,
			},
			{
				Unit:      proto.VolumeUsage_INODES,
				Available: freeINodes,
				Total:     totalINodes,
				Used:      usedINodes,
			},
		},
	}, nil
}

func (s *NodeService) NodeGetCapabilities(_ context.Context, _ *proto.NodeGetCapabilitiesRequest) (*proto.NodeGetCapabilitiesResponse, error) {
	return &proto.NodeGetCapabilitiesResponse{
		Capabilities: []*proto.NodeServiceCapability{
			{
				Type: &proto.NodeServiceCapability_Rpc{
					Rpc: &proto.NodeServiceCapability_RPC{
						Type: proto.NodeServiceCapability_RPC_EXPAND_VOLUME,
					},
				},
			},
			{
				Type: &proto.NodeServiceCapability_Rpc{
					Rpc: &proto.NodeServiceCapability_RPC{
						Type: proto.NodeServiceCapability_RPC_GET_VOLUME_STATS,
					},
				},
			},
		},
	}, nil
}

func (s *NodeService) NodeGetInfo(_ context.Context, _ *proto.NodeGetInfoRequest) (*proto.NodeGetInfoResponse, error) {
	resp := &proto.NodeGetInfoResponse{
		NodeId:            s.serverID,
		MaxVolumesPerNode: MaxVolumesPerNode,
		AccessibleTopology: &proto.Topology{
			Segments: map[string]string{
				TopologySegmentLocation: s.serverLocation,
			},
		},
	}
	return resp, nil
}

func (s *NodeService) NodeExpandVolume(_ context.Context, req *proto.NodeExpandVolumeRequest) (*proto.NodeExpandVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.VolumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume path")
	}
	volumeExists, err := s.volumeMountService.PathExists(req.VolumePath)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to check for volume existence: %s", err))
	}

	if !volumeExists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("volume %s is not available on this node", req.VolumePath))
	}

	if req.VolumeCapability.GetBlock() == nil {
		if err := s.volumeResizeService.Resize(req.VolumePath); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to resize volume: %s", err))
		}
	}

	return &proto.NodeExpandVolumeResponse{}, nil
}
