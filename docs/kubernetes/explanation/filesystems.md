# Filesystems

The csi-driver lets you choose the filesystem type used when formatting volumes, and pass extra options directly to the formatting command.

## Filesystem type

By default, volumes are formatted with `ext4`. You can select a different filesystem type by setting `csi.storage.k8s.io/fstype` in your storage class parameters.

## Formatting options

You can specify extra formatting options which are passed directly to `mkfs.FSTYPE` via the `fsFormatOptions` parameter in the storage class.

```yaml
parameters:
  csi.storage.k8s.io/fstype: xfs
  fsFormatOptions: "-i nrext64=1"
```

> [!WARNING]
> Formatting options are not validated. Passing invalid or unsupported options may cause volume formatting to fail, leaving the volume unusable.

## XFS compatibility defaults

When using `xfs` without any `fsFormatOptions`, the driver applies a default `mkfs` configuration to maximize compatibility with older Linux kernels. This configuration comes from the `xfsprogs-extra` Alpine package and currently targets Linux 4.19.

As soon as you set any `fsFormatOptions`, this default configuration no longer applies. You then become responsible for ensuring the resulting `mkfs.xfs` flags are supported by your current Linux kernel, either by relying on supported defaults or by setting flags appropriately.

> [!IMPORTANT]
> The targeted minimum Linux kernel version may be raised in a minor update. Such changes will be announced in the Release Notes.
