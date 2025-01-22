# How to import volumes

This guide explains how to import an existing Hetzner volume into your Kubernetes cluster with the csi-driver installed.

1. Detach your volume by running `hcloud volume detach <volume-name>`
2. Find the ID of your volume by running `hcloud volume list`
3. Create a new PersistentVolume and insert the volume ID into the `volumeHandle`

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: imported-data
spec:
  storageClassName: hcloud-volumes
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  csi:
    fsType: ext4
    driver: csi.hetzner.cloud
    volumeHandle: "<VOLUME-ID>"
```

4. Create a new PersistentVolumeClaim and link it to the PersistentVolume via `volumeName`

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: imported-data
spec:
  storageClassName: hcloud-volumes
  volumeName: imported-data # <-- reference PV name
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```
