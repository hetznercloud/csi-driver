#!/usr/bin/env bash
set -ueo pipefail

helm template --kube-version v1.31.4 hcloud-csi chart \
  --namespace kube-system |
    grep -v helm.sh/chart \
    > chart/.snapshots/default.yaml

helm template --kube-version v1.31.4 hcloud-csi chart \
  --namespace kube-system \
  -f chart/example-prod.values.yaml |
    grep -v helm.sh/chart \
    > chart/.snapshots/example-prod.yaml

helm template --kube-version v1.31.4 hcloud-csi chart \
  --namespace kube-system \
  -f chart/.snapshots/full.values.yaml |
    grep -v helm.sh/chart \
    > chart/.snapshots/full.yaml
