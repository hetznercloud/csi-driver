job "hcloud-csi-node" {
  datacenters = ["dc1"]
  namespace   = "default"
  type        = "system"

  group "node" {
    task "plugin" {
      driver = "docker"

      config {
        image      = "$SKAFFOLD_IMAGE"
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