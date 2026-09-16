package driver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hetznercloud/csi-driver/internal/csi"
	"github.com/hetznercloud/csi-driver/internal/utils"
	"github.com/hetznercloud/csi-driver/internal/volumes"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const (
	parameterKeyPVCName      = "csi.storage.k8s.io/pvc/name"
	parameterKeyPVCNamespace = "csi.storage.k8s.io/pvc/namespace"
	parameterKeyPVName       = "csi.storage.k8s.io/pv/name"
	parameterKeyLabels       = "labels"

	labelKeyPVCName      = "pvc-name"
	labelKeyPVCNamespace = "pvc-namespace"
	labelKeyPVName       = "pv-name"
	labelKeyManagedBy    = "managed-by"

	MaxLabelValueLength = 63
)

type ControllerService struct {
	proto.UnimplementedControllerServer

	logger                   *slog.Logger
	volumeService            volumes.Service
	location                 string
	enableProvidedByTopology bool
	extraVolumeLabels        map[string]string
}

func NewControllerService(
	logger *slog.Logger,
	volumeService volumes.Service,
	location string,
	enableProvidedByTopology bool,
	extraVolumeLabels map[string]string,
) *ControllerService {
	return &ControllerService{
		logger:                   logger,
		volumeService:            volumeService,
		location:                 location,
		enableProvidedByTopology: enableProvidedByTopology,
		extraVolumeLabels:        extraVolumeLabels,
	}
}

func (s *ControllerService) CreateVolume(ctx context.Context, req *proto.CreateVolumeRequest) (*proto.CreateVolumeResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing name")
	}
	if len(req.GetVolumeCapabilities()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "missing volume capabilities")
	}

	minSize, maxSize, ok := volumeSizeFromCapacityRange(req.GetCapacityRange())
	if !ok {
		return nil, status.Error(codes.OutOfRange, "invalid capacity range")
	}

	// Check if ALL volume capabilities are supported.
	for i, capability := range req.GetVolumeCapabilities() {
		if !isCapabilitySupported(capability) {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("capability at index %d is not supported", i))
		}
	}

	// If the container orchestration system did send topology requirements but
	// none of them carry a location segment, we must not silently fall back to
	// the controller's location: that can provision the volume in a location the
	// selected node can not reach, leaving the pod unschedulable (see #1428).
	location := s.location
	if reqs := req.GetAccessibilityRequirements(); len(reqs.GetPreferred()) > 0 || len(reqs.GetRequisite()) > 0 {
		loc := locationFromTopologyRequirement(reqs)
		if loc == nil {
			return nil, status.Errorf(codes.InvalidArgument,
				"accessibility requirements were provided but none contained a %q topology segment; "+
					"can not determine the location to create the volume in",
				TopologySegmentLocation)
		}
		location = *loc
	}

	var sourceSnapshot *csi.Snapshot
	if source := req.GetVolumeContentSource(); source != nil {
		if source.GetVolume() != nil {
			return nil, status.Error(codes.InvalidArgument, "volume cloning is not supported")
		}
		if source.GetSnapshot() == nil || source.GetSnapshot().GetSnapshotId() == "" {
			return nil, status.Error(codes.InvalidArgument, "invalid volume content source")
		}
		snapshotID, err := parseVolumeID(source.GetSnapshot().GetSnapshotId())
		if err != nil {
			return nil, status.Error(codes.NotFound, "snapshot not found")
		}
		sourceSnapshot, err = s.volumeService.GetSnapshotByID(ctx, snapshotID)
		if err != nil {
			if errors.Is(err, volumes.ErrSnapshotNotFound) {
				return nil, status.Error(codes.NotFound, "snapshot not found")
			}
			return nil, status.Errorf(codes.Internal, "failed to get snapshot: %s", err)
		}
		if !sourceSnapshot.Ready {
			return nil, status.Error(codes.Aborted, "snapshot is not ready")
		}
		if reqs := req.GetAccessibilityRequirements(); len(reqs.GetPreferred()) > 0 || len(reqs.GetRequisite()) > 0 {
			if location != sourceSnapshot.Location {
				return nil, status.Errorf(codes.InvalidArgument, "snapshot is in location %q, not requested location %q", sourceSnapshot.Location, location)
			}
		}
		location = sourceSnapshot.Location
		if maxSize > 0 && sourceSnapshot.Size > maxSize {
			return nil, status.Error(codes.OutOfRange, "snapshot is larger than the requested capacity limit")
		}
		if minSize < sourceSnapshot.Size {
			minSize = sourceSnapshot.Size
		}
	}

	volumeLabels := map[string]string{
		labelKeyManagedBy: "csi-driver",
	}

	maps.Copy(volumeLabels, s.extraVolumeLabels)
	if sourceSnapshot != nil {
		volumeLabels[volumes.SnapshotIDLabel] = strconv.FormatInt(sourceSnapshot.ID, 10)
	}

	for key, value := range req.GetParameters() {
		switch strings.ToLower(key) {
		case parameterKeyPVCName:
			volumeLabels[labelKeyPVCName] = value
		case parameterKeyPVCNamespace:
			volumeLabels[labelKeyPVCNamespace] = value
		case parameterKeyPVName:
			volumeLabels[labelKeyPVName] = value
		case parameterKeyLabels:
			customLabels, err := utils.ConvertLabelsToMap(value)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid format of parameter labels: %s", err)
			}
			maps.Copy(volumeLabels, customLabels)
		default:
			s.logger.Warn(fmt.Sprintf("invalid parameter key %s for CreateVolume", key))
		}
	}

	// Validate all labels before sending to the API.
	for k, v := range volumeLabels {
		// Truncate label values to fit API requirements
		if len(v) > MaxLabelValueLength {
			truncated := v[len(v)-MaxLabelValueLength:]
			// After truncation the first character might not be alphanumeric
			// (e.g. a dash), which violates the label spec. Strip any
			// leading non-alphanumeric characters.
			truncated = strings.TrimLeftFunc(truncated, func(r rune) bool {
				return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
			})
			s.logger.Warn(
				"volume label value truncated",
				"volume", req.GetName(),
				"key", k,
				"original", v,
				"truncated", truncated,
			)
			volumeLabels[k] = truncated
		}
	}

	labelsIface := make(map[string]any, len(volumeLabels))
	for k, v := range volumeLabels {
		labelsIface[k] = v
	}
	if _, err := hcloud.ValidateResourceLabels(labelsIface); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid volume labels: %s", err)
	}

	// Create the volume. The service handles idempotency as required by the CSI spec.
	volume, err := s.volumeService.Create(ctx, volumes.CreateOpts{
		Name:     req.GetName(),
		MinSize:  minSize,
		MaxSize:  maxSize,
		Location: location,
		Labels:   volumeLabels,
		Snapshot: sourceSnapshot,
	})
	if err != nil {
		s.logger.Error(
			"failed to create volume",
			"err", err,
		)
		code := codes.Internal
		switch { //nolint:gocritic
		case errors.Is(err, volumes.ErrVolumeAlreadyExists):
			code = codes.AlreadyExists
		}
		return nil, status.Error(code, fmt.Sprintf("failed to create volume: %s", err))
	}
	s.logger.Info(
		"created volume",
		"volume-id", volume.ID,
		"volume-name", volume.Name,
	)

	topology := &proto.Topology{
		Segments: map[string]string{
			TopologySegmentLocation: volume.Location,
		},
	}

	if s.enableProvidedByTopology {
		topology.Segments[ProvidedByLabel] = "cloud"
	}

	resp := &proto.CreateVolumeResponse{
		Volume: &proto.Volume{
			VolumeId:      strconv.FormatInt(volume.ID, 10),
			CapacityBytes: volume.SizeBytes(),
			AccessibleTopology: []*proto.Topology{
				topology,
			},
			VolumeContext: map[string]string{
				"fsFormatOptions": req.GetParameters()["fsFormatOptions"],
			},
			ContentSource: req.GetVolumeContentSource(),
		},
	}
	return resp, nil
}

func (s *ControllerService) DeleteVolume(ctx context.Context, req *proto.DeleteVolumeRequest) (*proto.DeleteVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid volume id")
	}

	if volumeID, err := parseVolumeID(req.GetVolumeId()); err == nil {
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
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing volume id")
	}
	if req.GetNodeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing node id")
	}
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "missing volume capabilities")
	}

	volumeID, err := parseVolumeID(req.GetVolumeId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	serverID, err := parseNodeID(req.GetNodeId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "node not found")
	}

	if !isCapabilitySupported(req.GetVolumeCapability()) {
		return nil, status.Error(codes.InvalidArgument, "capability is not supported")
	}
	if req.GetReadonly() {
		return nil, status.Error(codes.InvalidArgument, "readonly volumes are not supported")
	}

	volume := &csi.Volume{ID: volumeID}
	server := &csi.Server{ID: serverID}

	if err := s.volumeService.Attach(ctx, volume, server); err != nil {
		code := codes.Internal
		switch {
		case errors.Is(err, volumes.ErrVolumeNotFound):
			code = codes.NotFound
		case errors.Is(err, volumes.ErrServerNotFound):
			code = codes.NotFound
		case errors.Is(err, volumes.ErrAttached):
			code = codes.FailedPrecondition
		case errors.Is(err, volumes.ErrAttachLimitReached):
			code = codes.ResourceExhausted
		case errors.Is(err, volumes.ErrLockedServer):
			code = codes.Unavailable
		}
		return nil, status.Error(code, fmt.Sprintf("failed to publish volume: %s", err))
	}

	volume, err = s.volumeService.GetByID(ctx, volumeID)
	if err != nil {
		switch {
		case errors.Is(err, volumes.ErrVolumeNotFound):
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
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid volume id")
	}

	volumeID, err := parseVolumeID(req.GetVolumeId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}
	volume := &csi.Volume{ID: volumeID}

	var server *csi.Server
	if req.GetNodeId() != "" {
		serverID, err := parseNodeID(req.GetNodeId())
		if err != nil {
			return nil, status.Error(codes.NotFound, "node not found")
		}
		server = &csi.Server{ID: serverID}
	}

	if err := s.volumeService.Detach(ctx, volume, server); err != nil {
		code := codes.Internal
		switch {
		case errors.Is(err, volumes.ErrVolumeNotFound): // Based on the spec it is save to assume that the call was successful if the volume is not found
			resp := &proto.ControllerUnpublishVolumeResponse{}
			return resp, nil
		case errors.Is(err, volumes.ErrServerNotFound):
			code = codes.NotFound
		case errors.Is(err, volumes.ErrLockedServer):
			code = codes.Unavailable
		}
		return nil, status.Error(code, fmt.Sprintf("failed to unpublish volume: %s", err))
	}

	resp := &proto.ControllerUnpublishVolumeResponse{}
	return resp, nil
}

func (s *ControllerService) ValidateVolumeCapabilities(ctx context.Context, req *proto.ValidateVolumeCapabilitiesRequest) (*proto.ValidateVolumeCapabilitiesResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid volume id")
	}
	if len(req.GetVolumeCapabilities()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "missing volume capabilities")
	}

	volumeID, err := parseVolumeID(req.GetVolumeId())
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
	for _, capability := range req.GetVolumeCapabilities() {
		if !isCapabilitySupported(capability) {
			confirmed = false
			break
		}
	}

	resp := &proto.ValidateVolumeCapabilitiesResponse{}
	if confirmed {
		resp.Confirmed = &proto.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.GetVolumeCapabilities(),
		}
	}
	return resp, nil
}

func (s *ControllerService) ListVolumes(ctx context.Context, req *proto.ListVolumesRequest) (*proto.ListVolumesResponse, error) {
	if req.GetStartingToken() != "" {
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
			{
				Type: &proto.ControllerServiceCapability_Rpc{
					Rpc: &proto.ControllerServiceCapability_RPC{
						Type: proto.ControllerServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
					},
				},
			},
			{
				Type: &proto.ControllerServiceCapability_Rpc{
					Rpc: &proto.ControllerServiceCapability_RPC{
						Type: proto.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
					},
				},
			},
			{
				Type: &proto.ControllerServiceCapability_Rpc{
					Rpc: &proto.ControllerServiceCapability_RPC{
						Type: proto.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
					},
				},
			},
		},
	}
	return resp, nil
}

func (s *ControllerService) CreateSnapshot(ctx context.Context, req *proto.CreateSnapshotRequest) (*proto.CreateSnapshotResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing name")
	}
	if req.GetSourceVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing source volume id")
	}
	volumeID, err := parseVolumeID(req.GetSourceVolumeId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "source volume not found")
	}

	labels := map[string]string{labelKeyManagedBy: "csi-driver"}
	maps.Copy(labels, s.extraVolumeLabels)
	snapshot, err := s.volumeService.CreateSnapshot(ctx, volumes.CreateSnapshotOpts{
		Name:     req.GetName(),
		VolumeID: volumeID,
		Labels:   labels,
	})
	if err != nil {
		switch {
		case errors.Is(err, volumes.ErrVolumeNotFound):
			return nil, status.Error(codes.NotFound, "source volume not found")
		case errors.Is(err, volumes.ErrSnapshotAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, "snapshot name already exists for another source volume")
		default:
			return nil, status.Errorf(codes.Internal, "failed to create snapshot: %s", err)
		}
	}
	return &proto.CreateSnapshotResponse{Snapshot: snapshotToProto(snapshot)}, nil
}

func (s *ControllerService) DeleteSnapshot(ctx context.Context, req *proto.DeleteSnapshotRequest) (*proto.DeleteSnapshotResponse, error) {
	if req.GetSnapshotId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing snapshot id")
	}
	snapshotID := parseSnapshotIDForDelete(req.GetSnapshotId())
	if snapshotID == 0 {
		return &proto.DeleteSnapshotResponse{}, nil
	}
	if err := s.volumeService.DeleteSnapshot(ctx, &csi.Snapshot{ID: snapshotID}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete snapshot: %s", err)
	}
	return &proto.DeleteSnapshotResponse{}, nil
}

func parseSnapshotIDForDelete(id string) int64 {
	snapshotID, err := parseVolumeID(id)
	if err != nil {
		return 0
	}
	return snapshotID
}

func (s *ControllerService) ListSnapshots(ctx context.Context, req *proto.ListSnapshotsRequest) (*proto.ListSnapshotsResponse, error) {
	if req.GetMaxEntries() < 0 {
		return nil, status.Error(codes.InvalidArgument, "max entries must not be negative")
	}
	snapshots, err := s.volumeService.AllSnapshots(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list snapshots: %s", err)
	}

	if req.GetSnapshotId() != "" {
		id, parseErr := parseVolumeID(req.GetSnapshotId())
		if parseErr != nil {
			snapshots = nil
		} else {
			snapshots = slices.DeleteFunc(snapshots, func(snapshot *csi.Snapshot) bool { return snapshot.ID != id })
		}
	}
	if req.GetSourceVolumeId() != "" {
		id, parseErr := parseVolumeID(req.GetSourceVolumeId())
		if parseErr != nil {
			snapshots = nil
		} else {
			snapshots = slices.DeleteFunc(snapshots, func(snapshot *csi.Snapshot) bool { return snapshot.SourceVolumeID != id })
		}
	}

	start := 0
	if req.GetStartingToken() != "" {
		start, err = strconv.Atoi(req.GetStartingToken())
		if err != nil || start < 0 || start > len(snapshots) {
			return nil, status.Error(codes.Aborted, "invalid starting token")
		}
	}
	end := len(snapshots)
	if req.GetMaxEntries() > 0 && start+int(req.GetMaxEntries()) < end {
		end = start + int(req.GetMaxEntries())
	}
	resp := &proto.ListSnapshotsResponse{Entries: make([]*proto.ListSnapshotsResponse_Entry, 0, end-start)}
	for _, snapshot := range snapshots[start:end] {
		resp.Entries = append(resp.Entries, &proto.ListSnapshotsResponse_Entry{Snapshot: snapshotToProto(snapshot)})
	}
	if end < len(snapshots) {
		resp.NextToken = strconv.Itoa(end)
	}
	return resp, nil
}

func snapshotToProto(snapshot *csi.Snapshot) *proto.Snapshot {
	return &proto.Snapshot{
		SnapshotId:     strconv.FormatInt(snapshot.ID, 10),
		SourceVolumeId: strconv.FormatInt(snapshot.SourceVolumeID, 10),
		SizeBytes:      snapshot.SizeBytes(),
		CreationTime:   timestamppb.New(snapshot.Created),
		ReadyToUse:     snapshot.Ready,
	}
}

func (s *ControllerService) ControllerExpandVolume(ctx context.Context, req *proto.ControllerExpandVolumeRequest) (*proto.ControllerExpandVolumeResponse, error) {
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid volume id")
	}

	volumeID, err := parseVolumeID(req.GetVolumeId())
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
		switch { //nolint:gocritic
		case errors.Is(err, volumes.ErrVolumeNotFound):
			code = codes.NotFound
		}
		return nil, status.Error(code, fmt.Sprintf("failed to expand volume: %s", err))
	}

	if volume, err = s.volumeService.GetByID(ctx, volumeID); err != nil {
		code := codes.Internal
		switch { //nolint:gocritic
		case errors.Is(err, volumes.ErrVolumeNotFound):
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
