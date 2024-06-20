package driver

import (
	"context"
	"testing"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-kit/log"
)

var _ proto.IdentityServer = (*IdentityService)(nil)

type identityServiceTestEnv struct {
	ctx     context.Context
	service *IdentityService
}

func newIdentityServerTestEnv() identityServiceTestEnv {
	return identityServiceTestEnv{
		ctx: context.Background(),
		service: NewIdentityService(
			log.NewNopLogger(),
		),
	}
}

func TestIdentityServiceGetPluginInfo(t *testing.T) {
	env := newIdentityServerTestEnv()

	resp, err := env.service.GetPluginInfo(env.ctx, &proto.GetPluginInfoRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != PluginName {
		t.Errorf("unexpected name: %s", resp.Name)
	}
	if resp.VendorVersion != PluginVersion {
		t.Errorf("unexpected version: %s", resp.VendorVersion)
	}
}

func TestIdentityServiceGetPluginCapabilities(t *testing.T) {
	env := newIdentityServerTestEnv()

	resp, err := env.service.GetPluginCapabilities(env.ctx, &proto.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if c := len(resp.Capabilities); c != 4 {
		t.Fatalf("unexpected number of capabilities: %d", c)
	}

	cap1service := resp.Capabilities[0].GetService()
	if cap1service == nil {
		t.Fatalf("unexpected capability at index 0: %v", resp.Capabilities[0])
	}
	if cap1service.Type != proto.PluginCapability_Service_CONTROLLER_SERVICE {
		t.Errorf("unexpected service type: %s", cap1service.Type)
	}

	cap2service := resp.Capabilities[1].GetService()
	if cap2service == nil {
		t.Fatalf("unexpected capability at index 1: %v", resp.Capabilities[1])
	}
	if cap2service.Type != proto.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS {
		t.Errorf("unexpected service type: %s", cap2service.Type)
	}

	cap3volume := resp.Capabilities[2].GetVolumeExpansion()
	if cap3volume == nil {
		t.Fatalf("unexpected capability at index 2: %v", resp.Capabilities[2])
	}
	if cap3volume.Type != proto.PluginCapability_VolumeExpansion_ONLINE {
		t.Errorf("unexpected volume expansion type: %s", cap3volume.Type)
	}

	cap4volume := resp.Capabilities[3].GetVolumeExpansion()
	if cap4volume == nil {
		t.Fatalf("unexpected capability at index 3: %v", resp.Capabilities[3])
	}
	if cap4volume.Type != proto.PluginCapability_VolumeExpansion_OFFLINE {
		t.Errorf("unexpected volume expansion type: %s", cap4volume.Type)
	}
}

func TestIdentityServiceProbeReady(t *testing.T) {
	env := newIdentityServerTestEnv()

	env.service.SetReady(true)

	resp, err := env.service.Probe(env.ctx, &proto.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Ready.Value {
		t.Error("expected to be ready")
	}
}

func TestIdentityServiceProbeNotReady(t *testing.T) {
	env := newIdentityServerTestEnv()

	env.service.SetReady(false)

	resp, err := env.service.Probe(env.ctx, &proto.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Ready.Value {
		t.Error("expected to not be ready")
	}
}
