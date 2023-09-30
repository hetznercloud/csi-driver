# csi-driver Helm Chart

This Helm chart is the recommended installation method for [hcloud-csi-driver](https://github.com/hetznercloud/csi-driver) on Kubernetes.

## Quickstart

First, [install Helm 3](https://helm.sh/docs/intro/install/).

The following snippet will deploy csi-driver to the kube-system namespace.

```sh
# Sync the Hetzner Cloud helm chart repository to your local computer.
helm repo add hcloud https://charts.hetzner.cloud
helm repo update hcloud

# Install the latest version of the csi-driver chart.
helm install hcloud-csi hcloud/hcloud-csi -n kube-system
```

Please note that a secret containing the Hetzner Cloud token is necessary. See the main [Kubernetes Deployment](../docs/kubernetes/README.md) guide.

If you're unfamiliar with Helm it would behoove you to peep around the documentation. Perhaps start with the [Quickstart Guide](https://helm.sh/docs/intro/quickstart/)?

### Upgrading from static manifests

If you previously installed csi-driver with this command:

```sh
kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.5.1/deploy/kubernetes/hcloud-csi.yml
```

You can uninstall that same deployment, by running the following command:

```sh
kubectl delete -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.5.1/deploy/kubernetes/hcloud-csi.yml
```

Then you can follow the Quickstart installation steps above.

## Configuration

This chart aims to be highly flexible. Please review the [values.yaml](./values.yaml) for a full list of configuration options.
There are additional recommendations for production deployments in [`example-prod.values.yaml`](./example-prod.values.yaml).


If you've already deployed csi-driver using the `helm install` command above, you can easily change configuration values:

```sh
helm upgrade hcloud-csi hcloud/csi-driver -n kube-system --set metrics.serviceMonitor.enabled=true
```

### Multiple replicas

If you want to use multiple replicas for the controller you can change `controller.replicaCount` inside the helm values.

If you have more than 1 replica leader election will be turned on automatically.
