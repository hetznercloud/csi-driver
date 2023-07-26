# HashiCorp Nomad Hetzner Cloud csi-driver

## Preconditions

- Nomad >= 1.4.x cluster installed and running (tested on Nomad version 1.5.x).
- The HCL resources are meant to be executed on a machine having nomad installed (with access to the Nomad API).

## Getting Started

1. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/).

2. Deploy CSI Controller

```hcl
# hcloud-csi-controller.hcl

job "hcloud-csi-controller" {
  datacenters = ["dc1"]
  namespace   = "default"
  type        = "system"

  group "controller" {
    task "plugin" {
      driver = "docker"

      config {
        # Check version on https://hub.docker.com/r/hetznercloud/hcloud-csi-driver/tags
        image   = "hetznercloud/hcloud-csi-driver:v2.3.2"
        command = "bin/hcloud-csi-driver-controller"
      }

      env {
        CSI_ENDPOINT   = "unix://csi/csi.sock"
        ENABLE_METRICS = true
      }

      template {
        data = <<EOH
# WARNING: Consider using HashiCorp Vault for secrets management, see https://developer.hashicorp.com/nomad/docs/job-specification/template#vault-kv-api-v2
HCLOUD_TOKEN="<token goes here>"
EOH
        destination = "secrets/hcloud-token.env"
        env         = true
}

      csi_plugin {
        id        = "csi.hetzner.cloud"
        type      = "controller"
        mount_dir = "/csi"
      }

      resources {
        cpu    = 100
        memory = 64
      }
    }
  }
}
```

3. Deploy CSI Node

```hcl
# hcloud-csi-node.hcl
job "hcloud-csi-node" {
  datacenters = ["dc1"]
  namespace   = "default"
  type        = "system"

  group "node" {
    task "plugin" {
      driver = "docker"

      config {
        # Check version on https://hub.docker.com/r/hetznercloud/hcloud-csi-driver/tags
        image      = "hetznercloud/hcloud-csi-driver:v2.3.2"
        command    = "bin/hcloud-csi-driver-node"
        privileged = true
      }

      env {
        CSI_ENDPOINT   = "unix://csi/csi.sock"
        ENABLE_METRICS = true
      }

    template {
        data = <<EOH
# WARNING: Consider using HashiCorp Vault for secrets management, see https://developer.hashicorp.com/nomad/docs/job-specification/template#vault-kv-api-v2
HCLOUD_TOKEN="<token goes here>"
EOH
        destination = "secrets/hcloud-token.env"
        env         = true
      }

      csi_plugin {
        id        = "csi.hetzner.cloud"
        type      = "node"
        mount_dir = "/csi"
      }

      resources {
        cpu    = 100
        memory = 64
      }
    }
  }
}
```

4. Volume

Create a volume resource:

```
# vol.hcl

type      = "csi"
id        = "my-vol"
name      = "my-vol"
namespace = "default"
plugin_id = "csi.hetzner.cloud"

capability {
  access_mode     = "single-node-writer"
  attachment_mode = "file-system"
}

mount_options {
  fs_type     = "ext4"
  mount_flags = ["discard", "defaults"]
}
```
