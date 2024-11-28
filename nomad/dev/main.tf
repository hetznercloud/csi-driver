module "dev" {
  source = "github.com/hetznercloud/nomad-dev-env?ref=nomad-dev-env"

  hcloud_token = var.hcloud_token
}