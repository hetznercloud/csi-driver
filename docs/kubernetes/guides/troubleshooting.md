# Troubleshooting

This guide helps you diagnose and fix the most common problems with the hcloud-csi-driver. Start with [Gathering diagnostics](#gathering-diagnostics) to collect the information you need, then jump to the section that matches your symptom.

The examples assume the driver is installed in the `kube-system` namespace, as described in the [Quickstart](quickstart.md). Adjust the namespace if you installed it elsewhere.

## Gathering diagnostics

Before changing anything, find out what the driver is actually doing.

### Check that the driver pods are running

The driver runs as two workloads (see the
[architecture explanation](../explanation/architecture.md)): a `controller`
Deployment and a `node` DaemonSet.

```bash
kubectl get pods -n kube-system -l app.kubernetes.io/name=hcloud-csi
```

You should see one controller pod and one node pod **per schedulable node**. If
a node has no `node` pod, volumes cannot be mounted on that node.

### Read the driver logs

The main container in both workloads is called `hcloud-csi-driver`. This is
where the most useful messages appear, which are related to the codebase managed
in this repository.

```bash
# Controller (provisioning, attaching, resizing, Hetzner Cloud API calls)
kubectl logs -n kube-system -l app.kubernetes.io/name=hcloud-csi,app.kubernetes.io/component=controller \
  -c hcloud-csi-driver

# Node (formatting and mounting on a specific node)
kubectl logs -n kube-system -l app.kubernetes.io/name=hcloud-csi,app.kubernetes.io/component=node \
  -c hcloud-csi-driver --prefix
```

To watch a specific node's logs, find the pod scheduled on that node first:

```bash
kubectl get pods -n kube-system -l app.kubernetes.io/component=node -o wide
```

### Enable debug logs

By default the driver only logs at `info` level. When the standard logs are not
enough, raise the verbosity through two environment variables:

- `LOG_LEVEL=debug` increases the driver's log verbosity.
- `HCLOUD_DEBUG=true` logs the raw Hetzner Cloud API requests and responses. API tokens are redacted.

Set them on **both** components via the Helm values `controller.extraEnvVars`
and `node.extraEnvVars`:

```yaml
# values.yaml
controller:
  extraEnvVars:
    - name: LOG_LEVEL
      value: debug
    - name: HCLOUD_DEBUG
      value: "true"
node:
  extraEnvVars:
    - name: LOG_LEVEL
      value: debug
    - name: HCLOUD_DEBUG
      value: "true"
```

Apply the values and let the pods roll out:

```bash
helm upgrade hcloud-csi hcloud/hcloud-csi -n kube-system -f values.yaml
```

Once the new pods have started, you should see debug messages and raw API
requests in the logs.

If you deploy via raw manifests instead of Helm, add the same two environment
variables to the `hcloud-csi-driver` container of both workloads. Apply this
patch to `deploy/kubernetes/hcloud-csi.yml` with `git apply` and re-apply the
manifest:

```diff
--- a/deploy/kubernetes/hcloud-csi.yml
+++ b/deploy/kubernetes/hcloud-csi.yml
@@ -230,6 +230,10 @@
           securityContext:
             privileged: true
           env:
+            - name: LOG_LEVEL
+              value: debug
+            - name: HCLOUD_DEBUG
+              value: "true"
             - name: CSI_ENDPOINT
               value: unix:///run/csi/socket
             - name: METRICS_ENDPOINT
@@ -320,6 +324,10 @@
           args:
             - -controller
           env:
+            - name: LOG_LEVEL
+              value: debug
+            - name: HCLOUD_DEBUG
+              value: "true"
             - name: CSI_ENDPOINT
               value: unix:///run/csi/socket
             - name: METRICS_ENDPOINT
```

### Inspect the affected object

Kubernetes records most provisioning and mounting problems as Events on the PersistentVolumeClaim or the Pod:

```bash
kubectl describe pvc <PVC-NAME>
kubectl describe pod <POD-NAME>
```

Look at the `Events` section at the bottom of the output.

## PersistentVolumeClaim is stuck in `Pending`

A PVC that never leaves `Pending` means the **controller** could not provision a volume. Check the PVC events first (`kubectl describe pvc <PVC-NAME>`) and the controller logs.

### Missing or invalid API token

The controller needs a Hetzner Cloud API token to create volumes. By default it reads a secret named `hcloud` with a key `token` (see the [Quickstart](quickstart.md)). The token must have **read+write** permissions.

### Wrong or missing StorageClass

The PVC must reference a StorageClass whose `provisioner` is `csi.hetzner.cloud`.

```bash
kubectl get storageclass
```

If your PVC has no `storageClassName` and there is no default StorageClass, it will stay `Pending`. Either set `storageClassName: hcloud-volumes` on the PVC or mark a StorageClass as default.

### Volume limit or quota reached

If your project has reached its volume limit or quota, the Hetzner Cloud API rejects new volumes and the controller logs the API error. Check your limits in the [Hetzner Cloud Console](https://console.hetzner.cloud/) and delete unused volumes or request a limit increase via the [Support Center](https://console.hetzner.cloud/support).

## Pod is stuck in `ContainerCreating`

If the PVC is `Bound` but the Pod never starts, the volume could not be **attached or mounted** on the node. Check the Pod events (`kubectl describe pod <POD-NAME>`), the `controller` logs, and the `node` pod logs on that node.

### Volume and node are in different locations

Hetzner Cloud volumes are bound to a single location (e.g. `fsn1`, `nbg1`, `hel1`) and can only be attached to a server in the **same** location. If a Pod is scheduled onto a node in a different location than its volume, the attach fails.

The driver sets topology constraints to prevent this, but it can still happen when importing volumes or pinning Pods to specific nodes. Make sure `volumeBindingMode: WaitForFirstConsumer` is set on your StorageClass so the volume is created in the location where the Pod is scheduled.

See [Volume location](../explanation/volume-location.md) for more details.

### Non-standard kubelet directory

Some Kubernetes distributions (k0s, MicroK8s, …) use a non-standard kubelet directory. If the driver is not configured for it, mounts silently fail with errors about missing paths.

Set the Helm value `node.kubeletDir` to match your distribution:

- Standard: `/var/lib/kubelet`
- **k0s**: `/var/lib/k0s/kubelet`
- **microk8s**: `/var/snap/microk8s/common/var/lib/kubelet`

### No `node` pod on the target node

If the DaemonSet has no pod on the node where your Pod is scheduled, no mount can happen. This is usually caused by taints the DaemonSet does not tolerate, or by the node still starting up.

```bash
kubectl get pods -n kube-system -l app.kubernetes.io/component=node -o wide
```

Ensure the node is `Ready` and that the DaemonSet tolerates any custom taints on it.

## Volume resize has no effect

To grow a volume:

1. The StorageClass must have `allowVolumeExpansion: true`.
2. Increase `spec.resources.requests.storage` on the **PVC** (not the PV).

The controller resizes the underlying Hetzner Cloud volume, and the filesystem is expanded the next time it is mounted. Watch the controller and node logs if the new size does not appear. Note that volumes can only **grow**, never shrink.

## Volumes do not work on Robot / dedicated servers

The hcloud-csi-driver provisions **Hetzner Cloud** volumes, which are only available on Hetzner Cloud servers. Dedicated Robot servers cannot attach Cloud volumes.

If you run a mixed cluster, see [Integration with Robot servers](../explanation/integration-with-robot-servers.md) for what is and isn't supported.
