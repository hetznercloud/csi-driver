module "dev" {
  source = "github.com/hetznercloud/kubernetes-dev-env?ref=v0.6.0"

  name         = "csi-driver-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  hcloud_token = var.hcloud_token
  worker_count = 3
  hcloud_location = "hel1"

  k3s_channel = var.k3s_channel
}
