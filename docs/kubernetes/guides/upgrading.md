# Upgrading

For most upgrades — patch and minor versions within the same major version
(e.g. `v2.x` to `v2.y`) — no manual intervention is required. Upgrade by
re-running `helm upgrade` with the new chart version:

```bash
helm repo update
helm upgrade hcloud-csi hcloud/hcloud-csi -n kube-system
```

If you deploy via the static manifests instead of Helm, apply the manifest for
the target version:

```bash
kubectl apply -f https://raw.githubusercontent.com/hetznercloud/csi-driver/<VERSION>/deploy/kubernetes/hcloud-csi.yml
```

Before upgrading, check the [release notes](https://github.com/hetznercloud/csi-driver/releases)
for the versions you are skipping over, in case a specific release calls for
additional steps.

## Major version upgrades

Major version upgrades can contain breaking changes that require manual steps.
See the dedicated guide for your upgrade path:

- [Upgrading from v1 to v2](upgrading-from-v1-to-v2/)
