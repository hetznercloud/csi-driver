# How to import volumes

This guide explains how to import an existing Hetzner Volume into your Kubernetes cluster with the csi-driver installed.

1. Detach your volume by running:

```bash
hcloud volume detach <VOLUME-NAME>
```

2. Find the ID of your volume by running:

```bash
hcloud volume describe <VOLUME-NAME>
```

3. Create a new `PersistentVolume` and insert the volume ID into the `<VOLUME-ID>` and your volume location into `<VOLUME-LOCATION>`. This ensures that the topology constraints are set up correctly.

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
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: csi.hetzner.cloud/location
              operator: In
              values:
                - <VOLUME-LOCATION>
```

4. Create a new `PersistentVolumeClaim` and link it to the `PersistentVolume` via `volumeName`:

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

## Encrypted volumes

If your volume was previously encrypted, you need to provide a reference to the encryption secret in the persistent volume spec.

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
    nodePublishSecretRef: # <-- encryption secret reference
      name: <ENCRYPTION-SECRET-NAME>
      namespace: <ENCRYPTION-SECRET-NAMESPACE>
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: csi.hetzner.cloud/location
              operator: In
              values:
                - <VOLUME-LOCATION>
```
