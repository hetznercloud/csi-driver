#!/usr/bin/env bash
set -uex -o pipefail
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

k8s_test_version="${K8S_TEST_VERSION:-v1.26.3}"

mkdir -p "${SCRIPT_DIR}/test-binaries"
# TODO: Read linux-amd64 from env
curl --location "https://dl.k8s.io/${k8s_test_version}/kubernetes-test-linux-amd64.tar.gz" | \
  tar --strip-components=3 -C "${SCRIPT_DIR}/test-binaries" -zxf - kubernetes/test/bin/e2e.test kubernetes/test/bin/ginkgo

ginkgo="${SCRIPT_DIR}/test-binaries/ginkgo"
ginkgo_flags="-v --flakeAttempts=2"

e2e="${SCRIPT_DIR}/test-binaries/e2e.test"
e2e_flags="-storage.testdriver=${SCRIPT_DIR}/testdriver-1.23.yaml"

echo "Executing parallel tests"
${ginkgo} ${ginkgo_flags} \
  -nodes=6 \
  -focus='External.Storage' \
  -skip='\[Feature:|\[Disruptive\]|\[Serial\]' \
  "${e2e}" -- ${e2e_flags}

echo "Executing serial tests"
${ginkgo} ${ginkgo_flags} \
  -focus='External.Storage.*(\[Feature:|\[Serial\])' \
  -skip='\[Feature:SELinuxMountReadWriteOncePod\]' \
  "${e2e}" -- ${e2e_flags}

