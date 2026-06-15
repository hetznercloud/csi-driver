# Updating the Topology selection for `PersistentVolumes` created with csi-driver v2.0.0

This guide is intended for Kubernetes Cluster operators that installed the hcloud-csi-driver v2.0.0 and created Volumes. Unfortunately this version included a change that we had to revert, for details you can read the [issue #333](https://github.com/hetznercloud/csi-driver/issues/333).

The affected `PersistentVolumes` reference a wrong label in the `spec.nodeAffinity.required` fields. The volumes work as they should, but for consistency and future compatibility we recommend that the `PersistentVolumes` should be fixed.

Unfortunately the affected field is immutable, so we need to recreate the `PersistentVolume` in Kubernetes, the actual Volume with the data will not be touched, no data loss is expected.
This is only possible if it is not attached to a node, so any workload using the `PersistentVolume` needs to be paused for this.

## Pre-requisites

You need to have `kubectl` and the `hcloud` cli tool installed for this guide.

To avoid creating any new broken `PersistentVolumes` while you are still fixing old ones, you should upgrade to `v2.0.1` or `v2.1.0` before starting this guide. After all `PersistentVolumes` have been migrated, you should upgrade to `v2.1.0`

## Find affected `PersistentVolumes`

To find out if you are affected by this, and which `PersistentVolumes` need to be recreated, you can run the following command, which will output a list of affected `PersistentVolumes`.

```shell
kubectl get persistentvolume -o=custom-columns="NAME:.metadata.name,CLAIM:.spec.claimRef.name,TOPOLOGY:.spec.nodeAffinity.required.nodeSelectorTerms[*].matchExpressions[*].key,DRIVER:.metadata.annotations.pv\.kubernetes\.io/provisioned-by" | grep -e NAME -e "topology.kubernetes.io/region.*csi.hetzner.cloud" --color=never

NAME                                       CLAIM         TOPOLOGY                        DRIVER
pvc-0409d3b7-46c8-4a95-8475-dfbb053559c0   csi-test-6    topology.kubernetes.io/region   csi.hetzner.cloud
pvc-194d750a-bc28-4911-9618-8c6f3e61c404   csi-test-9    topology.kubernetes.io/region   csi.hetzner.cloud
pvc-2d1a5015-74c1-4746-b523-e1ce8d91705e   csi-test-5    topology.kubernetes.io/region   csi.hetzner.cloud
pvc-35cff6bc-dc4e-4f3d-9524-31d3217a77c4   csi-test-10   topology.kubernetes.io/region   csi.hetzner.cloud
pvc-44623554-8491-4ee0-a55b-92e9b6a5fa78   csi-test-8    topology.kubernetes.io/region   csi.hetzner.cloud
pvc-92b2e92d-d079-4df2-860d-b715996d9f86   csi-test-2    topology.kubernetes.io/region   csi.hetzner.cloud
pvc-c157fbd7-e26f-4ab6-8587-aa0ac737ee93   csi-test-4    topology.kubernetes.io/region   csi.hetzner.cloud
pvc-ca9a8389-596b-404a-85f8-56aa4362c00f   csi-test-3    topology.kubernetes.io/region   csi.hetzner.cloud
pvc-e7eccb3f-a842-452d-b10f-f8f88a40c267   csi-test-1    topology.kubernetes.io/region   csi.hetzner.cloud
pvc-eff7592a-616d-4188-9963-8c2640093d32   csi-test-7    topology.kubernetes.io/region   csi.hetzner.cloud
```

This (example) output means that 10 `PersistentVolumes` are affected. You should save this output somewhere, as you will need the names for the next steps.

## Re-create `PersistentVolume`

You need to repeat these steps for every `PersistentVolume`.

Start by pausing the application (if any) using the `PersistentVolume`. For example, if you have a Deployment that uses the `PersistentVolume` (through a `PersistentVolumeClaim`, see table above), you can scale it down to 0.

For the following step we provide a migration script, that will make a backup, and then delete and re-create the `PersistentVolume`. You can use this or execute the steps included in the script yourself.

Download the script:

```shell
$ curl https://raw.githubusercontent.com/hetznercloud/csi-driver/main/docs/v2.0.0-fix-volume-topology/fix-persistentvolume-topology.sh ./fix-persistentvolume-topology.sh
$ chmod +x ./fix-persistentvolume-topology.sh
```

Make sure that you have the right Kubernetes and hcloud contexts selected in the current shell.

Now you can run the script for a single `PersistentVolume`:

```shell
$ ./fix-persistent-volume.sh pvc-e7eccb3f-a842-452d-b10f-f8f88a40c267
[INFO] Creating a new directory to backup objects: ./hcloud-csi-fix-topology/pvc-e7eccb3f-a842-452d-b10f-f8f88a40c267
[INFO] Current state of Volume deletion protection: false
[INFO] Enabling Volume deletion protection
1.1s [===================================] 100.00%
Resource protection enabled for volume 123456789
[INFO] Removing finalizers from PersistentVolume
persistentvolume/pvc-e7eccb3f-a842-452d-b10f-f8f88a40c267 patched
[INFO] Deleting current PersistentVolume
persistentvolume "pvc-e7eccb3f-a842-452d-b10f-f8f88a40c267" deleted
persistentvolume/pvc-e7eccb3f-a842-452d-b10f-f8f88a40c267 patched
[INFO] Waiting for deletion to finish
[INFO] Creating new PersistentVolume
persistentvolume/pvc-e7eccb3f-a842-452d-b10f-f8f88a40c267 created
[INFO] Disabling Volume deletion protection which was added for migration
600ms [==================================] 100.00%
Resource protection disabled for volume 123456789
```

Once the script has successfully finished, you can scale up the workload again.

In case the script encountered an error and shows a message prefixed with `[ERR]` some pre-condition was not met:

- You are missing the `kubectl` or `hcloud` binaries
- The volume is not affected
- The volume does not belong to hcloud-csi-driver
- The volume is still attached to a server

You can fix these errors and then re-run the script.

In case something else goes wrong, the script makes backups of all resources in the directory`./hcloud-csi-fix-topology/$PERSITENT_VOLUME_NAME`, as logged by the script. You can use these to manually re-create the `PersistentVolume`.

If you have any issues, please feel free to open an issue on the [GitHub Repository](https://github.com/hetznercloud/csi-driver) or through the [Hetzner Ticket System](https://console.hetzner.cloud/support).
