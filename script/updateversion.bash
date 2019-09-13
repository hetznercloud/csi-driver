#!/bin/bash
VERSION="$1"
sed -i "" -e "s/image: hetznercloud\/hcloud-csi-driver:.*$/image: hetznercloud\/hcloud-csi-driver:$VERSION/g" deploy/kubernetes/*.yml
sed -i "" -e "s/## master/## v$VERSION/g" CHANGES.md
sed -i "" -e "s/PluginVersion = \".*\"$/PluginVersion = \"$VERSION\"/g" driver/driver.go
goimports -w driver/driver.go
