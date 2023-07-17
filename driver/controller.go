package driver

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hetznercloud/csi-driver/csi"
	"github.com/hetznercloud/csi-driver/volumes"
)

type ControllerService struct {
	logger        log.Logger
	volumeService volumes.Service
	location      string
}

func NewControllerService(
	logger log.Logger,
	volumeService volumes.Service,
	location string,
) *ControllerService {
	return &ControllerService{
		logger:        logger,
		volumeService: volumeService,
		location:      location,
	}
}

func (s *ControllerService) CreateVolume(ctx context.Context, req *proto.CreateVolumeRequest) (*proto.CreateVolumeResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "missing name")
	}
	if len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "missing volume capabilities")
	}

	minSize, maxSize, ok := volumeSizeFromCapacityRange(req.GetCapacityRange())
	if !ok {
		return nil, status.Error(codes.OutOfRange, "invalid capacity range")
	}

	// Check if ALL volume capabilities are supported.
	for i, cap := range req.VolumeCapabilities {
		if !isCapabilitySupported(cap) {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("capability at index %d is not supported", i))
		}
	}

	// Take the location where to create the volume from the request's
	// accessibility requirements, falling back to the location where the
	// controller pod has been scheduled if no requirements have been provided.
	var location = s.location
	if loc := locationFromTopologyRequirement(req.AccessibilityRequirements); loc != nil {
		location = *loc
	}

	// Create the volume. The service handles idempotency as required by the CSI spec.
	volume, err := s.volumeService.Create(ctx, volumes.CreateOpts{
		Name:     req.Name,
		MinSize:  minSize,
		MaxSize:  maxSize,
		Location: location,
	})
	if err != nil {
		level.Error(s.logger).Log(
			"msg", "failed to create volume",
			"err", err,
		)
		code := codes.Internal
		switch err {
		case volumes.ErrVolumeAlreadyExists:
			code = codes.AlreadyExists
		}
		return nil, status.Error(code, fmt.Sprintf("failed to create volume: %s", err))
	}
	level.Info(s.logger).Log(
		"msg", "created volume",
		"volume-id", volume.ID,
		"volume-name", volume.Name,
	)

	resp := &proto.CreateVolumeResponse{
		Volume: &proto.Volume{
			VolumeId:      strconv.FormatInt(volume.ID, 10),
			CapacityBytes: volume.SizeBytes(),
			AccessibleTopology: []*proto.Topology{
				{
					Segments: map[string]string{
						TopologySegmentLocation: volume.Location,
					},
				},
			},
		},
	}
	return resp, nil
}

func (s *ControllerService) DeleteVolume(ctx context.Context, req *proto.DeleteVolumeRequest) (*proto.DeleteVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid volume id")
	}

	if volumeID, err := parseVolumeID(req.VolumeId); err == nil {
		volume := &csi.Volume{ID: volumeID}
		if err := s.volumeService.Delete(ctx, volume); err != nil {
			if errors.Is(err, volumes.ErrVolumeNotFound) {
				return &proto.DeleteVolumeResponse{}, nil
			}
			if errors.Is(err, volumes.ErrAttached) {
				return nil, status.Error(codes.FailedPrecondition, err.Error())
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	resp := &proto.DeleteVolumeResponse{}
	return resp, nil
}

func (s *ControllerService) ControllerPublishVolume(ctx context.Context, req *proto.ControllerPublishVolumeRequest) (*proto.ControllerPublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing node id")
	}
	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "missing volume capabilities")
	}

	volumeID, err := parseVolumeID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	serverID, err := parseNodeID(req.NodeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "node not found")
	}

	if !isCapabilitySupported(req.VolumeCapability) {
		return nil, status.Error(codes.InvalidArgument, "capability is not supported")
	}
	if req.Readonly {
		return nil, status.Error(codes.InvalidArgument, "readonly volumes are not supported")
	}

	volume := &csi.Volume{ID: volumeID}
	server := &csi.Server{ID: serverID}

	if err := s.volumeService.Attach(ctx, volume, server); err != nil {
		code := codes.Internal
		switch err {
		case volumes.ErrVolumeNotFound:
			code = codes.NotFound
		case volumes.ErrServerNotFound:
			code = codes.NotFound
		case volumes.ErrAttached:
			code = codes.FailedPrecondition
		case volumes.ErrAttachLimitReached:
			code = codes.ResourceExhausted
		case volumes.ErrLockedServer:
			code = codes.Unavailable
		}
		return nil, status.Error(code, fmt.Sprintf("failed to publish volume: %s", err))
	}

	volume, err = s.volumeService.GetByID(ctx, volumeID)
	if err != nil {
		switch err {
		case volumes.ErrVolumeNotFound:
			return nil, status.Error(codes.NotFound, "volume not found")
		default:
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get volume: %s", err))
		}
	}

	resp := &proto.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{
			"devicePath": volume.LinuxDevice,
		},
	}
	return resp, nil
}

func (s *ControllerService) ControllerUnpublishVolume(ctx context.Context, req *proto.ControllerUnpublishVolumeRequest) (*proto.ControllerUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid volume id")
	}

	volumeID, err := parseVolumeID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}
	volume := &csi.Volume{ID: volumeID}

	var server *csi.Server
	if req.NodeId != "" {
		serverID, err := parseNodeID(req.NodeId)
		if err != nil {
			return nil, status.Error(codes.NotFound, "node not found")
		}
		server = &csi.Server{ID: serverID}
	}

	if err := s.volumeService.Detach(ctx, volume, server); err != nil {
		code := codes.Internal
		switch err {
		case volumes.ErrVolumeNotFound: // Based on the spec it is save to assume that the call was successful if the volume is not found
			resp := &proto.ControllerUnpublishVolumeResponse{}
			return resp, nil
		case volumes.ErrServerNotFound:
			code = codes.NotFound
		case volumes.ErrLockedServer:
			code = codes.Unavailable
		}
		return nil, status.Error(code, fmt.Sprintf("failed to unpublish volume: %s", err))
	}

	resp := &proto.ControllerUnpublishVolumeResponse{}
	return resp, nil
}

func (s *ControllerService) ValidateVolumeCapabilities(ctx context.Context, req *proto.ValidateVolumeCapabilitiesRequest) (*proto.ValidateVolumeCapabilitiesResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid volume id")
	}
	if len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "missing volume capabilities")
	}

	volumeID, err := parseVolumeID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	volume, err := s.volumeService.GetByID(ctx, volumeID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if volume == nil {
		return nil, status.Error(codes.NotFound, "volume does not exist")
	}

	confirmed := true
	for _, cap := range req.VolumeCapabilities {
		if !isCapabilitySupported(cap) {
			confirmed = false
			break
		}
	}

	resp := &proto.ValidateVolumeCapabilitiesResponse{}
	if confirmed {
		resp.Confirmed = &proto.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.VolumeCapabilities,
		}
	}
	return resp, nil
}

func (s *ControllerService) ListVolumes(ctx context.Context, req *proto.ListVolumesRequest) (*proto.ListVolumesResponse, error) {
	if req.StartingToken != "" {
		return nil, status.Error(codes.Aborted, "Starting token is not implemented")
	}

	vols, err := s.volumeService.All(ctx)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &proto.ListVolumesResponse{Entries: make([]*proto.ListVolumesResponse_Entry, len(vols))}
	for i, volume := range vols {
		resp.Entries[i] = &proto.ListVolumesResponse_Entry{
			Volume: &proto.Volume{
				VolumeId:      strconv.FormatInt(volume.ID, 10),
				CapacityBytes: volume.SizeBytes(),
				AccessibleTopology: []*proto.Topology{
					{
						Segments: map[string]string{
							TopologySegmentLocation: volume.Location,
						},
					},
				},
			},
		}
	}

	return resp, nil
}

func (s *ControllerService) GetCapacity(context.Context, *proto.GetCapacityRequest) (*proto.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "getting capacity is not supported")
}

func (s *ControllerService) ControllerGetCapabilities(context.Context, *proto.ControllerGetCapabilitiesRequest) (*proto.ControllerGetCapabilitiesResponse, error) {
	resp := &proto.ControllerGetCapabilitiesResponse{
		Capabilities: []*proto.ControllerServiceCapability{
			{
				Type: &proto.ControllerServiceCapability_Rpc{
					Rpc: &proto.ControllerServiceCapability_RPC{
						Type: proto.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
			{
				Type: &proto.ControllerServiceCapability_Rpc{
					Rpc: &proto.ControllerServiceCapability_RPC{
						Type: proto.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
					},
				},
			},
			{
				Type: &proto.ControllerServiceCapability_Rpc{
					Rpc: &proto.ControllerServiceCapability_RPC{
						Type: proto.ControllerServiceCapability_RPC_EXPAND_VOLUME,
					},
				},
			},
			{
				Type: &proto.ControllerServiceCapability_Rpc{
					Rpc: &proto.ControllerServiceCapability_RPC{
						Type: proto.ControllerServiceCapability_RPC_LIST_VOLUMES,
					},
				},
			},
		},
	}
	return resp, nil
}

func (s *ControllerService) CreateSnapshot(context.Context, *proto.CreateSnapshotRequest) (*proto.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "creating snapshots is not supported")
}

func (s *ControllerService) DeleteSnapshot(context.Context, *proto.DeleteSnapshotRequest) (*proto.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "deleting snapshots is not supported")
}

func (s *ControllerService) ListSnapshots(context.Context, *proto.ListSnapshotsRequest) (*proto.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "listing snapshots is not supported")
}

func (s *ControllerService) ControllerExpandVolume(ctx context.Context, req *proto.ControllerExpandVolumeRequest) (*proto.ControllerExpandVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid volume id")
	}

	volumeID, err := parseVolumeID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}
	volume := &csi.Volume{ID: volumeID}

	minSize, _, ok := volumeSizeFromCapacityRange(req.GetCapacityRange())
	if !ok {
		return nil, status.Error(codes.OutOfRange, "invalid capacity range")
	}

	if err := s.volumeService.Resize(ctx, volume, minSize); err != nil {
		code := codes.Internal
		switch err {
		case volumes.ErrVolumeNotFound:
			code = codes.NotFound
		}
		return nil, status.Error(code, fmt.Sprintf("failed to expand volume: %s", err))
	}

	if volume, err = s.volumeService.GetByID(ctx, volumeID); err != nil {
		code := codes.Internal
		switch err {
		case volumes.ErrVolumeNotFound:
			code = codes.NotFound
		}
		return nil, status.Error(code, fmt.Sprintf("failed to expand volume: %s", err))
	}

	resp := &proto.ControllerExpandVolumeResponse{
		CapacityBytes:         volume.SizeBytes(),
		NodeExpansionRequired: true,
	}
	return resp, nil
}

func (s *ControllerService) ControllerGetVolume(ctx context.Context, req *proto.ControllerGetVolumeRequest) (*proto.ControllerGetVolumeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ControllerGetVolume not implemented")
}
