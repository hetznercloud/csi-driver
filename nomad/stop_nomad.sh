#!/usr/bin/env bash

set -euo pipefail

nomad job stop -purge hcloud-csi-controller
echo "Deleted hcloud-csi-controller"

nomad job stop -purge hcloud-csi-node
echo "Deleted hcloud-csi-node"
