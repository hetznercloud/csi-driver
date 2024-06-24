# Setup the k3s cluster

locals {
  # The CIDR range for the Pods, must be included in the range of the
  # network (10.0.0.0/8) but must not overlap with the Subnet (10.0.0.0/24)
  cluster_cidr = "10.244.0.0/16"

  registry_service_ip = "10.43.0.2"
  registry_port       = 30666

  kubeconfig_path = abspath("${path.root}/files/kubeconfig.yaml")
  env_path        = abspath("${path.root}/files/env.sh")
}

resource "null_resource" "k3sup_control" {
  triggers = {
    id = hcloud_server.control.id
    ip = hcloud_server_network.control.ip
  }

  connection {
    host        = hcloud_server.control.ipv4_address
    private_key = tls_private_key.ssh.private_key_openssh
  }

  provisioner "remote-exec" {
    inline = ["mkdir -p /etc/rancher/k3s"]
  }
  provisioner "file" {
    content = yamlencode({
      "mirrors" : {
        "localhost:${local.registry_port}" : {
          "endpoint" : ["http://${local.registry_service_ip}:5000"]
        }
      }
    })
    destination = "/etc/rancher/k3s/registries.yaml"
  }

  provisioner "local-exec" {
    command = <<-EOT
      k3sup install --print-config=false \
        --ssh-key='${local_sensitive_file.ssh.filename}' \
        --ip='${hcloud_server.control.ipv4_address}' \
        --k3s-channel='${var.k3s_channel}' \
        --k3s-extra-args="\
          --kubelet-arg=cloud-provider=external \
          --cluster-cidr='${local.cluster_cidr}' \
          --disable-cloud-controller \
          --disable-network-policy \
          --disable=local-storage \
          --disable=servicelb \
          --disable=traefik \
          --flannel-backend=none \
          --node-external-ip='${hcloud_server.control.ipv4_address}' \
          --node-ip='${hcloud_server_network.control.ip}'" \
        --local-path='${local.kubeconfig_path}'
    EOT
  }
}

resource "null_resource" "k3sup_worker" {
  count = var.worker_count

  triggers = {
    id = hcloud_server.worker[count.index].id
    ip = hcloud_server_network.worker[count.index].ip

    # Wait the control-plane to be initialized, and re-join the new cluster if the
    # control-plane server changed.
    control_id = null_resource.k3sup_control.id
  }

  connection {
    host        = hcloud_server.worker[count.index].ipv4_address
    private_key = tls_private_key.ssh.private_key_openssh
  }

  provisioner "remote-exec" {
    inline = ["mkdir -p /etc/rancher/k3s"]
  }
  provisioner "file" {
    content = yamlencode({
      "mirrors" : {
        "localhost:${local.registry_port}" : {
          "endpoint" : ["http://${local.registry_service_ip}:5000"]
        }
      }
    })
    destination = "/etc/rancher/k3s/registries.yaml"
  }

  provisioner "local-exec" {
    command = <<-EOT
      k3sup join \
        --ssh-key='${local_sensitive_file.ssh.filename}' \
        --ip='${hcloud_server.worker[count.index].ipv4_address}' \
        --server-ip='${hcloud_server.control.ipv4_address}' \
        --k3s-channel='${var.k3s_channel}' \
        --k3s-extra-args="\
          --kubelet-arg='cloud-provider=external' \
          --node-external-ip='${hcloud_server.worker[count.index].ipv4_address}' \
          --node-ip='${hcloud_server_network.worker[count.index].ip}'"
      EOT
  }
}

# Configure kubernetes

data "local_sensitive_file" "kubeconfig" {
  depends_on = [null_resource.k3sup_control]
  filename   = local.kubeconfig_path
}

provider "kubernetes" {
  config_path = data.local_sensitive_file.kubeconfig.filename
}

resource "kubernetes_secret_v1" "hcloud_token" {
  metadata {
    name      = "hcloud"
    namespace = "kube-system"
  }

  data = {
    token   = var.hcloud_token
    network = hcloud_network.cluster.id
  }
}

provider "helm" {
  kubernetes {
    config_path = data.local_sensitive_file.kubeconfig.filename
  }
}

resource "helm_release" "cilium" {
  name       = "cilium"
  chart      = "cilium"
  repository = "https://helm.cilium.io"
  namespace  = "kube-system"
  version    = "1.13.1"
  wait       = true

  set {
    name  = "operator.replicas"
    value = "1"
  }
  set {
    name  = "ipam.mode"
    value = "kubernetes"
  }
  set {
    name  = "tunnel"
    value = "disabled"
  }
  set {
    name  = "ipv4NativeRoutingCIDR"
    value = local.cluster_cidr
  }
}

resource "helm_release" "hcloud_cloud_controller_manager" {
  name       = "hcloud-cloud-controller-manager"
  chart      = "hcloud-cloud-controller-manager"
  repository = "https://charts.hetzner.cloud"
  namespace  = "kube-system"
  version    = "1.19.0"
  wait       = true

  set {
    name  = "networking.enabled"
    value = "true"
  }
}

resource "helm_release" "docker_registry" {
  name       = "docker-registry"
  chart      = "docker-registry"
  repository = "https://helm.twun.io"
  namespace  = "kube-system"
  version    = "2.2.3"
  wait       = true

  set {
    name  = "service.clusterIP"
    value = local.registry_service_ip
  }
  set {
    name  = "tolerations[0].key"
    value = "node.cloudprovider.kubernetes.io/uninitialized"
  }
  set {
    name  = "tolerations[0].operator"
    value = "Exists"
  }
}

# Export files

resource "local_file" "registry_port_forward" {
  source          = "${path.module}/registry-port-forward.sh"
  filename        = "${path.root}/files/registry-port-forward.sh"
  file_permission = "0755"
}

resource "local_file" "env" {
  content         = <<-EOT
    #!/usr/bin/env bash

    export KUBECONFIG=${data.local_sensitive_file.kubeconfig.filename}
    export SKAFFOLD_DEFAULT_REPO=localhost:${local.registry_port}
  EOT
  filename        = local.env_path
  file_permission = "0644"
}
