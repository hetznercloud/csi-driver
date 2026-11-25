<!--
---
date: "2026-09-07"
date_changed: "2026-11-16"
title: "Encrypting Volumes with LUKS"
tags: []
language: "en"
description: ""
docs_type: ["how_to"]
product_category: ["Integrations"]
translation: ["Integrations", "Kubernetes CSI driver", "Guides", "Encrypting Volumes with LUKS"]
scrape_type: "whole"
priority: 80
---
-->

# Volumes Encrypted with LUKS

To add encryption with LUKS you have to create a dedicate secret containing an encryption passphrase and duplicate the default `hcloud-volumes` storage class with added parameters referencing this secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: encryption-secret
  namespace: kube-system
stringData:
  encryption-passphrase: foobar

---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: hcloud-volumes-encrypted
provisioner: csi.hetzner.cloud
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
parameters:
  csi.storage.k8s.io/node-publish-secret-name: encryption-secret
  csi.storage.k8s.io/node-publish-secret-namespace: kube-system
```

Your nodes might need to have `cryptsetup` installed to mount the Volumes with LUKS.

## Disabling dm-crypt workqueues

By default, dm-crypt queues encryption and decryption to kernel workqueues. Bypassing them can significantly reduce I/O latency, especially for small synchronous writes. Both workqueues can be disabled independently by adding the following keys to the encryption secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: encryption-secret
  namespace: kube-system
stringData:
  encryption-passphrase: foobar
  encryption-disable-read-workqueue: "true"
  encryption-disable-write-workqueue: "true"
```

Please consider the following before enabling these options:

- With `encryption-disable-write-workqueue` enabled, writes are encrypted in the context of the process submitting the I/O instead of in kernel threads. The CPU time is therefore accounted to the Pod's cgroup and counts towards its CPU limit.
- Requires Linux 5.9+ on the nodes. Publishing the Volume fails on older kernels.
- The options are applied when the LUKS device is opened. Changing them only affects Volumes after they were unpublished and published again, e.g. by restarting the Pod.
