package driver

import (
	"context"
	"fmt"
	"strconv"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hetzner.cloud/csi/volumes"
)

type NodeService struct {
	logger             log.Logger
	server             *hcloud.Server
	volumeService      volumes.Service
	volumeMountService volumes.MountService
}

func NewNodeService(
	logger log.Logger,
	server *hcloud.Server,
	volumeService volumes.Service,
	volumeMountService volumes.MountService,
) *NodeService {
	return &NodeService{
		logger:             logger,
		server:             server,
		volumeService:      volumeService,
		volumeMountService: volumeMountService,
	}
}

func (s *NodeService) NodeStageVolume(ctx context.Context, req *proto.NodeStageVolumeRequest) (*proto.NodeStageVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing staging target path")
	}
	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "missing volume capability")
	}

	mount := req.VolumeCapability.GetMount()
	if mount == nil {
		return nil, status.Error(codes.InvalidArgument, "no mount capability")
	}

	volumeID, err := parseVolumeID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	volume, err := s.volumeService.GetByID(ctx, volumeID)
	if err != nil {
		switch err {
		case volumes.ErrVolumeNotFound:
			return nil, status.Error(codes.NotFound, "volume not found")
		default:
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get volume: %s", err))
		}
	}

	opts := volumes.NewMountOpts()
	if mount.FsType != "" {
		opts.FSType = mount.FsType
	}
	opts.Additional = mount.MountFlags

	if err := s.volumeMountService.Stage(volume, req.StagingTargetPath, opts); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to stage volume: %s", err))
	}

	resp := &proto.NodeStageVolumeResponse{}
	return resp, nil
}

func (s *NodeService) NodeUnstageVolume(ctx context.Context, req *proto.NodeUnstageVolumeRequest) (*proto.NodeUnstageVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing staging target path")
	}

	volumeID, err := parseVolumeID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	volume, err := s.volumeService.GetByID(ctx, volumeID)
	if err != nil {
		switch err {
		case volumes.ErrVolumeNotFound:
			return nil, status.Error(codes.NotFound, "volume not found")
		default:
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get volume: %s", err))
		}
	}

	if err := s.volumeMountService.Unstage(volume, req.StagingTargetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unstage volume: %s", err))
	}

	resp := &proto.NodeUnstageVolumeResponse{}
	return resp, nil
}

func (s *NodeService) NodePublishVolume(ctx context.Context, req *proto.NodePublishVolumeRequest) (*proto.NodePublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing staging target path")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing target path")
	}

	mount := req.VolumeCapability.GetMount()
	if mount == nil {
		return nil, status.Error(codes.InvalidArgument, "no mount capability")
	}

	volumeID, err := parseVolumeID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	volume, err := s.volumeService.GetByID(ctx, volumeID)
	if err != nil {
		switch err {
		case volumes.ErrVolumeNotFound:
			return nil, status.Error(codes.NotFound, "volume not found")
		default:
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get volume: %s", err))
		}
	}

	opts := volumes.NewMountOpts()
	opts.Readonly = req.Readonly
	if mount.FsType != "" {
		opts.FSType = mount.FsType
	}
	opts.Additional = mount.MountFlags

	if err := s.volumeMountService.Publish(volume, req.TargetPath, req.StagingTargetPath, opts); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to publish volume: %s", err))
	}

	resp := &proto.NodePublishVolumeResponse{}
	return resp, nil
}

func (s *NodeService) NodeUnpublishVolume(ctx context.Context, req *proto.NodeUnpublishVolumeRequest) (*proto.NodeUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "missing target path")
	}

	volumeID, err := parseVolumeID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	volume, err := s.volumeService.GetByID(ctx, volumeID)
	if err != nil {
		switch err {
		case volumes.ErrVolumeNotFound:
			return nil, status.Error(codes.NotFound, "volume not found")
		default:
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get volume: %s", err))
		}
	}

	if err := s.volumeMountService.Unpublish(volume, req.TargetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unpublish volume: %s", err))
	}

	resp := &proto.NodeUnpublishVolumeResponse{}
	return resp, nil
}

func (s *NodeService) NodeGetVolumeStats(ctx context.Context, req *proto.NodeGetVolumeStatsRequest) (*proto.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "volume stats are not supported")
}

func (s *NodeService) NodeGetCapabilities(ctx context.Context, req *proto.NodeGetCapabilitiesRequest) (*proto.NodeGetCapabilitiesResponse, error) {
	resp := &proto.NodeGetCapabilitiesResponse{
		Capabilities: []*proto.NodeServiceCapability{
			{
				Type: &proto.NodeServiceCapability_Rpc{
					Rpc: &proto.NodeServiceCapability_RPC{
						Type: proto.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
		},
	}
	return resp, nil
}

func (s *NodeService) NodeGetInfo(context.Context, *proto.NodeGetInfoRequest) (*proto.NodeGetInfoResponse, error) {
	if s.server == nil || s.server.Datacenter == nil || s.server.Datacenter.Location == nil || s.server.Datacenter.Location.Name == "" {
		return nil, status.Error(codes.Internal, "cannot determine node location")
	}
	location := s.server.Datacenter.Location.Name

	resp := &proto.NodeGetInfoResponse{
		NodeId:            strconv.Itoa(s.server.ID),
		MaxVolumesPerNode: MaxVolumesPerNode,
		AccessibleTopology: &proto.Topology{
			Segments: map[string]string{
				TopologySegmentLocation: location,
			},
		},
	}
	return resp, nil
}

func (s *NodeService) NodeExpandVolume(context.Context, *proto.NodeExpandVolumeRequest) (*proto.NodeExpandVolumeResponse, error) {
	panic("implement me")
}
