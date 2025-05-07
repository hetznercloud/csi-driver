#!/usr/bin/env bash
set -ueo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}")"  &> /dev/null && pwd)"
COVERDIR="$SCRIPT_DIR/../coverage"
COVERDIR_CONTROLLER="$COVERDIR/controller"
COVERDIR_NODE="$COVERDIR/node"

# Create/clean coverage directory
if [ -d "$COVERDIR" ]; then
  echo "$COVERDIR already exists; cleaning"
  rm -r "$COVERDIR"
fi
mkdir "$COVERDIR"

signal_coverage_write() {
  mkdir "$1"
  for i in "${@:2}"; do
    echo "Sending USR1 signal to $i"
    kubectl -n kube-system exec -t "$i" -c hcloud-csi-driver -- kill -USR1 1
    sleep 0.5
    echo "Pulling coverage from $i into $1"
    kubectl cp -n kube-system -c hcloud-csi-driver "$i:/coverage" "$1"
  done

  go tool covdata textfmt -i "$1" -o "$1/coverage.txt"
}

CONTROLLER_PODS=$(
  kubectl -n kube-system get pods \
    --no-headers -o custom-columns=":metadata.name" \
    -l app.kubernetes.io/instance=hcloud-csi \
    -l app.kubernetes.io/component=controller
)
NODE_PODS=$(
  kubectl -n kube-system get pods \
    --no-headers -o custom-columns=":metadata.name" \
    -l app.kubernetes.io/instance=hcloud-csi \
    -l app.kubernetes.io/component=node
)

# shellcheck disable=SC2086
signal_coverage_write "$COVERDIR_CONTROLLER" $CONTROLLER_PODS
# shellcheck disable=SC2086
signal_coverage_write "$COVERDIR_NODE" $NODE_PODS
