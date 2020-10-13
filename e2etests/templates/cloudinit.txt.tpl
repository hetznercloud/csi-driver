#cloud-config
write_files:
- content: |
    net.bridge.bridge-nf-call-ip6tables = 1
    net.bridge.bridge-nf-call-iptables = 1
  path: /etc/sysctl.d/k8s.conf
- content: |
    apiVersion: kubeadm.k8s.io/v1beta2
    kind: ClusterConfiguration
    kubernetesVersion: v{{.K8sVersion}}
    networking:
      podSubnet: "10.244.0.0/16"
  path: /tmp/kubeadm-config.yaml
- content: |
    [Service]
    Environment="KUBELET_EXTRA_ARGS=--cloud-provider=external"
  path: /etc/systemd/system/kubelet.service.d/20-hcloud.conf

runcmd:
- sysctl --system
- apt install -y apt-transport-https curl
- curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
- echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list
- apt update
- apt install -y kubectl={{.K8sVersion}}-00 kubeadm={{.K8sVersion}}-00 kubelet={{.K8sVersion}}-00 docker.io
- systemctl daemon-reload
- systemctl restart kubelet
- kubeadm init  --config /tmp/kubeadm-config.yaml
- mkdir -p /root/.kube
- cp -i /etc/kubernetes/admin.conf /root/.kube/config
- until KUBECONFIG=/root/.kube/config kubectl get node; do sleep 2;done
- KUBECONFIG=/root/.kube/config kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
- KUBECONFIG=/root/.kube/config kubectl -n kube-system patch ds kube-flannel-ds --type json -p '[{"op":"add","path":"/spec/template/spec/tolerations/-","value":{"key":"node.cloudprovider.kubernetes.io/uninitialized","value":"true","effect":"NoSchedule"}}]'
- KUBECONFIG=/root/.kube/config kubectl -n kube-system create secret generic hcloud-csi --from-literal=token={{.HcloudToken}}
- KUBECONFIG=/root/.kube/config kubectl -n kube-system create secret generic hcloud --from-literal=token={{.HcloudToken}}
- KUBECONFIG=/root/.kube/config kubectl apply -f  https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/ccm.yaml
- cd /root/ && curl --location https://dl.k8s.io/v{{.K8sVersion}}/kubernetes-test-linux-amd64.tar.gz | tar --strip-components=3 -zxf - kubernetes/test/bin/e2e.test kubernetes/test/bin/ginkgo
- KUBECONFIG=/root/.kube/config kubectl taint nodes --all node-role.kubernetes.io/master-

