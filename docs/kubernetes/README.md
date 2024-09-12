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

Root servers can be part of the cluster, but the CSI plugin doesn't work there. Taint the root server as follows to skip that node for the DaemonSet.

```bash
kubectl label nodes <node name> instance.hetzner.cloud/is-root-server=true
```

## Versioning policy

We aim to support the latest three versions of Kubernetes. When a Kubernetes
version is marked as _End Of Life_, we will stop support for it and remove the
version from our CI tests. This does not necessarily mean that the
csi-driver does not still work with this version. We will
not fix bugs related only to an unsupported version.

Current Kubernetes Releases: https://kubernetes.io/releases/

| Kubernetes | CSI Driver |                                                                                   Deployment File |
|------------|-----------:|--------------------------------------------------------------------------------------------------:|
| 1.31       |     2.9.0+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.9.0/deploy/kubernetes/hcloud-csi.yml |
| 1.30       |     2.9.0+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.9.0/deploy/kubernetes/hcloud-csi.yml |
| 1.29       |     2.9.0+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.9.0/deploy/kubernetes/hcloud-csi.yml |
| 1.28       |     2.9.0+ | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.9.0/deploy/kubernetes/hcloud-csi.yml |
| 1.27       |      2.9.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.9.0/deploy/kubernetes/hcloud-csi.yml |
| 1.26       |      2.7.1 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.7.1/deploy/kubernetes/hcloud-csi.yml |
| 1.25       |      2.6.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.6.0/deploy/kubernetes/hcloud-csi.yml |
| 1.24       |      2.4.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.4.0/deploy/kubernetes/hcloud-csi.yml |
| 1.23       |      2.2.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v2.2.0/deploy/kubernetes/hcloud-csi.yml |
| 1.22       |      1.6.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |
| 1.21       |      1.6.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |
| 1.20       |      1.6.0 | https://raw.githubusercontent.com/hetznercloud/csi-driver/v1.6.0/deploy/kubernetes/hcloud-csi.yml |
