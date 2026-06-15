# XFS Filesystem

When using XFS as the filesystem type and no `fsFormatOptions` are set, we apply a default configuration to mkfs to ensure maximum compatibility with older Linux kernel versions. This configuration file is from the `xfsprogs-extra` alpine package and currently targets Linux 4.19.

> [!NOTE]
> The targeted minimum Linux Kernel version may be raised in a minor update, we will announce this in the Release Notes.

If you set any options at all, it is your responsible to make sure that all default flags from `mkfs.xfs` are supported on your current Linux Kernel version or that you set the flags appropriately.
