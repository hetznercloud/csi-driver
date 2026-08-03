module "infra" {
  source = "github.com/hetznercloud/kubernetes-dev-env//modules/infra?ref=v0.11.0"

  name         = "csi-driver-${replace(var.name, "/[^a-zA-Z0-9-_]/", "-")}"
  hcloud_token = var.hcloud_token
  worker_count = 3

  k3s_channel = var.k3s_channel

  # Share the generated files with the k8s state
  output_dir = abspath("${path.root}/../files")
}
