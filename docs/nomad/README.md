# HashiCorp Nomad Hetzner Cloud csi-driver

## Preconditions

- Nomad >= 1.4.x cluster installed following the [Nomad Reference Architecture for production deployments](https://developer.hashicorp.com/nomad/tutorials/enterprise/production-reference-architecture-vm-with-consul). The setup was tested on Nomad Community, version 1.5.x.
- The cluster nodes need to have the `docker` driver installed & configured with [`allow_privileged = true`](https://developer.hashicorp.com/nomad/docs/drivers/docker#allow_privileged).
- The HCL resources are meant to be executed on a machine having nomad installed (with access to the Nomad API).

## Getting Started

### CSI Setup

1. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/).

2. Create a Nomad Variable for the HCLOUD token:

> [!NOTE]
> Consider using HashiCorp Vault for secrets management, see https://developer.hashicorp.com/nomad/docs/job-specification/template#vault-kv-api-v2

```sh
export HCLOUD_TOKEN="..."
nomad var put secrets/hcloud hcloud_token=$HCLOUD_TOKEN
```

3. Create a CSI Controller Job:

```hcl
# file: hcloud-csi-controller.hcl

job "hcloud-csi-controller" {
  datacenters = ["dc1"]
  namespace   = "default"
  type        = "service"

  group "controller" {

    ### NOTE
    # We define (at least) 2 allocations to increase the availability in case of a node failure with
    # a controller allocation running on that node. On a "Single Node Cluster", the group stanzas
    # might need modification or should be removed.
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
        # Get the latest version on https://hub.docker.com/r/hetznercloud/hcloud-csi-driver/tags
        image   = "hetznercloud/hcloud-csi-driver:v2.5.1"
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
        destination = "${NOMAD_SECRETS_DIR}/hcloud-token.env"
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

4. Create a CSI Node Job:

```hcl
# file: hcloud-csi-node.hcl
job "hcloud-csi-node" {
  datacenters = ["dc1"]
  namespace   = "default"
  type        = "system"

  group "node" {
    task "plugin" {
      driver = "docker"

      config {
        # Get the latest version on https://hub.docker.com/r/hetznercloud/hcloud-csi-driver/tags
        image      = "hetznercloud/hcloud-csi-driver:v2.5.1"
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
        destination = "${NOMAD_SECRETS_DIR}/hcloud-token.env"
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

5. Deploy the Jobs:

The following commands deploy the job resources created previously on your Nomad cluster:

```sh
nomad job run hcloud-csi-controller.hcl
nomad job run hcloud-csi-node.hcl
```

6. Verify the status:

To ensure the plugin is running and healthy, check the web UI on path `/ui/csi/plugins/csi.hetzner.cloud` or by using the CLI:

```sh
nomad plugin status
```

### Volumes Setup

1. Define a Volume:

Create a file `db-vol.hcl` for the volume resource:

> [!NOTE]
> See [Nomad Volume Specification](https://developer.hashicorp.com/nomad/docs/other-specifications/volume) for more information.

```hcl
# file: db-vol.hcl

type      = "csi"
id        = "db-vol"
name      = "db-vol"
namespace = "default"
plugin_id = "csi.hetzner.cloud"

# Default minimum capacity for Hetzner Cloud is 10G
capacity_min = "10G"

capability {
  access_mode     = "single-node-writer"
  attachment_mode = "file-system"
}

mount_options {
  fs_type     = "ext4"
  mount_flags = ["discard", "defaults"]
}
```
> [!IMPORTANT]
> The volume will be created in the same Hetzner Cloud Location as the controller is deployed into.

To define the Hetzner Cloud Location (CLI: `hcloud location list`) you would like to create the volume into, append the following snippet into the volume resource definition:

```hcl
topology_request {
  required {
   # Use your desired location name here
    topology { segments { "csi.hetzner.cloud/location" = "fsn1" } }
  }
}
```

2. Create a Volume:

```sh
nomad volume create db-vol.hcl
```

> [!NOTE]
>  The hcloud cli provides a convenient way to verify if the volume was created: `hcloud volume list`.

### Make use of the Volume

1. Create a Job definition:

The following example describes how to mount the volume in a Docker Nomad job definition (especially see the parts commented with `### THIS!`):

```hcl
# file: mariadb.hcl

job "mariadb" {
  datacenters = ["dc1"]
  namespace   = "default"
  type        = "service"

  group "mariadb" {
    network {
      port "mariadb" {
        to           = 3306
      }
    }

    ### THIS!
    volume "db-volume" {
      type            = "csi"
      read_only       = false
      source          = "db-vol"
      attachment_mode = "file-system"
      access_mode     = "single-node-writer"
      per_alloc       = false
    }

    task "mariadb" {
      driver = "docker"
      config {
        image = "mariadb:10.11"
        ports = [
          "mariadb",
        ]
      }

      ### THIS!
      volume_mount {
        volume      = "db-volume"
        destination = "/var/lib/mysql"
      }

      env {
        MARIADB_ROOT_PASSWORD = "<...>"
        MARIADB_DATABASE      = "<...>"
        MARIADB_USER          = "<...>"
        MARIADB_PASSWORD      = "<...>"
      }

      service {
        name = "db"
        # Uses nomad native service discovery. To use consul once configured, set it to "consul".
        # Also see https://developer.hashicorp.com/nomad/docs/job-specification/service#service-block
        provider = "nomad"
        port = "mariadb"
      }

      resources {
        cpu    = 300
        memory = 256
      }

    }
  }
}
```

2. Create the Job:

```sh
nomad job run mariadb.hcl
```

### Volumes encryption with LUKS

To add encryption with LUKS you have to provide a secret containing the encryption passphrase as part of the volume definition. The secret must be named `encryption-passphrase`. The volume will then be LUKS encrypted on first use.

```hcl
# file: db-vol.hcl

secrets {
  "encryption-passphrase" = "<your_encryption_value>"
}
```


> [!NOTE]
> Consider using HashiCorp Vault for secrets management, see https://developer.hashicorp.com/nomad/docs/job-specification/template#vault-kv-api-v2