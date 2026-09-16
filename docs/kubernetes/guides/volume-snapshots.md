# Volume snapshots

> This guide documents the proposed implementation. It requires the native
> block-volume snapshot API proposed in
> [hcloud-go#921](https://github.com/hetznercloud/hcloud-go/pull/921).

Install the Kubernetes `VolumeSnapshot` CRDs and the cluster-wide snapshot
controller before enabling snapshot resources. The hcloud-csi Helm chart runs
the CSI snapshotter sidecar, but it does not own those cluster-wide components.

Create a snapshot class:

```yaml
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshotClass
metadata:
  name: hcloud-volumes
driver: csi.hetzner.cloud
deletionPolicy: Delete
```

Create a crash-consistent snapshot of a bound claim:

```yaml
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: database-backup
spec:
  volumeSnapshotClassName: hcloud-volumes
  source:
    persistentVolumeClaimName: database
```

The source volume may remain attached. Applications that require a
transactionally consistent backup must quiesce writes before creating the
snapshot.

Restore a claim from the ready snapshot:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: database-restored
spec:
  storageClassName: hcloud-volumes
  dataSource:
    name: database-backup
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

The requested capacity must be at least the snapshot size. Restored volumes
remain in the snapshot's location.
