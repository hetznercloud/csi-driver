# HashiCorp Nomad Hetzner Cloud csi-driver

## Preconditions

- Nomad >= 1.4.x cluster installed and running (tested on Nomad version 1.5.x).
- The HCL resources are meant to be executed on a machine having nomad installed (with access to the Nomad API).

## Getting Started

1. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/).

2. Create a Nomad Variable for the HCLOUD token:

> [!NOTE]
> Consider using HashiCorp Vault for secrets management, see https://developer.hashicorp.com/nomad/docs/job-specification/template#vault-kv-api-v2

```sh
export HCLOUD_TOKEN="..."
nomad var put secrets/hcloud hcloud_token=$HCLOUD_TOKEN
```

3. Create a CSI Controller Job

```hcl
# hcloud-csi-controller.hcl

job "hcloud-csi-controller" {
  datacenters = ["dc1"]
  namespace   = "default"
  type        = "service"

  group "controller" {
    count = 2

    constraint {
      distinct_hosts = true
    }

    update {
      max_parallel     = 1
      canary           = 1
      min_healthy_time = "10s"
      healthy_deadline = "1m"
      auto_revert      = true
      auto_promote     = true
    }

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
        data        = <<EOH
HCLOUD_TOKEN="{{ with nomadVar "secrets/hcloud" }}{{ .hcloud_token }}{{ end }}"
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

4. Create a CSI Node Job

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
        data        = <<EOH
HCLOUD_TOKEN="{{ with nomadVar "secrets/hcloud" }}{{ .hcloud_token }}{{ end }}"
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

5. Deploy Jobs

```
nomad job run hcloud-csi-controller.hcl
nomad job run hcloud-csi-node.hcl

# Check status
nomad plugin status
```

6. Define a Volume

Create a file `vol.hcl` for the volume resource:

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

and run it:

```sh
nomad volume create vol.hcl
```
