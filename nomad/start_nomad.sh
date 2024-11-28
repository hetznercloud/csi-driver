#!/bin/bash

set -euo pipefail

if nomad job inspect hcloud-csi-controller > /dev/null
then
    nomad job stop -purge hcloud-csi-controller
fi

controller="$(mktemp)"
envsubst < "./nomad/hcloud-csi-controller.hcl" > $controller
sed -i 's/localhost:30666/docker-registry.service.consul:5000/' $controller
nomad job run $controller

if nomad job inspect hcloud-csi-node > /dev/null
then
    nomad job stop -purge hcloud-csi-node
fi

node="$(mktemp)"
envsubst < "./nomad/hcloud-csi-node.hcl" > $node
sed -i 's/localhost:30666/docker-registry.service.consul:5000/' $node
nomad job run $node
