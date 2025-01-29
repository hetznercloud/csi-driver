#!/usr/bin/env bash
set -ueo pipefail

# Template the chart with pre-built values to get the legacy deployment files
# Also remove labels that are Helm specific
helm template --kube-version v1.31.4 hcloud-csi chart \
  --namespace kube-system \
  --set metrics.enabled=true \
  --set controller.matchLabelsOverride.app=hcloud-csi-controller \
  --set controller.podLabels.app=hcloud-csi-controller \
  --set node.matchLabelsOverride.app=hcloud-csi \
  --set node.podLabels.app=hcloud-csi |
    grep -v helm.sh/chart |
    grep -v app.kubernetes.io/managed-by \
    > deploy/kubernetes/hcloud-csi.yml
