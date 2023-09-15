#!/usr/bin/env bash
set -ueo pipefail

VERSION="$1"

if [[ -z $VERSION ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# Update version
sed -e "s/version: .*/version: $VERSION/" --in-place chart/Chart.yaml

# Package the chart for publishing
helm package chart
