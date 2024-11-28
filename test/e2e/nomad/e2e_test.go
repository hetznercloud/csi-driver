//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
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

func TestVolumeLifecycle(t *testing.T) {
	volReq := &nomad.CSIVolume{
		ID:                   "db-vol",
		Name:                 "db-vol",
		Namespace:            "default",
		PluginID:             "csi.hetzner.cloud",
		RequestedCapacityMin: 10737418240,
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

	var volID int64
	t.Run("volume creation", func(t *testing.T) {
		vol, _, err := cluster.CreateVolume(volReq, &nomad.WriteOptions{})
		if err != nil {
			t.Error(err)
		}
		assert.Len(t, vol, 1)

		volID, err := strconv.ParseInt(vol[0].ExternalID, 10, 64)
		if err != nil {
			t.Error(err)
		}

		hcloudVolume, _, err := cluster.hcloudClient.Volume.GetByID(context.Background(), volID)
		if err != nil {
			t.Error(err)
		}

		assert.NotNilf(t, hcloudVolume, "could not find volume with ID %d on hcloud", volID)
	})

	t.Run("volume deletion", func(t *testing.T) {
		err := cluster.DeleteVolume(volReq.ID, &nomad.WriteOptions{})
		if err != nil {
			t.Error(err)
		}

		hcloudVolume, _, err := cluster.hcloudClient.Volume.GetByID(context.Background(), volID)
		if err != nil {
			t.Error(err)
		}

		assert.Nil(t, hcloudVolume, "hcloud volume was deleted in nomad, but still exists")
	})

}
