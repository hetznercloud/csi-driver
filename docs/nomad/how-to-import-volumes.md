# How to import volumes

This guide explains how to import an existing Hetzner Volume into your Nomad cluster with the csi-driver installed.

1. Make sure your volume is detached by running:

```bash
hcloud volume detach <VOLUME-NAME>
```

2. Find the ID and location of your volume by running:

```bash
hcloud volume describe <VOLUME-NAME>
```

3. Define a new volume and provide values for `<VOLUME-NAME>`, `<VOLUME-ID>`, and `<VOLUME-LOCATION>`:

```yaml
# volume.hcl

type      = "csi"
id        = "<VOLUME-NAME>"
name      = "<VOLUME-NAME>"
namespace = "default"
plugin_id = "csi.hetzner.cloud"

# For 'nomad volume register', provide the external ID from the storage
# provider.
external_id = "<VOLUME-ID>"

capability {
  access_mode     = "single-node-writer"
  attachment_mode = "file-system"
}

mount_options {
  fs_type     = "ext4"
  mount_flags = ["discard", "defaults"]
}

topology_request {
  required {
    topology { segments { "csi.hetzner.cloud/location" = "<VOLUME-LOCATION>" } }
  }
}
```

4. Register the volume:

> [!NOTE]
> This might error with `Error registering volume: EOF`. Nonetheless, you can verify if the volume was created by running `nomad volume status`.

```bash
nomad register volume.hcl
nomad volume status
```
