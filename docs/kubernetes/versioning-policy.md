# Versioning policy

We aim to support the latest three versions of Kubernetes. When a Kubernetes
version is marked as _End Of Life_, we will stop support for it and remove the
version from our CI tests. This does not necessarily mean that the
csi-driver does not still work with this version. We will
not fix bugs related only to an unsupported version.

Current Kubernetes Releases: https://kubernetes.io/releases/

| Kubernetes | CSI Driver |                                                                                    Deployment File |
| ---------- | ---------: | -------------------------------------------------------------------------------------------------: |
| 1.35       |   v2.18.3+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.18.3/deploy/kubernetes/hcloud-csi.yml |
| 1.34       |   v2.18.3+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.18.3/deploy/kubernetes/hcloud-csi.yml |
| 1.33       |   v2.18.3+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.18.3/deploy/kubernetes/hcloud-csi.yml |
| 1.32       |   v2.18.3+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.18.3/deploy/kubernetes/hcloud-csi.yml |
| 1.31       |    v2.18.1 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.18.1/deploy/kubernetes/hcloud-csi.yml |
| 1.30       |    v2.17.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.17.0/deploy/kubernetes/hcloud-csi.yml |
| 1.29       |    v2.13.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.13.0/deploy/kubernetes/hcloud-csi.yml |
| 1.28       |    v2.10.1 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.10.1/deploy/kubernetes/hcloud-csi.yml |
| 1.27       |     v2.9.0 |  https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.9.0/deploy/kubernetes/hcloud-csi.yml |
| 1.26       |     v2.7.1 |  https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.7.1/deploy/kubernetes/hcloud-csi.yml |
| 1.25       |     v2.6.0 |  https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.6.0/deploy/kubernetes/hcloud-csi.yml |
| 1.24       |     v2.4.0 |  https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.4.0/deploy/kubernetes/hcloud-csi.yml |
| 1.23       |     v2.2.0 |  https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.2.0/deploy/kubernetes/hcloud-csi.yml |
| 1.22       |     v1.6.0 |  https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |
| 1.21       |     v1.6.0 |  https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |
| 1.20       |     v1.6.0 |  https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |
