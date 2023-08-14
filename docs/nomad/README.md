# HashiCorp Nomad Hetzner Cloud csi-driver

## Preconditions

- Nomad >= 1.4.x cluster installed and running (tested on Nomad version 1.5.x).
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

5. Deploy the Jobs:

```sh
nomad job run hcloud-csi-controller.hcl
nomad job run hcloud-csi-node.hcl

# Check status
nomad plugin status
```

### Volumes Setup

1. Define a Volume

Create a file `db-vol.hcl` for the volume resource:

```
# file: vol.hcl

type      = "csi"
id        = "db-vol"
name      = "db-vol"
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

2. Create a Volume:

```sh
nomad volume create db-vol.hcl
```

> [!HINT]
>  The hcloud cli provides a convient way to verify if the volume was created: `hcloud volume list`.

### Make use of the Volume

1. Create a Job definition

The following example describes how to mount the volume in a Docker Nomad job definition:

```hcl
# file: mariadb.nomad

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
nomad job run mardiadb.nomad
```
