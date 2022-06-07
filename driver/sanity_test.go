package driver

import (
	"container/list"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/kit/log"
	"github.com/kubernetes-csi/csi-test/v3/pkg/sanity"
	"google.golang.org/grpc"

	"github.com/hetznercloud/csi-driver/csi"
	"github.com/hetznercloud/csi-driver/volumes"
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

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	volumeService := volumes.NewIdempotentService(
		log.With(logger, "component", "idempotent-volume-service"),
		&sanityVolumeService{},
	)
	volumeMountService := &sanityMountService{}
	volumeResizeService := &sanityResizeService{}
	volumeStatsService := &sanityStatsService{}
	controllerService := NewControllerService(
		log.With(logger, "component", "driver-controller-service"),
		volumeService,
		"testloc",
	)
	identityService := NewIdentityService(
		log.With(logger, "component", "driver-identity-service"),
	)
	nodeService := NewNodeService(
		log.With(logger, "component", "driver-node-service"),
		"123456",
		"loc",
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

	tempDir, err := ioutil.TempDir("", "csi")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testConfig := sanity.NewTestConfig()
	testConfig.TargetPath = tempDir + "/hcloud-csi-sanity-target"
	testConfig.Address = endpoint
	sanity.Test(t, testConfig)
}

type sanityVolumeService struct {
	mu      sync.Mutex
	volumes list.List
}

func (s *sanityVolumeService) Create(ctx context.Context, opts volumes.CreateOpts) (*csi.Volume, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for e := s.volumes.Front(); e != nil; e = e.Next() {
		v := e.Value.(*csi.Volume)
		if v.Name == opts.Name {
			return nil, volumes.ErrVolumeAlreadyExists
		}
	}

	volume := &csi.Volume{
		ID:          uint64(s.volumes.Len() + 1),
		Name:        opts.Name,
		Size:        opts.MinSize,
		Location:    opts.Location,
		LinuxDevice: fmt.Sprintf("/dev/disk/by-id/scsi-0HC_Volume_%d", s.volumes.Len()+1),
	}

	s.volumes.PushBack(volume)
	return volume, nil
}

func (s *sanityVolumeService) GetByID(ctx context.Context, id uint64) (*csi.Volume, error) {
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

func (s *sanityVolumeService) GetByName(ctx context.Context, name string) (*csi.Volume, error) {
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

func (s *sanityVolumeService) Delete(ctx context.Context, volume *csi.Volume) error {
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

func (s *sanityVolumeService) Resize(ctx context.Context, volume *csi.Volume, size int) error {
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

func (s *sanityVolumeService) Attach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	return nil
}

func (s *sanityVolumeService) Detach(ctx context.Context, volume *csi.Volume, server *csi.Server) error {
	return nil
}

type sanityMountService struct{}

func (s *sanityMountService) Publish(targetPath string, devicePath string, opts volumes.MountOpts) error {
	return nil
}

func (s *sanityMountService) Unpublish(targetPath string) error {
	return nil
}

func (s *sanityMountService) PathExists(path string) (bool, error) {
	if path == "some/path" {
		return false, nil
	}
	return true, nil
}

func (s *sanityMountService) FormatDisk(disk string, fstype string) error {
	return nil
}

func (s *sanityMountService) DetectDiskFormat(disk string) (string, error) {
	return "ext4", nil
}

type sanityResizeService struct{}

func (s *sanityResizeService) Resize(volumePath string) error {
	return nil
}

type sanityStatsService struct{}

func (s *sanityStatsService) ByteFilesystemStats(volumePath string) (totalBytes int64, availableBytes int64, usedBytes int64, err error) {
	return 1, 1, 1, nil
}
func (s *sanityStatsService) INodeFilesystemStats(volumePath string) (total int64, used int64, free int64, err error) {
	return 1, 1, 1, nil
}
