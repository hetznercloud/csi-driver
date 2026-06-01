# Alternative Kubelet Directory

Some Kubernetes distributions use a non-standard path for the Kubelet directory.
The hcloud-csi-driver needs to know about this to successfully mount volumes. You can
configure this through the Helm Chart Value `node.kubeletDir`.

- Standard: `/var/lib/kubelet`
- **k0s**: `/var/lib/k0s/kubelet`
- **microk8s**: `/var/snap/microk8s/common/var/lib/kubelet`
