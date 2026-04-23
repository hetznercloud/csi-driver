module "dev" {
  source = "v0.2.3" # renovate: datasource=github-releases depName=hetznercloud/nomad-dev-env

  hcloud_token = var.hcloud_token
}
