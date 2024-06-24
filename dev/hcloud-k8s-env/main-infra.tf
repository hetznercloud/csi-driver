# Setup the infrastructure

provider "hcloud" {
  token = var.hcloud_token
}

locals {
  labels = {
    env = var.name
  }
}

# SSH Key

resource "tls_private_key" "ssh" {
  algorithm = "ED25519"
}

resource "local_sensitive_file" "ssh" {
  content  = tls_private_key.ssh.private_key_openssh
  filename = abspath("${path.root}/files/id_ed25519")
}

resource "hcloud_ssh_key" "default" {
  name       = var.name
  public_key = tls_private_key.ssh.public_key_openssh
  labels     = local.labels
}

# Network

resource "hcloud_network" "cluster" {
  name     = var.name
  ip_range = "10.0.0.0/8"
  labels   = local.labels
}

resource "hcloud_network_subnet" "cluster" {
  network_id   = hcloud_network.cluster.id
  network_zone = "eu-central"
  type         = "cloud"
  ip_range     = "10.0.0.0/24"
}

# Control Plane Node

resource "hcloud_server" "control" {
  name        = "${var.name}-control"
  server_type = var.hcloud_server_type
  location    = var.hcloud_location
  image       = var.hcloud_image
  ssh_keys    = [hcloud_ssh_key.default.id]
  labels      = local.labels

  connection {
    host        = self.ipv4_address
    private_key = tls_private_key.ssh.private_key_openssh
  }

  provisioner "remote-exec" {
    inline = ["cloud-init status --wait || test $? -eq 2"]
  }
}

resource "hcloud_server_network" "control" {
  server_id = hcloud_server.control.id
  subnet_id = hcloud_network_subnet.cluster.id
}

# Worker / Agent Nodes

variable "worker_count" {
  type    = number
  default = 3
}

resource "hcloud_server" "worker" {
  count = var.worker_count

  name        = "${var.name}-worker-${count.index}"
  server_type = var.hcloud_server_type
  location    = var.hcloud_location
  image       = var.hcloud_image
  ssh_keys    = [hcloud_ssh_key.default.id]
  labels      = local.labels

  connection {
    host        = self.ipv4_address
    private_key = tls_private_key.ssh.private_key_openssh
  }

  provisioner "remote-exec" {
    inline = ["cloud-init status --wait || test $? -eq 2"]
  }
}

resource "hcloud_server_network" "worker" {
  count = var.worker_count

  server_id = hcloud_server.worker[count.index].id
  subnet_id = hcloud_network_subnet.cluster.id
}
