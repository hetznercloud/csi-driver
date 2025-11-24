# Kubernetes Hetzner Cloud csi-driver

## Getting Started

1. Create a read+write API token in the [Hetzner Cloud Console](https://console.hetzner.cloud/).

2. Create a secret containing the token:

   ```
   # secret.yml
   apiVersion: v1
   kind: Secret
   metadata:
     name: hcloud
     namespace: kube-system
   stringData:
     token: YOURTOKEN
   ```

   and apply it:

   ```
   kubectl apply -f <secret.yml>
   ```

3. Deploy the CSI driver and wait until everything is up and running:

   Have a look at our [Version Matrix](README.md#versioning-policy) to pick the correct version.

   ```sh
   # Sync the Hetzner Cloud helm chart repository to your local computer.
   helm repo add hcloud https://charts.hetzner.cloud
   helm repo update hcloud

   # Install the latest version of the csi-driver chart.
   helm install hcloud-csi hcloud/hcloud-csi -n kube-system
   ```

   <details>
     <summary><b>Alternative</b>: Using a plain manifest</summary>

   ```
   kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.5.1/deploy/kubernetes/hcloud-csi.yml
   ```

   </details>

4. To verify everything is working, create a persistent volume claim and a pod
   which uses that volume:

   ```
   apiVersion: v1
   kind: PersistentVolumeClaim
   metadata:
     name: csi-pvc
   spec:
     accessModes:
     - ReadWriteOnce
     resources:
       requests:
         storage: 10Gi
     storageClassName: hcloud-volumes
   ---
   kind: Pod
   apiVersion: v1
   metadata:
     name: my-csi-app
   spec:
     containers:
       - name: my-frontend
         image: busybox
         volumeMounts:
         - mountPath: "/data"
           name: my-csi-volume
         command: [ "sleep", "1000000" ]
     volumes:
       - name: my-csi-volume
         persistentVolumeClaim:
           claimName: csi-pvc
   ```

   Once the pod is ready, exec a shell and check that your volume is mounted at `/data`.

   ```
   kubectl exec -it my-csi-app -- /bin/sh
   ```

### Alternative Kubelet Directory

Some Kubernetes distributions use a non-standard path for the Kubelet directory.
The csi-driver needs to know about this to successfully mount volumes. You can
configure this through the Helm Chart Value `node.kubeletDir`.

- Standard: `/var/lib/kubelet`
- **k0s**: `/var/lib/k0s/kubelet`
- **microk8s**: `/var/snap/microk8s/common/var/lib/kubelet`

### Volumes Encrypted with LUKS

To add encryption with LUKS you have to create a dedicate secret containing an encryption passphrase and duplicate the default `hcloud-volumes` storage class with added parameters referencing this secret:

```
apiVersion: v1
kind: Secret
metadata:
 name: encryption-secret
 namespace: kube-system
stringData:
 encryption-passphrase: foobar

---

apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
 name: hcloud-volumes-encrypted
provisioner: csi.hetzner.cloud
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
parameters:
 csi.storage.k8s.io/node-publish-secret-name: encryption-secret
 csi.storage.k8s.io/node-publish-secret-namespace: kube-system
```

Your nodes might need to have `cryptsetup` installed to mount the volumes with LUKS.

### Formatting Options

You can specify extra formatting options which are passed directly to `mkfs.FSTYPE` via the `fsFormatOptions` parameter in the storage class.

#### Example

```yaml
parameters:
  csi.storage.k8s.io/fstype: xfs
  fsFormatOptions: "-i nrext64=1"
```

### XFS Filesystem

When using XFS as the filesystem type and no `fsFormatOptions` are set, we apply a default configuration to mkfs to ensure maximum compatibility with older Linux kernel versions. This configuration file is from the `xfsprogs-extra` alpine package and currently targets Linux 4.19.

> [!NOTE]
> The targeted minimum Linux Kernel version may be raised in a minor update, we will announce this in the Release Notes.

If you set any options at all, it is your responsible to make sure that all default flags from `mkfs.xfs` are supported on your current Linux Kernel version or that you set the flags appropriately.

### Volume Location

During the initialization of the CSI controller, the default location for all volumes is determined based on the following prioritized methods (evaluated in order from 1 to 4). However, when `volumeBindingMode: WaitForFirstConsumer` is used, the volume's location is determined by the node where the Pod is scheduled, and the default location is not applicable. For more details, refer to the official [Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode).

1. The location is explicitly set using the `HCLOUD_VOLUME_DEFAULT_LOCATION` variable.
2. The location is derived by querying a server specified by the `HCLOUD_SERVER_ID` variable.
3. If neither of the above is set, the `KUBE_NODE_NAME` environment variable defaults to the name of the node where the CSI controller is scheduled. This node name is then used to query the Hetzner API for a matching server and its location.
4. As a final fallback, the [Hetzner metadata service](https://docs.hetzner.cloud/reference/cloud#server-metadata) is queried to obtain the server ID, which is then used to fetch the location from the Hetzner API.

### Volume Labels

It is possible to set labels for all newly created volumes. By default, all volumes are labeled as follows:

- `pvc-name`
- `pvc-namespace`
- `pv-name`
- `managed-by=csi-driver`

To add extra labels to all created volumes set `HCLOUD_VOLUME_EXTRA_LABELS` in the format `key=value,...`.
This is also configurable from the Helm chart by the value `controller.volumeExtraLabels`, e.g:

```yaml
controller:
  volumeExtraLabels:
    cluster: myCluster
    env: prod
```

It is also possible to set only labels on specific volumes created by a storage class. To do this, you need to set `labels` in the format `key=value,...` as `extraParameters` inside the storage class.

There is an example to set the `labels` for the storage class over the Helm chart values:

```yaml
storageClasses:
  - name: hcloud-volumes
    defaultStorageClass: true
    reclaimPolicy: Delete
    extraParameters:
      labels: cluster=myCluster,env=prod
```

## Upgrading

To upgrade the csi-driver version, you just need to apply the new manifests to your cluster.

In case of a new major version, there might be manual steps that you need to follow to upgrade the csi-driver. See the following section for a list of major updates and their required steps.

### From v1 to v2

There are three breaking changes between v1.6 and v2.0 that require user intervention. Please take care to follow these steps, as otherwise the update might fail.

**Before the rollout**:

1. The secret containing the API token was renamed from `hcloud-csi` to `hcloud`. This change was made so both the cloud-controller-manager and the csi-driver can use the same secret. Check that you have a secret `hcloud` in the namespace `kube-system`, and that the secret contains the API token, as described in the section [Getting Started](#getting-started):

   ```shell
   $ kubectl get secret -n kube-system hcloud
   ```

2. We added a new field to our `CSIDriver` resource to support [CSI volume fsGroup policy management](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#configure-volume-permission-and-ownership-change-policy-for-pods). This change requires a replacement of the `CSIDriver` object. You need to manually delete the old object:

   ```shell
   $ kubectl delete csidriver csi.hetzner.cloud
   ```

   The new `CSIDriver` will be installed when you apply the new manifests.

3. Stop the old pods to make sure that only everything is replaced in order and no incompatible pods are running side-by-side:

   ```shell
   $ kubectl delete statefulset -n kube-system hcloud-csi-controller
   $ kubectl delete daemonset -n kube-system hcloud-csi-node
   ```

4. We changed the way the device path of mounted volumes is communicated to the node service. This requires changes to the `VolumeAttachment` objects, where we need to add information to the `status.attachmentMetadata` field. Execute the linked script to automatically add the required information. This requires `kubectl` version `v1.24+`, even if your cluster is running v1.23.

   ```shell
   $ kubectl version
   $ curl -O https://raw.githubusercontent.com/hetznercloud/csi-driver/main/docs/v2-fix-volumeattachments/fix-volumeattachments.sh
   $ chmod +x ./fix-volumeattachments.sh
   $ ./fix-volumeattachments.sh
   ```

**Rollout the new manifest**:

```shell
$ kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.5.1/deploy/kubernetes/hcloud-csi.yml
```

**After the rollout**:

1. Delete the now unused secret `hcloud-csi` in the namespace `kube-system`:

   ```shell
   $ kubectl delete secret -n kube-system hcloud-csi
   ```

2. Remove old resources that have been replaced:

   ```shell
   $ kubectl delete clusterrolebinding hcloud-csi
   $ kubectl delete clusterrole hcloud-csi
   $ kubectl delete serviceaccount -n kube-system hcloud-csi
   ```

## Integration with Root Servers

Root servers can be part of the cluster, but the CSI plugin doesn't work there. To ensure proper topology evaluation, labels are needed to indicate whether a node is a cloud VM or a dedicated server from Robot. If you are using the `hcloud-cloud-controller-manager` version 1.21.0 or later, these labels are added automatically. Otherwise, you will need to label the nodes manually.

### Adding labels manually

**Cloud Servers**

```bash
kubectl label nodes <node name> instance.hetzner.cloud/provided-by=cloud
```

**Root Servers**

```bash
kubectl label nodes <node name> instance.hetzner.cloud/provided-by=robot
```

### DEPRECATED: Old Label

We prefer that you use our [new label](#new-label). The label `instance.hetzner.cloud/is-robot-server` will be deprecated in future releases.

**Cloud Servers**

```bash
kubectl label nodes <node name> instance.hetzner.cloud/is-root-server=false
```

**Root Servers**

```bash
kubectl label nodes <node name> instance.hetzner.cloud/is-root-server=true
```

### Pods stuck in pending

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

## Versioning policy

We aim to support the latest three versions of Kubernetes. When a Kubernetes
version is marked as _End Of Life_, we will stop support for it and remove the
version from our CI tests. This does not necessarily mean that the
csi-driver does not still work with this version. We will
not fix bugs related only to an unsupported version.

Current Kubernetes Releases: https://kubernetes.io/releases/

| Kubernetes | CSI Driver |                                                                                    Deployment File |
| ---------- | ---------: | -------------------------------------------------------------------------------------------------: |
| 1.34       |   v2.18.1+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.17.0/deploy/kubernetes/hcloud-csi.yml |
| 1.33       |   v2.18.1+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.17.0/deploy/kubernetes/hcloud-csi.yml |
| 1.32       |   v2.18.1+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.17.0/deploy/kubernetes/hcloud-csi.yml |
| 1.31       |    v2.18.1 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.17.0/deploy/kubernetes/hcloud-csi.yml |
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
