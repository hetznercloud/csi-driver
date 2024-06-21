module "dev" {
  source = "./hcloud-k8s-env"

  name         = "csi-driver-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  hcloud_token = var.hcloud_token

  k3s_channel = var.k3s_channel
}
