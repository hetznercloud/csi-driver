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
    initialCapacity = 10737418240 // 10GiB
    resizedCapacity = 11811160064 // 11GiB
    resizedCapacityGB = 11
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
	volReq := &nomad.CSIVolume{
		ID:                   "db-vol",
		Name:                 "db-vol",
		Namespace:            "default",
		PluginID:             "csi.hetzner.cloud",
		RequestedCapacityMin: initialCapacity,
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
        volReq.RequestedCapacityMin = resizedCapacity 

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
                resizedCapacityGB,
                "Expected vol size %d, but got %d",
                resizedCapacityGB,
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

