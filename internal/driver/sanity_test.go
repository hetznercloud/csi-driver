package driver

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-test/v5/pkg/sanity"
	"google.golang.org/grpc"

	"github.com/hetznercloud/csi-driver/internal/csi"
	"github.com/hetznercloud/csi-driver/internal/volumes"
)

func TestSanity(t *testing.T) {
	const endpoint = "csi-sanity.sock"

	if err := os.Remove(endpoint); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(endpoint)

	th := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(th)

	volumeService := volumes.NewIdempotentService(
		logger.With("component", "idempotent-volume-service"),
		&sanityVolumeService{},
	)
	volumeMountService := &sanityMountService{}
	volumeResizeService := &sanityResizeService{}
	volumeStatsService := &sanityStatsService{}

	controllerService := NewControllerService(
		logger.With("component", "driver-controller-service"),
		volumeService,
		"testloc",
		false,
		map[string]string{"clusterName": "myCluster"},
	)

	identityService := NewIdentityService(
		logger.With("component", "driver-identity-service"),
	)

	nodeService := NewNodeService(
		logger.With("component", "driver-node-service"),
		"123456",
		"loc",
		false,
		volumeMountService,
		volumeResizeService,
		volumeStatsService,
	)

	grpcServer := grpc.NewServer()
	proto.RegisterControllerServer(grpcServer, controllerService)
	proto.RegisterIdentityServer(grpcServer, identityService)
	proto.RegisterNodeServer(grpcServer, nodeService)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			fmt.Printf("%s\n", err)
		}
	}()

	tempDir, err := os.MkdirTemp("", "csi")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testConfig := sanity.NewTestConfig()
	testConfig.CreateTargetDir = func(path string) (string, error) {
		return tempDir + "/hcloud-csi-sanity-target", nil
	}
	testConfig.Address = endpoint
	sanity.Test(t, testConfig)
}

type sanityVolumeService struct {
	mu      sync.Mutex
	volumes list.List
}

func (s *sanityVolumeService) All(_ context.Context) ([]*csi.Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vols := []*csi.Volume{}
	for e := s.volumes.Front(); e != nil; e = e.Next() {
		v := e.Value.(*csi.Volume)
		vols = append(vols, v)
	}
	return vols, nil
}

func (s *sanityVolumeService) Create(_ context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for e := s.volumes.Front(); e != nil; e = e.Next() {
		v := e.Value.(*csi.Volume)
		if v.Name == opts.Name {
			return nil, volumes.ErrVolumeAlreadyExists
		}
	}

	volume := &csi.Volume{
		ID:          int64(s.volumes.Len() + 1),
		Name:        opts.Name,
		Size:        opts.MinSize,
		Location:    opts.Location,
		LinuxDevice: fmt.Sprintf("/dev/disk/by-id/scsi-0HC_Volume_%d", s.volumes.Len()+1),
	}

	s.volumes.PushBack(volume)
	return volume, nil
}

func (s *sanityVolumeService) GetByID(_ context.Context, id int64) (*csi.Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for e := s.volumes.Front(); e != nil; e = e.Next() {
		v := e.Value.(*csi.Volume)
		if v.ID == id {
			return v, nil
		}
	}

	return nil, volumes.ErrVolumeNotFound
}

func (s *sanityVolumeService) GetByName(_ context.Context, name string) (*csi.Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for e := s.volumes.Front(); e != nil; e = e.Next() {
		v := e.Value.(*csi.Volume)
		if v.Name == name {
			return v, nil
		}
	}

	return nil, volumes.ErrVolumeNotFound
}

func (s *sanityVolumeService) Delete(_ context.Context, volume *csi.Volume) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for e := s.volumes.Front(); e != nil; e = e.Next() {
		v := e.Value.(*csi.Volume)
		if v.ID == volume.ID {
			s.volumes.Remove(e)
			return nil
		}
	}

	return volumes.ErrVolumeNotFound
}

func (s *sanityVolumeService) Resize(_ context.Context, volume *csi.Volume, size int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for e := s.volumes.Front(); e != nil; e = e.Next() {
		v := e.Value.(*csi.Volume)
		if v.ID == volume.ID {
			v.Size = size
			return nil
		}
	}

	return volumes.ErrVolumeNotFound
}

func (s *sanityVolumeService) Attach(_ context.Context, _ *csi.Volume, _ *csi.Server) error {
	return nil
}

func (s *sanityVolumeService) Detach(_ context.Context, _ *csi.Volume, _ *csi.Server) error {
	return nil
}

type sanityMountService struct{}

func (s *sanityMountService) Publish(_ string, _ string, _ volumes.MountOpts) error {
	return nil
}

func (s *sanityMountService) Unpublish(_ string) error {
	return nil
}

func (s *sanityMountService) PathExists(path string) (bool, error) {
	if path == "some/path" {
		return false, nil
	}
	return true, nil
}

type sanityResizeService struct{}

func (s *sanityResizeService) Resize(volumePath string) error {
	if volumePath == "some/path" {
		return errors.New("path not found")
	}
	return nil
}

type sanityStatsService struct{}

func (s *sanityStatsService) ByteFilesystemStats(_ string) (totalBytes int64, availableBytes int64, usedBytes int64, err error) {
	return 1, 1, 1, nil
}
func (s *sanityStatsService) INodeFilesystemStats(_ string) (total int64, used int64, free int64, err error) {
	return 1, 1, 1, nil
}
