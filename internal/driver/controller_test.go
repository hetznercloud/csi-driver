package driver

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hetznercloud/csi-driver/internal/csi"
	"github.com/hetznercloud/csi-driver/internal/mock"
	"github.com/hetznercloud/csi-driver/internal/volumes"
)

var _ proto.ControllerServer = (*ControllerService)(nil)

type controllerServiceTestEnv struct {
	ctx           context.Context
	service       *ControllerService
	volumeService *mock.VolumeService
}

func newControllerServiceTestEnv() *controllerServiceTestEnv {
	logger := slog.New(slog.DiscardHandler)
	volumeService := &mock.VolumeService{}

	return &controllerServiceTestEnv{
		ctx: context.Background(),
		service: NewControllerService(
			logger,
			volumeService,
			"testloc",
			false,
			map[string]string{"clusterName": "myCluster"},
		),
		volumeService: volumeService,
	}
}

func TestControllerServiceCreateVolume(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.service.enableProvidedByTopology = true

	env.volumeService.CreateFunc = func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
		if opts.Name != "testvol" {
			t.Errorf("unexpected name passed to volume service: %s", opts.Name)
		}
		if opts.MinSize != MinVolumeSize+1 {
			t.Errorf("unexpected min size passed to volume service: %d", opts.MinSize)
		}
		if opts.MaxSize != 2*MinVolumeSize {
			t.Errorf("unexpected max size passed to volume service: %d", opts.MaxSize)
		}
		if opts.Location != "testloc" {
			t.Errorf("unexpected location passed to volume service: %s", opts.Location)
		}
		if v, ok := opts.Labels["clusterName"]; !ok || v != "myCluster" {
			t.Errorf("unexpected labels passed to volume service: %s", opts.Labels)
		}
		if v, ok := opts.Labels[labelKeyPVCName]; !ok || v != "pvc-name" {
			t.Errorf("unexpected labels passed to volume service: %s", opts.Labels)
		}
		if v, ok := opts.Labels[labelKeyPVCNamespace]; !ok || v != "default" {
			t.Errorf("unexpected labels passed to volume service: %s", opts.Labels)
		}
		if v, ok := opts.Labels[labelKeyPVName]; !ok || v != "pv-name" {
			t.Errorf("unexpected labels passed to volume service: %s", opts.Labels)
		}
		return &csi.Volume{
			ID:       1,
			Name:     opts.Name,
			Size:     opts.MinSize,
			Location: opts.Location,
		}, nil
	}

	req := &proto.CreateVolumeRequest{
		Name: "testvol",
		CapacityRange: &proto.CapacityRange{
			RequiredBytes: MinVolumeSize*GB + 100,
			LimitBytes:    2 * MinVolumeSize * GB,
		},
		Parameters: map[string]string{
			parameterKeyPVCName:      "pvc-name",
			parameterKeyPVCNamespace: "default",
			parameterKeyPVName:       "pv-name",
		},
		VolumeCapabilities: []*proto.VolumeCapability{
			{
				AccessType: &proto.VolumeCapability_Mount{
					Mount: &proto.VolumeCapability_MountVolume{},
				},
				AccessMode: &proto.VolumeCapability_AccessMode{
					Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}
	resp, err := env.service.CreateVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetVolume().GetVolumeId() != "1" {
		t.Errorf("unexpected value for VolumeId: %s", resp.GetVolume().GetVolumeId())
	}
	if resp.GetVolume().GetCapacityBytes() != (MinVolumeSize+1)*1024*1024*1024 {
		t.Errorf("unexpected value for CapacityBytes: %d", resp.GetVolume().GetCapacityBytes())
	}
	if len(resp.GetVolume().GetAccessibleTopology()) == 1 {
		top := resp.GetVolume().GetAccessibleTopology()[0]
		if loc := top.GetSegments()[TopologySegmentLocation]; loc != "testloc" {
			t.Errorf("unexpected location segment in topology: %s", loc)
		}
		if env.service.enableProvidedByTopology {
			if provider := top.GetSegments()[ProvidedByLabel]; provider != "cloud" {
				t.Errorf("unexpected provider segment in topology: %s", provider)
			}
		}
	} else {
		t.Errorf("unexpected number of topologies: %d", len(resp.GetVolume().GetAccessibleTopology()))
	}
}

func TestControllerServiceCreateVolumeWithParameterLabels(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.service.enableProvidedByTopology = true

	env.volumeService.CreateFunc = func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
		if opts.Name != "testvol" {
			t.Errorf("unexpected name passed to volume service: %s", opts.Name)
		}
		if opts.MinSize != MinVolumeSize+1 {
			t.Errorf("unexpected min size passed to volume service: %d", opts.MinSize)
		}
		if opts.MaxSize != 2*MinVolumeSize {
			t.Errorf("unexpected max size passed to volume service: %d", opts.MaxSize)
		}
		if opts.Location != "testloc" {
			t.Errorf("unexpected location passed to volume service: %s", opts.Location)
		}
		if v, ok := opts.Labels["test"]; !ok || v != "bel-which-needs-to-be-truncated-with-important-infos-at-the-end" {
			t.Errorf("unexpected labels passed to volume service: %s", opts.Labels)
		}
		if v, ok := opts.Labels["clusterName"]; !ok || v != "myCluster" {
			t.Errorf("unexpected labels passed to volume service: %s", opts.Labels)
		}
		return &csi.Volume{
			ID:       1,
			Name:     opts.Name,
			Size:     opts.MinSize,
			Location: opts.Location,
		}, nil
	}

	req := &proto.CreateVolumeRequest{
		Name: "testvol",
		CapacityRange: &proto.CapacityRange{
			RequiredBytes: MinVolumeSize*GB + 100,
			LimitBytes:    2 * MinVolumeSize * GB,
		},
		Parameters: map[string]string{"labels": "test=this-is-a-very-long-label-which-needs-to-be-truncated-with-important-infos-at-the-end"},
		VolumeCapabilities: []*proto.VolumeCapability{
			{
				AccessType: &proto.VolumeCapability_Mount{
					Mount: &proto.VolumeCapability_MountVolume{},
				},
				AccessMode: &proto.VolumeCapability_AccessMode{
					Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}
	resp, err := env.service.CreateVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetVolume().GetVolumeId() != "1" {
		t.Errorf("unexpected value for VolumeId: %s", resp.GetVolume().GetVolumeId())
	}
	if resp.GetVolume().GetCapacityBytes() != (MinVolumeSize+1)*1024*1024*1024 {
		t.Errorf("unexpected value for CapacityBytes: %d", resp.GetVolume().GetCapacityBytes())
	}
	if len(resp.GetVolume().GetAccessibleTopology()) == 1 {
		top := resp.GetVolume().GetAccessibleTopology()[0]
		if loc := top.GetSegments()[TopologySegmentLocation]; loc != "testloc" {
			t.Errorf("unexpected location segment in topology: %s", loc)
		}
		if env.service.enableProvidedByTopology {
			if provider := top.GetSegments()[ProvidedByLabel]; provider != "cloud" {
				t.Errorf("unexpected provider segment in topology: %s", provider)
			}
		}
	} else {
		t.Errorf("unexpected number of topologies: %d", len(resp.GetVolume().GetAccessibleTopology()))
	}
}

func TestControllerServiceCreateVolumeWithTruncatedLabelStartingWithDash(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.service.enableProvidedByTopology = true

	env.volumeService.CreateFunc = func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
		// Input: "a" + 63x"-" + "b" (65 chars). After truncating to last 63: 62x"-" + "b".
		// After stripping leading non-alphanumeric chars: "b".
		if v, ok := opts.Labels["test"]; !ok || v != "b" {
			t.Errorf("unexpected label value after truncation: %q", v)
		}
		return &csi.Volume{
			ID:       1,
			Name:     opts.Name,
			Size:     opts.MinSize,
			Location: opts.Location,
		}, nil
	}

	req := &proto.CreateVolumeRequest{
		Name: "testvol",
		CapacityRange: &proto.CapacityRange{
			RequiredBytes: MinVolumeSize*GB + 100,
			LimitBytes:    2 * MinVolumeSize * GB,
		},
		Parameters: map[string]string{"labels": "test=a---------------------------------------------------------------b"},
		VolumeCapabilities: []*proto.VolumeCapability{
			{
				AccessType: &proto.VolumeCapability_Mount{
					Mount: &proto.VolumeCapability_MountVolume{},
				},
				AccessMode: &proto.VolumeCapability_AccessMode{
					Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}
	resp, err := env.service.CreateVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetVolume().GetVolumeId() != "1" {
		t.Errorf("unexpected value for VolumeId: %s", resp.GetVolume().GetVolumeId())
	}
}

func TestControllerServiceCreateVolumeWithLocation(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.CreateFunc = func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
		if opts.Location != "explicit" {
			t.Errorf("unexpected location passed to volume service: %s", opts.Location)
		}
		return &csi.Volume{
			ID:       1,
			Name:     opts.Name,
			Size:     opts.MinSize,
			Location: opts.Location,
		}, nil
	}

	req := &proto.CreateVolumeRequest{
		Name: "testvol",
		CapacityRange: &proto.CapacityRange{
			RequiredBytes: 5*GB + 100,
			LimitBytes:    10 * GB,
		},
		VolumeCapabilities: []*proto.VolumeCapability{
			{
				AccessType: &proto.VolumeCapability_Mount{
					Mount: &proto.VolumeCapability_MountVolume{},
				},
				AccessMode: &proto.VolumeCapability_AccessMode{
					Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		AccessibilityRequirements: &proto.TopologyRequirement{
			Preferred: []*proto.Topology{
				{
					Segments: map[string]string{
						TopologySegmentLocation: "explicit",
					},
				},
			},
		},
	}
	resp, err := env.service.CreateVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetVolume().GetAccessibleTopology()) == 1 {
		top := resp.GetVolume().GetAccessibleTopology()[0]
		if loc := top.GetSegments()[TopologySegmentLocation]; loc != "explicit" {
			t.Errorf("unexpected location segment in topology: %s", loc)
		}
	} else {
		t.Errorf("unexpected number of topologies: %d", len(resp.GetVolume().GetAccessibleTopology()))
	}
}

func TestControllerServiceCreateVolumeInputErrors(t *testing.T) {
	env := newControllerServiceTestEnv()

	testCases := []struct {
		Name string
		Req  *proto.CreateVolumeRequest
		Code codes.Code
	}{
		{
			Name: "empty name",
			Req: &proto.CreateVolumeRequest{
				CapacityRange: &proto.CapacityRange{
					RequiredBytes: 5*GB + 100,
					LimitBytes:    10 * GB,
				},
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessType: &proto.VolumeCapability_Mount{
							Mount: &proto.VolumeCapability_MountVolume{},
						},
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "invalid label",
			Req: &proto.CreateVolumeRequest{
				Name: "test",
				CapacityRange: &proto.CapacityRange{
					RequiredBytes: 5*GB + 100,
					LimitBytes:    10 * GB,
				},
				Parameters: map[string]string{"labels": "=test"},
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessType: &proto.VolumeCapability_Mount{
							Mount: &proto.VolumeCapability_MountVolume{},
						},
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "empty capabilities",
			Req: &proto.CreateVolumeRequest{
				Name: "test",
				CapacityRange: &proto.CapacityRange{
					RequiredBytes: 5*GB + 100,
					LimitBytes:    10 * GB,
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "invalid capacity range",
			Req: &proto.CreateVolumeRequest{
				Name: "test",
				CapacityRange: &proto.CapacityRange{
					RequiredBytes: 10 * GB,
					LimitBytes:    5 * GB,
				},
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessType: &proto.VolumeCapability_Mount{
							Mount: &proto.VolumeCapability_MountVolume{},
						},
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			Code: codes.OutOfRange,
		},
		{
			Name: "unsupported capability",
			Req: &proto.CreateVolumeRequest{
				Name: "test",
				CapacityRange: &proto.CapacityRange{
					RequiredBytes: 5 * GB,
					LimitBytes:    10 * GB,
				},
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessType: &proto.VolumeCapability_Mount{
							Mount: &proto.VolumeCapability_MountVolume{},
						},
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "topology requirement without location segment",
			Req: &proto.CreateVolumeRequest{
				Name: "test",
				CapacityRange: &proto.CapacityRange{
					RequiredBytes: 5 * GB,
					LimitBytes:    10 * GB,
				},
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessType: &proto.VolumeCapability_Mount{
							Mount: &proto.VolumeCapability_MountVolume{},
						},
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
				AccessibilityRequirements: &proto.TopologyRequirement{
					Preferred: []*proto.Topology{
						{
							Segments: map[string]string{
								ProvidedByLabel: "cloud",
							},
						},
					},
				},
			},
			Code: codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.CreateVolume(env.ctx, testCase.Req)
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServiceCreateVolumeCreateErrors(t *testing.T) {
	env := newControllerServiceTestEnv()

	testCases := []struct {
		Name        string
		CreateError error
		Code        codes.Code
	}{
		{
			Name:        "volume already exists",
			CreateError: volumes.ErrVolumeAlreadyExists,
			Code:        codes.AlreadyExists,
		},
		{
			Name:        "internal error",
			CreateError: io.EOF,
			Code:        codes.Internal,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			env.volumeService.CreateFunc = func(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
				return nil, testCase.CreateError
			}

			_, err := env.service.CreateVolume(env.ctx, &proto.CreateVolumeRequest{
				Name: "test",
				CapacityRange: &proto.CapacityRange{
					RequiredBytes: 5 * GB,
					LimitBytes:    10 * GB,
				},
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessType: &proto.VolumeCapability_Mount{
							Mount: &proto.VolumeCapability_MountVolume{},
						},
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			})
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServiceDeleteVolume(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.DeleteFunc = func(ctx context.Context, volume *csi.Volume) error {
		if volume.ID != 1 {
			t.Errorf("unexpected volume id passed to volume service: %d", volume.ID)
		}
		return nil
	}

	req := &proto.DeleteVolumeRequest{
		VolumeId: "1",
	}
	_, err := env.service.DeleteVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerServiceDeleteVolumeInputErrors(t *testing.T) {
	env := newControllerServiceTestEnv()

	testCases := []struct {
		Name string
		Req  *proto.DeleteVolumeRequest
		Code codes.Code
	}{
		{
			Name: "empty volume id",
			Req: &proto.DeleteVolumeRequest{
				VolumeId: "",
			},
			Code: codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.DeleteVolume(env.ctx, testCase.Req)
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServiceDeleteVolumeAttached(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.DeleteFunc = func(ctx context.Context, volume *csi.Volume) error {
		return volumes.ErrAttached
	}

	_, err := env.service.DeleteVolume(env.ctx, &proto.DeleteVolumeRequest{
		VolumeId: "1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestControllerServiceDeleteVolumeInvalidID(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.DeleteFunc = func(ctx context.Context, volume *csi.Volume) error {
		return nil
	}

	_, err := env.service.DeleteVolume(env.ctx, &proto.DeleteVolumeRequest{
		VolumeId: "xxx",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerServiceDeleteVolumeInternalError(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.DeleteFunc = func(ctx context.Context, volume *csi.Volume) error {
		return io.EOF
	}

	_, err := env.service.DeleteVolume(env.ctx, &proto.DeleteVolumeRequest{
		VolumeId: "1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestControllerServicePublishVolume(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.GetByIDFunc = func(ctx context.Context, id int64) (*csi.Volume, error) {
		if id != 1 {
			t.Errorf("unexpected volume id passed to volume service: %d", id)
		}
		return &csi.Volume{ID: id, LinuxDevice: "foopath"}, nil
	}

	env.volumeService.AttachFunc = func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
		if volume.ID != 1 {
			t.Errorf("unexpected volume id passed to volume service: %d", volume.ID)
		}
		if server.ID != 2 {
			t.Errorf("unexpected server id passed to volume service: %d", server.ID)
		}
		return nil
	}

	req := &proto.ControllerPublishVolumeRequest{
		VolumeId: "1",
		NodeId:   "2",
		VolumeCapability: &proto.VolumeCapability{
			AccessMode: &proto.VolumeCapability_AccessMode{
				Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}
	resp, err := env.service.ControllerPublishVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	if devicePath := resp.GetPublishContext()["devicePath"]; devicePath != "foopath" {
		t.Errorf("unexpected devicePath returned from publish: %s", devicePath)
	}
}

func TestControllerServicePublishVolumeInputErrors(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.AttachFunc = func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
		return nil
	}

	testCases := []struct {
		Name string
		Req  *proto.ControllerPublishVolumeRequest
		Code codes.Code
	}{
		{
			Name: "readonly",
			Req: &proto.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "2",
				Readonly: true,
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "empty capabilities",
			Req: &proto.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "2",
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "unsupported capability",
			Req: &proto.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "2",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "empty volume id",
			Req: &proto.ControllerPublishVolumeRequest{
				VolumeId: "",
				NodeId:   "2",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "empty node id",
			Req: &proto.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "invalid volume id",
			Req: &proto.ControllerPublishVolumeRequest{
				VolumeId: "abc",
				NodeId:   "2",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			Code: codes.NotFound,
		},
		{
			Name: "invalid node id",
			Req: &proto.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "abc",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			Code: codes.NotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.ControllerPublishVolume(env.ctx, testCase.Req)
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServicePublishVolumeAttachErrors(t *testing.T) {
	env := newControllerServiceTestEnv()

	testCases := []struct {
		Name        string
		AttachError error
		Code        codes.Code
	}{
		{
			Name:        "volume not found",
			AttachError: volumes.ErrVolumeNotFound,
			Code:        codes.NotFound,
		},
		{
			Name:        "server not found",
			AttachError: volumes.ErrServerNotFound,
			Code:        codes.NotFound,
		},
		{
			Name:        "already attached",
			AttachError: volumes.ErrAttached,
			Code:        codes.FailedPrecondition,
		},
		{
			Name:        "attach limit reached",
			AttachError: volumes.ErrAttachLimitReached,
			Code:        codes.ResourceExhausted,
		},
		{
			Name:        "unavailable",
			AttachError: volumes.ErrLockedServer,
			Code:        codes.Unavailable,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			env.volumeService.AttachFunc = func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
				return testCase.AttachError
			}
			_, err := env.service.ControllerPublishVolume(env.ctx, &proto.ControllerPublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "2",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			})
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServiceUnpublishVolume(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.DetachFunc = func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
		if volume.ID != 1 {
			t.Errorf("unexpected volume id passed to volume service: %d", volume.ID)
		}
		if server.ID != 2 {
			t.Errorf("unexpected server id passed to volume service: %d", server.ID)
		}
		return nil
	}

	req := &proto.ControllerUnpublishVolumeRequest{
		VolumeId: "1",
		NodeId:   "2",
	}
	_, err := env.service.ControllerUnpublishVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerServiceUnpublishVolumeNoNode(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.DetachFunc = func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
		if server != nil {
			t.Errorf("unexpected server id passed to volume service: %d", server.ID)
		}
		return nil
	}

	req := &proto.ControllerUnpublishVolumeRequest{
		VolumeId: "1",
		NodeId:   "",
	}
	_, err := env.service.ControllerUnpublishVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerServiceUnpublishVolumeInputErrors(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.DetachFunc = func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
		return nil
	}

	testCases := []struct {
		Name string
		Req  *proto.ControllerUnpublishVolumeRequest
		Code codes.Code
	}{
		{
			Name: "empty volume id",
			Req: &proto.ControllerUnpublishVolumeRequest{
				VolumeId: "",
				NodeId:   "2",
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "invalid volume id",
			Req: &proto.ControllerUnpublishVolumeRequest{
				VolumeId: "abc",
				NodeId:   "2",
			},
			Code: codes.NotFound,
		},
		{
			Name: "invalid node id",
			Req: &proto.ControllerUnpublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "abc",
			},
			Code: codes.NotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.ControllerUnpublishVolume(env.ctx, testCase.Req)
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServiceUnpublishVolumeDetachErrors(t *testing.T) {
	env := newControllerServiceTestEnv()

	testCases := []struct {
		Name        string
		DetachError error
		Code        codes.Code
	}{
		{
			Name:        "server not found",
			DetachError: volumes.ErrServerNotFound,
			Code:        codes.NotFound,
		},
		{
			Name:        "unavailable",
			DetachError: volumes.ErrLockedServer,
			Code:        codes.Unavailable,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			env.volumeService.DetachFunc = func(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
				return testCase.DetachError
			}
			_, err := env.service.ControllerUnpublishVolume(env.ctx, &proto.ControllerUnpublishVolumeRequest{
				VolumeId: "1",
				NodeId:   "2",
			})
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServiceControllerGetCapabilities(t *testing.T) {
	env := newControllerServiceTestEnv()

	resp, err := env.service.ControllerGetCapabilities(env.ctx, &proto.ControllerGetCapabilitiesRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.GetCapabilities()) != 7 {
		t.Fatalf("unexpected number of capabilities: %d", len(resp.GetCapabilities()))
	}
}

func TestControllerServiceCreateSnapshot(t *testing.T) {
	env := newControllerServiceTestEnv()
	created := time.Date(2026, 9, 16, 5, 0, 0, 0, time.UTC)
	env.volumeService.CreateSnapshotFunc = func(_ context.Context, opts volumes.CreateSnapshotOpts) (*csi.Snapshot, error) {
		if opts.Name != "backup" || opts.VolumeID != 42 {
			t.Fatalf("unexpected create options: %+v", opts)
		}
		if opts.Labels[labelKeyManagedBy] != "csi-driver" || opts.Labels["clusterName"] != "myCluster" {
			t.Fatalf("unexpected labels: %v", opts.Labels)
		}
		return &csi.Snapshot{ID: 7, Name: opts.Name, SourceVolumeID: opts.VolumeID, Size: 10, Created: created, Ready: true}, nil
	}

	resp, err := env.service.CreateSnapshot(env.ctx, &proto.CreateSnapshotRequest{Name: "backup", SourceVolumeId: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetSnapshot().GetSnapshotId() != "7" || resp.GetSnapshot().GetSourceVolumeId() != "42" {
		t.Fatalf("unexpected snapshot response: %+v", resp.GetSnapshot())
	}
	if resp.GetSnapshot().GetSizeBytes() != 10*GB || !resp.GetSnapshot().GetReadyToUse() {
		t.Fatalf("unexpected snapshot state: %+v", resp.GetSnapshot())
	}
	if !resp.GetSnapshot().GetCreationTime().AsTime().Equal(created) {
		t.Fatalf("unexpected creation time: %v", resp.GetSnapshot().GetCreationTime())
	}
}

func TestControllerServiceDeleteSnapshot(t *testing.T) {
	env := newControllerServiceTestEnv()
	env.volumeService.DeleteSnapshotFunc = func(_ context.Context, snapshot *csi.Snapshot) error {
		if snapshot.ID != 7 {
			t.Fatalf("unexpected snapshot ID: %d", snapshot.ID)
		}
		return nil
	}
	if _, err := env.service.DeleteSnapshot(env.ctx, &proto.DeleteSnapshotRequest{SnapshotId: "7"}); err != nil {
		t.Fatal(err)
	}
}

func TestControllerServiceListSnapshots(t *testing.T) {
	env := newControllerServiceTestEnv()
	env.volumeService.AllSnapshotsFunc = func(context.Context) ([]*csi.Snapshot, error) {
		return []*csi.Snapshot{
			{ID: 1, SourceVolumeID: 42, Size: 10, Ready: true},
			{ID: 2, SourceVolumeID: 42, Size: 20, Ready: true},
			{ID: 3, SourceVolumeID: 99, Size: 30, Ready: true},
		}, nil
	}

	first, err := env.service.ListSnapshots(env.ctx, &proto.ListSnapshotsRequest{SourceVolumeId: "42", MaxEntries: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.GetEntries()) != 1 || first.GetEntries()[0].GetSnapshot().GetSnapshotId() != "1" || first.GetNextToken() != "1" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	second, err := env.service.ListSnapshots(env.ctx, &proto.ListSnapshotsRequest{SourceVolumeId: "42", StartingToken: first.GetNextToken(), MaxEntries: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.GetEntries()) != 1 || second.GetEntries()[0].GetSnapshot().GetSnapshotId() != "2" || second.GetNextToken() != "" {
		t.Fatalf("unexpected second page: %+v", second)
	}
}

func TestControllerServiceCreateVolumeFromSnapshot(t *testing.T) {
	env := newControllerServiceTestEnv()
	env.volumeService.GetSnapshotByIDFunc = func(_ context.Context, id int64) (*csi.Snapshot, error) {
		return &csi.Snapshot{ID: id, SourceVolumeID: 11, Size: 20, Location: "fsn1", Ready: true}, nil
	}
	env.volumeService.CreateFunc = func(_ context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
		if opts.Snapshot == nil || opts.Snapshot.ID != 7 || opts.MinSize != 20 || opts.Location != "fsn1" {
			t.Fatalf("unexpected restore options: %+v", opts)
		}
		return &csi.Volume{ID: 43, Name: opts.Name, Size: opts.MinSize, Location: opts.Location}, nil
	}

	req := &proto.CreateVolumeRequest{
		Name:          "restored",
		CapacityRange: &proto.CapacityRange{LimitBytes: 25 * GB},
		VolumeCapabilities: []*proto.VolumeCapability{{
			AccessType: &proto.VolumeCapability_Mount{Mount: &proto.VolumeCapability_MountVolume{}},
			AccessMode: &proto.VolumeCapability_AccessMode{Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		}},
		VolumeContentSource: &proto.VolumeContentSource{Type: &proto.VolumeContentSource_Snapshot{
			Snapshot: &proto.VolumeContentSource_SnapshotSource{SnapshotId: "7"},
		}},
	}
	resp, err := env.service.CreateVolume(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetVolume().GetContentSource().GetSnapshot().GetSnapshotId() != "7" {
		t.Fatalf("snapshot source not preserved: %+v", resp.GetVolume())
	}
}

func TestControllerServiceValidateVolumeCapabilities(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.GetByIDFunc = func(ctx context.Context, id int64) (*csi.Volume, error) {
		return &csi.Volume{ID: id}, nil
	}

	testCases := []struct {
		Name string
		Req  *proto.ValidateVolumeCapabilitiesRequest
		Code codes.Code
	}{
		{
			Name: "SINGLE_NODE_WRITER",
			Req: &proto.ValidateVolumeCapabilitiesRequest{
				VolumeId: "1",
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			Code: codes.OK,
		},
		{
			Name: "SINGLE_NODE_MULTI_WRITER",
			Req: &proto.ValidateVolumeCapabilitiesRequest{
				VolumeId: "1",
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_MULTI_WRITER,
						},
					},
				},
			},
			Code: codes.OK,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.ValidateVolumeCapabilities(env.ctx, testCase.Req)
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServiceValidateVolumeCapabilitiesInputErrors(t *testing.T) {
	env := newControllerServiceTestEnv()

	testCases := []struct {
		Name string
		Req  *proto.ValidateVolumeCapabilitiesRequest
		Code codes.Code
	}{
		{
			Name: "empty volume id",
			Req: &proto.ValidateVolumeCapabilitiesRequest{
				VolumeId: "",
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "empty capabilities",
			Req: &proto.ValidateVolumeCapabilitiesRequest{
				VolumeId: "1",
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "invalid volume id",
			Req: &proto.ValidateVolumeCapabilitiesRequest{
				VolumeId: "xxx",
				VolumeCapabilities: []*proto.VolumeCapability{
					{
						AccessMode: &proto.VolumeCapability_AccessMode{
							Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			Code: codes.NotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.ValidateVolumeCapabilities(env.ctx, testCase.Req)
			if status.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestControllerServiceValidateVolumeCapabilitiesVolumeNotFound(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.GetByIDFunc = func(ctx context.Context, id int64) (*csi.Volume, error) {
		return nil, nil
	}

	req := &proto.ValidateVolumeCapabilitiesRequest{
		VolumeId: "1",
		VolumeCapabilities: []*proto.VolumeCapability{
			{
				AccessMode: &proto.VolumeCapability_AccessMode{
					Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}
	_, err := env.service.ValidateVolumeCapabilities(env.ctx, req)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestControllerServiceValidateVolumeCapabilitiesInternalError(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.GetByIDFunc = func(ctx context.Context, id int64) (*csi.Volume, error) {
		return nil, io.EOF
	}

	req := &proto.ValidateVolumeCapabilitiesRequest{
		VolumeId: "1",
		VolumeCapabilities: []*proto.VolumeCapability{
			{
				AccessMode: &proto.VolumeCapability_AccessMode{
					Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}
	_, err := env.service.ValidateVolumeCapabilities(env.ctx, req)
	if status.Code(err) != codes.Internal {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestControllerServiceValidateVolumeCapabilitiesUnsupportedCapability(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.GetByIDFunc = func(ctx context.Context, id int64) (*csi.Volume, error) {
		return &csi.Volume{ID: id}, nil
	}

	req := &proto.ValidateVolumeCapabilitiesRequest{
		VolumeId: "1",
		VolumeCapabilities: []*proto.VolumeCapability{
			{
				AccessMode: &proto.VolumeCapability_AccessMode{
					Mode: proto.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
				},
			},
		},
	}
	resp, err := env.service.ValidateVolumeCapabilities(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetConfirmed() != nil {
		t.Errorf("unexpected confirmation: %v", resp.GetConfirmed())
	}
}
