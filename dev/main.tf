module "dev" {
  source = "github.com/hetznercloud/kubernetes-dev-env?ref=v0.5.0"

  name         = "csi-driver-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  hcloud_token = var.hcloud_token
  worker_count = 3

  k3s_channel = var.k3s_channel
}
