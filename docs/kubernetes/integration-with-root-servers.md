# Integration with Root Servers

Root servers can be part of the cluster, but the CSI plugin doesn't work there. To ensure proper topology evaluation, labels are needed to indicate whether a node is a cloud VM or a dedicated server from Robot. If you are using the `hcloud-cloud-controller-manager` version 1.21.0 or later, these labels are added automatically. Otherwise, you will need to label the nodes manually.

## Adding labels manually

**Cloud Servers**

```bash
kubectl label nodes <node name> instance.hetzner.cloud/provided-by=cloud
```

**Root Servers**

```bash
kubectl label nodes <node name> instance.hetzner.cloud/provided-by=robot
```

## DEPRECATED: Old Label

We prefer that you use our [new label](#new-label). The label `instance.hetzner.cloud/is-robot-server` will be deprecated in future releases.

**Cloud Servers**

```bash
kubectl label nodes <node name> instance.hetzner.cloud/is-root-server=false
```

**Root Servers**

```bash
kubectl label nodes <node name> instance.hetzner.cloud/is-root-server=true
```

## Pods stuck in pending

The current behavior of the scheduler can cause Pods to be stuck in `Pending` when using the integration with Robot servers.

To address this behavior, you can set `enableProvidedByTopology` to `true` in the Helm Chart configuration. This setting prevents pods from being scheduled on nodes — specifically, Robot servers — where Hetzner volumes are unavailable. Enabling this option adds the `instance.hetzner.cloud/provided-by` label to the [allowed topologies](https://kubernetes.io/docs/concepts/storage/storage-classes/#allowed-topologies) section of the storage classes that are created. Additionally, this label is included in the `topologyKeys` section of `csinode` objects, and a node affinity is set up for each persistent volume. This workaround does not work with the [old label](#deprecated-old-label).

> [!WARNING]
> Once enabled, this feature cannot be easily disabled. It automatically adds required nodeAffinities to each volume and the topology keys to `csinode` objects. If the feature is later disabled, the topology keys are removed from the `csinode` objects, leaving volumes with required affinities that cannot be satisfied.

> [!NOTE]
> After enabling this feature, the workaround for the Kubernetes upstream issue only works on newly created volumes, as old volumes are not updated with the required node affinity.

```yaml
global:
  enableProvidedByTopology: true
```

Further information on the upstream issue can be found [here](https://github.com/kubernetes-csi/external-provisioner/issues/544).
