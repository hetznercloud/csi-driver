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
        image   = "$SKAFFOLD_IMAGE"
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