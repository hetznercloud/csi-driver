module "dev" {
  source = "github.com/hetznercloud/terraform-k8s-dev?ref=v0.2.0"

  name         = "csi-driver-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  hcloud_token = var.hcloud_token

  k3s_channel = var.k3s_channel
}
