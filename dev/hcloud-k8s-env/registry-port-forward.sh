#!/usr/bin/env bash

set -ue -o pipefail

run() {
  unit="k8s-registry-port-forward.service"
  description="Port Forward for Container Registry of k8s dev environment"

  systemctl --user stop "$unit" 2> /dev/null || true

  systemd-run --user \
    --unit="$unit" \
    --description="$description" \
    --same-dir \
    --setenv="KUBECONFIG=$KUBECONFIG" \
    --collect \
    kubectl port-forward -n kube-system svc/docker-registry 30666:5000
}

run
