package driver

import (
	"context"
	"io"
	"testing"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/hetznercloud/csi-driver/mock"
	"github.com/hetznercloud/csi-driver/volumes"
)

var _ proto.NodeServer = (*NodeService)(nil)

type nodeServiceTestEnv struct {
	ctx                 context.Context
	service             *NodeService
	volumeMountService  *mock.VolumeMountService
	volumeResizeService *mock.VolumeResizeService
}

func newNodeServerTestEnv() nodeServiceTestEnv {
	var (
		volumeMountService  = &mock.VolumeMountService{}
		volumeResizeService = &mock.VolumeResizeService{}
		volumeStatsService  = &mock.VolumeStatsService{}
	)
	return nodeServiceTestEnv{
		ctx: context.Background(),
		service: NewNodeService(
			log.NewNopLogger(),
			"1",
			"loc",
			volumeMountService,
			volumeResizeService,
			volumeStatsService,
		),
		volumeMountService:  volumeMountService,
		volumeResizeService: volumeResizeService,
	}
}

func TestNodeServiceNodePublishVolume(t *testing.T) {
	env := newNodeServerTestEnv()

	env.volumeMountService.PublishFunc = func(targetPath string, devicePath string, opts volumes.MountOpts) error {
		if targetPath != "target" {
			t.Errorf("unexpected target path passed to volume service: %s", targetPath)
		}
		if devicePath != "devpath" {
			t.Errorf("unexpected device path passed to volume mount service: %s", devicePath)
		}
		return nil
	}

	_, err := env.service.NodePublishVolume(env.ctx, &proto.NodePublishVolumeRequest{
		VolumeId:   "1",
		TargetPath: "target",
		VolumeCapability: &proto.VolumeCapability{
			AccessMode: &proto.VolumeCapability_AccessMode{
				Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
			AccessType: &proto.VolumeCapability_Mount{
				Mount: &proto.VolumeCapability_MountVolume{
					FsType:     "ext4",
					MountFlags: []string{"flag1", "flag2"},
				},
			},
		},
		PublishContext: map[string]string{
			"devicePath": "devpath",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNodeServiceNodePublishBlockVolume(t *testing.T) {
	env := newNodeServerTestEnv()

	env.volumeMountService.PublishFunc = func(
		targetPath, devicePath string, opts volumes.MountOpts,
	) error {
		if targetPath != "target" {
			t.Errorf("unexpected target path: %s", targetPath)
		}
		if devicePath != "devpath" {
			t.Errorf("unexpected device path: %s", devicePath)
		}
		return nil
	}

	_, err := env.service.NodePublishVolume(env.ctx, &proto.NodePublishVolumeRequest{
		VolumeId:   "1",
		TargetPath: "target",
		VolumeCapability: &proto.VolumeCapability{
			AccessMode: &proto.VolumeCapability_AccessMode{
				Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
			AccessType: &proto.VolumeCapability_Block{
				Block: &proto.VolumeCapability_BlockVolume{},
			},
		},
		PublishContext: map[string]string{"devicePath": "devpath"},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestNodeServiceNodePublishPublishError(t *testing.T) {
	env := newNodeServerTestEnv()

	env.volumeMountService.PublishFunc = func(targetPath string, devicePath string, opts volumes.MountOpts) error {
		return io.EOF
	}

	_, err := env.service.NodePublishVolume(env.ctx, &proto.NodePublishVolumeRequest{
		VolumeId:   "1",
		TargetPath: "target",
		VolumeCapability: &proto.VolumeCapability{
			AccessMode: &proto.VolumeCapability_AccessMode{
				Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
			AccessType: &proto.VolumeCapability_Mount{
				Mount: &proto.VolumeCapability_MountVolume{
					FsType:     "ext4",
					MountFlags: []string{"flag1", "flag2"},
				},
			},
		},
		PublishContext: map[string]string{
			"devicePath": "devpath",
		},
	})
	if grpc.Code(err) != codes.Internal {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNodeServiceNodePublishVolumeInputErrors(t *testing.T) {
	env := newNodeServerTestEnv()

	testCases := []struct {
		Name string
		Req  *proto.NodePublishVolumeRequest
		Code codes.Code
	}{
		{
			Name: "empty volume id",
			Req: &proto.NodePublishVolumeRequest{
				TargetPath: "target",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
					AccessType: &proto.VolumeCapability_Mount{
						Mount: &proto.VolumeCapability_MountVolume{
							FsType:     "ext4",
							MountFlags: []string{"flag1", "flag2"},
						},
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "empty target path",
			Req: &proto.NodePublishVolumeRequest{
				VolumeId: "1",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
					AccessType: &proto.VolumeCapability_Mount{
						Mount: &proto.VolumeCapability_MountVolume{
							FsType:     "ext4",
							MountFlags: []string{"flag1", "flag2"},
						},
					},
				},
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "no mount access type",
			Req: &proto.NodePublishVolumeRequest{
				VolumeId:   "1",
				TargetPath: "target",
				VolumeCapability: &proto.VolumeCapability{
					AccessMode: &proto.VolumeCapability_AccessMode{
						Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			Code: codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.NodePublishVolume(env.ctx, testCase.Req)
			if grpc.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

func TestNodeServiceNodeUnpublishVolume(t *testing.T) {
	env := newNodeServerTestEnv()

	env.volumeMountService.UnpublishFunc = func(targetPath string) error {
		if targetPath != "target" {
			t.Errorf("unexpected target path passed to volume service: %s", targetPath)
		}
		return nil
	}

	_, err := env.service.NodeUnpublishVolume(env.ctx, &proto.NodeUnpublishVolumeRequest{
		VolumeId:   "1",
		TargetPath: "target",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNodeServiceNodeUnpublishUnpublishError(t *testing.T) {
	env := newNodeServerTestEnv()

	env.volumeMountService.UnpublishFunc = func(targetPath string) error {
		return io.EOF
	}

	_, err := env.service.NodeUnpublishVolume(env.ctx, &proto.NodeUnpublishVolumeRequest{
		VolumeId:   "1",
		TargetPath: "target",
	})
	if grpc.Code(err) != codes.Internal {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNodeServiceNodeUnpublishVolumeInputErrors(t *testing.T) {
	env := newNodeServerTestEnv()

	testCases := []struct {
		Name string
		Req  *proto.NodeUnpublishVolumeRequest
		Code codes.Code
	}{
		{
			Name: "empty volume id",
			Req: &proto.NodeUnpublishVolumeRequest{
				TargetPath: "target",
			},
			Code: codes.InvalidArgument,
		},
		{
			Name: "empty target path",
			Req: &proto.NodeUnpublishVolumeRequest{
				VolumeId: "1",
			},
			Code: codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.NodeUnpublishVolume(env.ctx, testCase.Req)
			if grpc.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}

func TestNodeServiceNodeGetCapabilities(t *testing.T) {
	env := newNodeServerTestEnv()

	req := &proto.NodeGetCapabilitiesRequest{}
	resp, err := env.service.NodeGetCapabilities(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if c := len(resp.Capabilities); c != 2 {
		t.Fatalf("unexpected number of capabilities: %d", c)
	}

	caprpc := resp.Capabilities[0].GetRpc()
	if caprpc == nil {
		t.Fatal("unexpected capability at index 0")
	}
	if caprpc.Type != proto.NodeServiceCapability_RPC_EXPAND_VOLUME {
		t.Errorf("unexpected type: %s", caprpc.Type)
	}

	caprpc = resp.Capabilities[1].GetRpc()
	if caprpc == nil {
		t.Fatal("unexpected capability at index 1")
	}
	if caprpc.Type != proto.NodeServiceCapability_RPC_GET_VOLUME_STATS {
		t.Errorf("unexpected type: %s", caprpc.Type)
	}
}

func TestNodeServiceNodeGetInfo(t *testing.T) {
	env := newNodeServerTestEnv()

	req := &proto.NodeGetInfoRequest{}
	resp, err := env.service.NodeGetInfo(env.ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.NodeId != "1" {
		t.Errorf("unexpected node id: %s", resp.NodeId)
	}
	if resp.MaxVolumesPerNode != MaxVolumesPerNode {
		t.Errorf("unexpected max volumes per node: %d", resp.MaxVolumesPerNode)
	}
}

func TestNodeServiceNodeExpandVolume(t *testing.T) {
	env := newNodeServerTestEnv()

	env.volumeMountService.PathExistsFunc = func(path string) (bool, error) {
		if path != "volumePath" {
			t.Errorf("unexpected volume path passed to volume mount service: %s", path)
		}
		return true, nil
	}
	env.volumeResizeService.ResizeFunc = func(volumePath string) error {
		if volumePath != "volumePath" {
			t.Errorf("unexpected volume path passed to volume service: %s", volumePath)
		}
		return nil
	}

	_, err := env.service.NodeExpandVolume(env.ctx, &proto.NodeExpandVolumeRequest{
		VolumeId:   "1",
		VolumePath: "volumePath",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNodeServiceNodeExpandBlockVolume(t *testing.T) {
	env := newNodeServerTestEnv()

	env.volumeMountService.PathExistsFunc = func(path string) (bool, error) {
		if path != "volumePath" {
			t.Errorf("unexpected volume path passed to volume mount service: %s", path)
		}
		return true, nil
	}
	env.volumeResizeService.ResizeFunc = func(volumePath string) error {
		t.Errorf("This function should never be called.")
		return nil
	}

	_, err := env.service.NodeExpandVolume(env.ctx, &proto.NodeExpandVolumeRequest{
		VolumeId:   "1",
		VolumePath: "volumePath",
		VolumeCapability: &proto.VolumeCapability{
			AccessMode: &proto.VolumeCapability_AccessMode{
				Mode: proto.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
			AccessType: &proto.VolumeCapability_Block{Block: &proto.VolumeCapability_BlockVolume{}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNodeServiceNodeExpandVolumeInputErrors(t *testing.T) {
	env := newNodeServerTestEnv()

	testCases := []struct {
		Name string
		Req  *proto.NodeExpandVolumeRequest
		Code codes.Code
	}{
		{
			Name: "empty volume path",
			Req: &proto.NodeExpandVolumeRequest{
				VolumeId: "1",
			},
			Code: codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			_, err := env.service.NodeExpandVolume(env.ctx, testCase.Req)
			if grpc.Code(err) != testCase.Code {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
