#!/usr/bin/env bash
set -ueo pipefail

# Template the chart with pre-built values to get the legacy deployment files
helm template hcloud-csi chart \
  --namespace kube-system \
  --set metrics.enabled=true \
  --set controller.matchLabelsOverride.app=hcloud-csi-controller \
  --set controller.podLabels.app=hcloud-csi-controller \
  --set node.matchLabelsOverride.app=hcloud-csi \
  --set node.podLabels.app=hcloud-csi \
  > deploy/kubernetes/hcloud-csi.yml
