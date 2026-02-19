# Getting Started

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

   Have a look at our [Version Matrix](versioning-policy.md) to pick the correct version.

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

## Alternative Kubelet Directory

Some Kubernetes distributions use a non-standard path for the Kubelet directory.
The csi-driver needs to know about this to successfully mount volumes. You can
configure this through the Helm Chart Value `node.kubeletDir`.

- Standard: `/var/lib/kubelet`
- **k0s**: `/var/lib/k0s/kubelet`
- **microk8s**: `/var/snap/microk8s/common/var/lib/kubelet`

## Volumes Encrypted with LUKS

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

## Formatting Options

You can specify extra formatting options which are passed directly to `mkfs.FSTYPE` via the `fsFormatOptions` parameter in the storage class.

### Example

```yaml
parameters:
  csi.storage.k8s.io/fstype: xfs
  fsFormatOptions: "-i nrext64=1"
```

## XFS Filesystem

When using XFS as the filesystem type and no `fsFormatOptions` are set, we apply a default configuration to mkfs to ensure maximum compatibility with older Linux kernel versions. This configuration file is from the `xfsprogs-extra` alpine package and currently targets Linux 4.19.

> [!NOTE]
> The targeted minimum Linux Kernel version may be raised in a minor update, we will announce this in the Release Notes.

If you set any options at all, it is your responsible to make sure that all default flags from `mkfs.xfs` are supported on your current Linux Kernel version or that you set the flags appropriately.

## Volume Location

During the initialization of the CSI controller, the default location for all volumes is determined based on the following prioritized methods (evaluated in order from 1 to 4). However, when `volumeBindingMode: WaitForFirstConsumer` is used, the volume's location is determined by the node where the Pod is scheduled, and the default location is not applicable. For more details, refer to the official [Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode).

1. The location is explicitly set using the `HCLOUD_VOLUME_DEFAULT_LOCATION` variable.
2. The location is derived by querying a server specified by the `HCLOUD_SERVER_ID` variable.
3. If neither of the above is set, the `KUBE_NODE_NAME` environment variable defaults to the name of the node where the CSI controller is scheduled. This node name is then used to query the Hetzner API for a matching server and its location.
4. As a final fallback, the [Hetzner metadata service](https://docs.hetzner.cloud/reference/cloud#server-metadata) is queried to obtain the server ID, which is then used to fetch the location from the Hetzner API.

## Volume Labels

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
