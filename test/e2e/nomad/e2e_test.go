//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/hetznercloud/csi-driver/internal/driver"
	"github.com/stretchr/testify/assert"
)

const (
	ResizedCapacity   = 11811160064 // 11GiB
	ResizedCapacityGB = 11
)

var cluster *Cluster

func TestMain(m *testing.M) {
	var err error
	cluster, err = NewCluster()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()

	if err := cluster.Cleanup(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

func TestGetPluginInfo(t *testing.T) {
	plugin, _, err := cluster.nomadClient.CSIPlugins().Info(driver.PluginName, &nomad.QueryOptions{})
	if err != nil {
		t.Error(err)
	}

	assert.NotNil(t, plugin, "Expected plugin from Nomad to be not nil")

	assert.Equalf(
		t,
		plugin.Version,
		driver.PluginVersion,
		"Expected plugin version %s, but got %s",
		driver.PluginVersion,
		plugin.Version,
	)
}

func TestVolumeLifecycle(t *testing.T) {
	volReq := CreateVolumeSpec("db-vol")

	var hcloudVolID int64
	t.Run("volume creation", func(t *testing.T) {
		vol, _, err := cluster.CreateVolume(volReq, &nomad.WriteOptions{})
		if err != nil {
			t.Error(err)
		}
		assert.Len(t, vol, 1)

		hcloudVolID, err = strconv.ParseInt(vol[0].ExternalID, 10, 64)
		if err != nil {
			t.Error(err)
		}

		hcloudVolume, _, err := cluster.hcloudClient.Volume.GetByID(context.Background(), hcloudVolID)
		if err != nil {
			t.Error(err)
		}

		assert.NotNilf(t, hcloudVolume, "could not find volume with ID %d on hcloud", hcloudVolID)
	})

	t.Run("volume resize", func(t *testing.T) {
		volReq.RequestedCapacityMin = ResizedCapacity

		_, _, err := cluster.nomadClient.CSIVolumes().Create(volReq, &nomad.WriteOptions{})
		if err != nil {
			t.Error(err)
		}

		hcloudVolume, _, err := cluster.hcloudClient.Volume.GetByID(context.Background(), hcloudVolID)
		if err != nil {
			t.Error(err)
		}

		if assert.NotNilf(t, hcloudVolume, "could not find volume with ID %d on hcloud", hcloudVolID) {
			assert.Equalf(
				t,
				hcloudVolume.Size,
				ResizedCapacityGB,
				"Expected vol size %d, but got %d",
				ResizedCapacityGB,
				hcloudVolume.Size,
			)
		}
	})

	t.Run("volume deletion", func(t *testing.T) {
		err := cluster.DeleteVolume(volReq.ID, &nomad.WriteOptions{})
		if err != nil {
			t.Error(err)
		}

		hcloudVolume, _, err := cluster.hcloudClient.Volume.GetByID(context.Background(), hcloudVolID)
		if err != nil {
			t.Error(err)
		}

		assert.Nil(t, hcloudVolume, "hcloud volume was deleted in nomad, but still exists")
	})
}

func TestVolumeWrite(t *testing.T) {
	volID := "test-vol"
	jobID := "test-writer"
	volReq := CreateVolumeSpec(volID)
	job := CreateBusyboxWithVolumeJobSpec(jobID, volID, "/test")

	t.Run("create volume", func(t *testing.T) {
		vol, _, err := cluster.CreateVolume(volReq, &nomad.WriteOptions{})
		if err != nil {
			t.Error(err)
		}
		assert.Len(t, vol, 1)
	})

    // Used to ensure that the job for verifying the data is scheduled on another node
    var previousNodeID string
	t.Run("write to volume", func(t *testing.T) {
		_, _, err := cluster.nomadClient.Jobs().Register(job, &nomad.WriteOptions{})
		if err != nil {
			t.Error(err)
		}
		defer func() {
			_, _, err = cluster.nomadClient.Jobs().Deregister(*job.ID, true, &nomad.WriteOptions{})
			if err != nil {
				t.Error(err)
			}
		}()

		allocStub, err := cluster.WaitForRunningJob(*job.ID)
		if err != nil {
			t.Error(err)
			return
		}

        previousNodeID = allocStub.NodeID

		alloc, _, err := cluster.nomadClient.Allocations().Info(allocStub.ID, &nomad.QueryOptions{})
		if err != nil {
			t.Error(err)
			return
		}

		exitCode, err := cluster.ExecInAlloc(alloc, jobID, []string{
            "dd",
            "if=/dev/random",
            "of=/test/data",
            "bs=1M",
            "count=1",
        })
		if err != nil {
			t.Error(err)
		}
		assert.Equalf(t, 0, exitCode, "could not write test data - exit code: %d", exitCode)
	})

	t.Run("verify volume data", func(t *testing.T) {
        // try to schedule job on another node
        constraint := &nomad.Affinity{
            LTarget: "${node.unique.id}",
            RTarget: previousNodeID,
            Operand: "!=",
        }
        job.Affinities = append(job.Affinities, constraint)

		_, _, err := cluster.nomadClient.Jobs().Register(job, &nomad.WriteOptions{})
		if err != nil {
			t.Error(err)
		}
		defer func() {
			_, _, err = cluster.nomadClient.Jobs().Deregister(*job.ID, true, &nomad.WriteOptions{})
			if err != nil {
				t.Error(err)
			}
		}()

		allocStub, err := cluster.WaitForRunningJob(*job.ID)
		if err != nil {
			t.Error(err)
			return
		}

		alloc, _, err := cluster.nomadClient.Allocations().Info(allocStub.ID, &nomad.QueryOptions{})
		if err != nil {
			t.Error(err)
			return
		}

        // verify that file exists and has a size greater than zero
		exitCode, err := cluster.ExecInAlloc(alloc, jobID, []string{
            "test",
            "-s",
            "/test/data",
        })
		if err != nil {
			t.Error(err)
		}
		assert.Equalf(t, 0, exitCode, "could not verify test data - exit code: %d", exitCode)
	})

	t.Run("delete volume", func(t *testing.T) {
        // with retries, as volume can still be in use for a couple of seconds after job got deleted,
        // which results in a internal server error
		for i := range 10 {
			if err := cluster.DeleteVolume(volReq.ID, &nomad.WriteOptions{}); err == nil {
				break
			}
			backoffSleep(i)
		}
	})
}
