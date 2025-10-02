package driver

import (
	"context"
	"log/slog"
	"testing"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
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
			slog.New(slog.DiscardHandler),
		),
	}
}

func TestIdentityServiceGetPluginInfo(t *testing.T) {
	env := newIdentityServerTestEnv()

	resp, err := env.service.GetPluginInfo(env.ctx, &proto.GetPluginInfoRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetName() != PluginName {
		t.Errorf("unexpected name: %s", resp.GetName())
	}
	if resp.GetVendorVersion() != PluginVersion {
		t.Errorf("unexpected version: %s", resp.GetVendorVersion())
	}
}

func TestIdentityServiceGetPluginCapabilities(t *testing.T) {
	env := newIdentityServerTestEnv()

	resp, err := env.service.GetPluginCapabilities(env.ctx, &proto.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if c := len(resp.GetCapabilities()); c != 4 {
		t.Fatalf("unexpected number of capabilities: %d", c)
	}

	cap1service := resp.GetCapabilities()[0].GetService()
	if cap1service == nil {
		t.Fatalf("unexpected capability at index 0: %v", resp.GetCapabilities()[0])
	}
	if cap1service.GetType() != proto.PluginCapability_Service_CONTROLLER_SERVICE {
		t.Errorf("unexpected service type: %s", cap1service.GetType())
	}

	cap2service := resp.GetCapabilities()[1].GetService()
	if cap2service == nil {
		t.Fatalf("unexpected capability at index 1: %v", resp.GetCapabilities()[1])
	}
	if cap2service.GetType() != proto.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS {
		t.Errorf("unexpected service type: %s", cap2service.GetType())
	}

	cap3volume := resp.GetCapabilities()[2].GetVolumeExpansion()
	if cap3volume == nil {
		t.Fatalf("unexpected capability at index 2: %v", resp.GetCapabilities()[2])
	}
	if cap3volume.GetType() != proto.PluginCapability_VolumeExpansion_ONLINE {
		t.Errorf("unexpected volume expansion type: %s", cap3volume.GetType())
	}

	cap4volume := resp.GetCapabilities()[3].GetVolumeExpansion()
	if cap4volume == nil {
		t.Fatalf("unexpected capability at index 3: %v", resp.GetCapabilities()[3])
	}
	if cap4volume.GetType() != proto.PluginCapability_VolumeExpansion_OFFLINE {
		t.Errorf("unexpected volume expansion type: %s", cap4volume.GetType())
	}
}

func TestIdentityServiceProbeReady(t *testing.T) {
	env := newIdentityServerTestEnv()

	env.service.SetReady(true)

	resp, err := env.service.Probe(env.ctx, &proto.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.GetReady().GetValue() {
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
	if resp.GetReady().GetValue() {
		t.Error("expected to not be ready")
	}
}
