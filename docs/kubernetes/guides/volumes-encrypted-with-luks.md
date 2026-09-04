<!--
---
date: "2026-09-07"
date_changed: "2026-09-07"
title: "Encrypted with LUKS"
tags: []
language: "en"
description: ""
docs_type: ["how_to"]
product_category: ["Integrations"]
translation: ["Integrations", "CSI driver", "How-To: Volumes", "Encrypted with LUKS"]
scrape_type: "whole"
priority: 90
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
