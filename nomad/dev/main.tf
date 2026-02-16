module "dev" {
  source = "github.com/hetznercloud/nomad-dev-env?ref=v0.2.2" # renovate: datasource=github-releases depName=hetznercloud/nomad-dev-env

  hcloud_token = var.hcloud_token
}
