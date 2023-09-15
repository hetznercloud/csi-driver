#!/usr/bin/env bash
set -ueo pipefail

helm template hcloud-csi chart \
  --namespace kube-system \
  | grep -v helm.sh/chart \
  > chart/.snapshots/default.yaml

helm template hcloud-csi chart \
  --namespace kube-system \
  -f chart/example-prod.values.yaml \
  | grep -v helm.sh/chart \
  > chart/.snapshots/example-prod.yaml
