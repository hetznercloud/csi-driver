# Filesystem Formatting Options

You can specify extra formatting options which are passed directly to `mkfs.FSTYPE` via the `fsFormatOptions` parameter in the storage class.

## Example

```yaml
parameters:
  csi.storage.k8s.io/fstype: xfs
  fsFormatOptions: "-i nrext64=1"
```
