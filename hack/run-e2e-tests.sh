#!/usr/bin/env bash
set -uex -o pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

kube_version=${KUBE_VERSION:-v1.21.3}

testdriver="$SCRIPT_DIR/e2e-storage-driver.yml"

image="k8s.gcr.io/conformance-amd64:$kube_version"

docker run -v $testdriver:$testdriver -v $KUBECONFIG:$KUBECONFIG -e "KUBECONFIG=$KUBECONFIG" $image \
  /usr/local/bin/ginkgo -succinct \
    -focus='External.Storage.*(\[Feature:|\[Serial\])' \
    -flakeAttempts=2 \
  /usr/local/bin/e2e.test -- \
    -storage.testdriver=$testdriver

docker run -v $testdriver:$testdriver -v $KUBECONFIG:$KUBECONFIG -e "KUBECONFIG=$KUBECONFIG" $image \
  /usr/local/bin/ginkgo -succinct \
    -nodes=3 \
    -focus='External.Storage' \
    -skip='\[Feature:|\[Disruptive\]|\[Serial\]' \
    -flakeAttempts=2 \
  /usr/local/bin/e2e.test -- \
    -storage.testdriver=$testdriver
