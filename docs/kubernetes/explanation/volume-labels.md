# Volume Labels

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

## Label Validation and Truncation

All volume labels are validated against the [Hetzner Cloud API requirements](https://docs.hetzner.cloud/reference/cloud#description/labels) before a volume is created. If any label does not pass validation, the volume creation will fail with an `InvalidArgument` error.

Label values that exceed the maximum length of 63 characters are automatically truncated from the left, keeping the last 63 characters. This is especially relevant for automatically set labels like `pvc-name`, `pvc-namespace`, and `pv-name`, which may contain long Kubernetes resource names.
