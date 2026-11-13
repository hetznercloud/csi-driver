<!--
---
date: "2026-09-07"
date_changed: "2026-11-13"
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

- Requires Linux 5.9+ on the nodes. Publishing the Volume fails on older kernels.
- The options are applied when the LUKS device is opened. Changing them only affects Volumes after they were unpublished and published again, e.g. by restarting the Pod.
- Encryption runs in the context of the process submitting the I/O instead of the kernel workqueue. CPU time is accounted to the Pod and may lead to throttling for Pods with tight CPU limits.
- Throughput for large sequential I/O may decrease, as the work is no longer distributed across CPUs.
- See [cloudflare/linux#1](https://github.com/cloudflare/linux/issues/1) for background and reported issues.
