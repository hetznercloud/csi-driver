package driver

import (
	"context"
	"io"
	"testing"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/log"
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
	logger := log.NewNopLogger()
	volumeService := &mock.VolumeService{}

	return &controllerServiceTestEnv{
		ctx: context.Background(),
		service: NewControllerService(
			logger,
			volumeService,
			"testloc",
		),
		volumeService: volumeService,
	}
}

func TestControllerServiceCreateVolume(t *testing.T) {
	env := newControllerServiceTestEnv()

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
	if resp.Volume.VolumeId != "1" {
		t.Errorf("unexpected value for VolumeId: %s", resp.Volume.VolumeId)
	}
	if resp.Volume.CapacityBytes != (MinVolumeSize+1)*1024*1024*1024 {
		t.Errorf("unexpected value for CapacityBytes: %d", resp.Volume.CapacityBytes)
	}
	if len(resp.Volume.AccessibleTopology) == 1 {
		top := resp.Volume.AccessibleTopology[0]
		if loc := top.Segments[TopologySegmentLocation]; loc != "testloc" {
			t.Errorf("unexpected location segment in topology: %s", loc)
		}
	} else {
		t.Errorf("unexpected number of topologies: %d", len(resp.Volume.AccessibleTopology))
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
	if len(resp.Volume.AccessibleTopology) == 1 {
		top := resp.Volume.AccessibleTopology[0]
		if loc := top.Segments[TopologySegmentLocation]; loc != "explicit" {
			t.Errorf("unexpected location segment in topology: %s", loc)
		}
	} else {
		t.Errorf("unexpected number of topologies: %d", len(resp.Volume.AccessibleTopology))
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

	if devicePath := resp.PublishContext["devicePath"]; devicePath != "foopath" {
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

	if len(resp.Capabilities) != 4 {
		t.Fatalf("unexpected number of capabilities: %d", len(resp.Capabilities))
	}
}

func TestControllerServiceValidateVolumeCapabilities(t *testing.T) {
	env := newControllerServiceTestEnv()

	env.volumeService.GetByIDFunc = func(ctx context.Context, id int64) (*csi.Volume, error) {
		return &csi.Volume{ID: id}, nil
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
	resp, err := env.service.ValidateVolumeCapabilities(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Confirmed == nil {
		t.Fatal("expected confirmation")
	}
	if len(resp.Confirmed.VolumeCapabilities) != 1 {
		t.Errorf("unexpected confirmed capabilities: %v", resp.Confirmed.VolumeCapabilities)
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
	if resp.Confirmed != nil {
		t.Errorf("unexpected confirmation: %v", resp.Confirmed)
	}
}
