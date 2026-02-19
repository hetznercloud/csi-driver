# Upgrading

To upgrade the csi-driver version, you just need to apply the new manifests to your cluster.

In case of a new major version, there might be manual steps that you need to follow to upgrade the csi-driver. See the following section for a list of major updates and their required steps.

## From v1 to v2

There are three breaking changes between v1.6 and v2.0 that require user intervention. Please take care to follow these steps, as otherwise the update might fail.

**Before the rollout**:

1. The secret containing the API token was renamed from `hcloud-csi` to `hcloud`. This change was made so both the cloud-controller-manager and the csi-driver can use the same secret. Check that you have a secret `hcloud` in the namespace `kube-system`, and that the secret contains the API token, as described in the section [Getting Started](getting-started.md):

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
