<!--
---
date: "2026-09-07"
date_changed: "2026-09-07"
title: "Installing the CSI driver"
tags: []
language: "en"
description: ""
docs_type: ["getting_started"]
product_category: ["Integrations"]
translation: ["Integrations", "CSI driver", "Getting Started", "Installing the CSI driver"]
scrape_type: "whole"
priority: 90
---
-->

# Quick start

1. Create a read+write API token in the [Hetzner Console](https://console.hetzner.com/) as described in [this document](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/).

2. Create a secret containing your Hetzner Console API token:
   
   ```bash
   kubectl -n kube-system create secret generic hcloud --from-literal=token=<hcloud API token>
   ```

3. Add the Helm repository:
   
   ```bash
   helm repo add hcloud https://charts.hetzner.cloud
   helm repo update hcloud
   ```

4. Install the chart:
   
   ```bash
   helm install hcloud-csi hcloud/hcloud-csi -n kube-system
   ```
