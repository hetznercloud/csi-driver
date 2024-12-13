package e2e

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	nomad "github.com/hashicorp/nomad/api"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const InitialVolumeCapacity = 10737418240 // 10GiB

type Cluster struct {
	hcloudClient *hcloud.Client
	nomadClient  *nomad.Client

	volumesCreated map[string]struct{}
	lock           sync.Mutex
}

func NewCluster() (*Cluster, error) {
	token := os.Getenv("HCLOUD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("HCLOUD_TOKEN env variable is not set")
	}

	hcloudOpts := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithApplication("nomad-csi-e2e", "v2.11.0"), // x-releaser-pleaser-version
		hcloud.WithPollOpts(hcloud.PollOpts{
			BackoffFunc: hcloud.ExponentialBackoffWithOpts(hcloud.ExponentialBackoffOpts{
				Base:       time.Second,
				Multiplier: 2,
				Cap:        10 * time.Second,
			}),
		}),
	}

	hcloudClient := hcloud.NewClient(hcloudOpts...)
	if hcloudClient == nil {
		return nil, fmt.Errorf("hcloud client could not be initialized")
	}

	nomadAddr := os.Getenv("NOMAD_ADDR")
	if nomadAddr == "" {
		return nil, fmt.Errorf("NOMAD_ADDR env variable is not set")
	}
	nomadCACert := os.Getenv("NOMAD_CACERT")
	if nomadCACert == "" {
		return nil, fmt.Errorf("NOMAD_CACERT env variable is not set")
	}
	nomadClientCert := os.Getenv("NOMAD_CLIENT_CERT")
	if nomadClientCert == "" {
		return nil, fmt.Errorf("NOMAD_CLIENT_CERT env variable is not set")
	}
	nomadClientKey := os.Getenv("NOMAD_CLIENT_KEY")
	if nomadClientKey == "" {
		return nil, fmt.Errorf("NOMAD_CLIENT_KEY env variable is not set")
	}

	nomadConfig := nomad.DefaultConfig()

	nomadClient, err := nomad.NewClient(nomadConfig)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		hcloudClient:   hcloudClient,
		nomadClient:    nomadClient,
		volumesCreated: make(map[string]struct{}),
		lock:           sync.Mutex{},
	}, nil
}

func (cluster *Cluster) Cleanup() []error {
	var cleanupErrors []error

	for volName := range cluster.volumesCreated {
		vol, _, err := cluster.hcloudClient.Volume.GetByName(context.Background(), volName)
		if err != nil {
			cleanupErrors = append(cleanupErrors, err)
			continue
		}
		if vol == nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("volume %s not found on hcloud", volName))
			continue
		}
		_, err = cluster.hcloudClient.Volume.Delete(context.Background(), vol)
		if err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}

	return cleanupErrors
}

func (cluster *Cluster) CreateVolume(volReq *nomad.CSIVolume, w *nomad.WriteOptions) ([]*nomad.CSIVolume, *nomad.WriteMeta, error) {
	vol, meta, err := cluster.nomadClient.CSIVolumes().Create(volReq, w)
	if err != nil {
		return nil, nil, err
	}

	cluster.lock.Lock()
	defer cluster.lock.Unlock()

	cluster.volumesCreated[volReq.ID] = struct{}{}

	return vol, meta, err
}

func (cluster *Cluster) DeleteVolume(externalVolID string, w *nomad.WriteOptions) error {
	err := cluster.nomadClient.CSIVolumes().Delete(externalVolID, w)
	if err != nil {
		return err
	}

	cluster.lock.Lock()
	defer cluster.lock.Unlock()

	delete(cluster.volumesCreated, externalVolID)

	return nil
}

func (cluster *Cluster) ExecInAlloc(alloc *nomad.Allocation, task string, command []string) (int, error) {
	exitCode, err := cluster.nomadClient.Allocations().Exec(
		context.Background(),
		alloc,
		task,
		true,
		command,
		os.Stdin,
		os.Stdout,
		os.Stderr,
		make(<-chan nomad.TerminalSize),
		&nomad.QueryOptions{},
	)
	if err != nil {
		return exitCode, err
	}
	return exitCode, nil
}

func (cluster *Cluster) WaitForRunningJob(jobID string) (*nomad.AllocationListStub, error) {
	for retry := range 10 {
		allocs, _, err := cluster.nomadClient.Jobs().Allocations(
			jobID,
			false,
			&nomad.QueryOptions{},
		)
		if err != nil {
			return nil, err
		}

		for _, alloc := range allocs {
			if alloc.ClientStatus == "running" {
				return alloc, nil
			}
		}

		backoffSleep(retry)
	}
	return nil, fmt.Errorf("no running allocation for job %s", jobID)
}

func CreateVolumeSpec(id string) *nomad.CSIVolume {
	return &nomad.CSIVolume{
		ID:                   id,
		Name:                 id,
		Namespace:            "default",
		PluginID:             "csi.hetzner.cloud",
		RequestedCapacityMin: InitialVolumeCapacity,
		RequestedCapabilities: []*nomad.CSIVolumeCapability{
			{
				AccessMode:     "single-node-writer",
				AttachmentMode: "file-system",
			},
		},
		MountOptions: &nomad.CSIMountOptions{
			FSType: "ext4",
			MountFlags: []string{
				"discard",
				"defaults",
			},
		},
	}
}

func CreateBusyboxWithVolumeJobSpec(id string, volumeID string, mountPath string) *nomad.Job {
	job := nomad.NewServiceJob(id, id, "global", 50)
	taskGroup := nomad.NewTaskGroup(id, 1)

	taskGroup.Volumes = map[string]*nomad.VolumeRequest{
		volumeID: {
			Name:           volumeID,
			ReadOnly:       false,
			Type:           "csi",
			Source:         volumeID,
			AttachmentMode: "file-system",
			AccessMode:     "single-node-writer",
			PerAlloc:       false,
		},
	}

	task := nomad.NewTask(id, "docker")
	task = task.SetConfig("image", "busybox:stable")
	task = task.SetConfig("command", "sleep")
	task = task.SetConfig("args", []string{"3600"})

	task.VolumeMounts = append(task.VolumeMounts, &nomad.VolumeMount{
		Volume:      &volumeID,
		Destination: &mountPath,
	})

	taskGroup = taskGroup.AddTask(task)
	job = job.AddTaskGroup(taskGroup)
	return job
}

func backoffSleep(retry int) {
	delay := math.Pow(2, float64(retry))
	delay = math.Min(delay, 16)
	time.Sleep(time.Second * time.Duration(delay))
}
