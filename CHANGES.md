# Changes

## Unreleased

- Update Go to 1.16
- Update csi driver container to alpine linux v3.13

## v1.5.1

- Add correct deployment file for latest version

## v1.5.0

- Allow using the node name as node indicator instead of using the
  metadata service
- Allow to tune the log level using the `LOG_LEVEL` environment variable
- Update k8s dependencies to v1.17.12
- Update Go to 1.15
- Update hcloud-go to 1.22.0
- Update csi driver container to alpine linux v3.12
- Note: As of this release all versions are end-to-end tested against the 
  official Kubernetes testsuite, as a result a few smaller issues where fixed

## v1.4.0

- Allow mounting of Hetzner Cloud Volumes as raw block volumes.
- Add label (`app: hcloud-csi`) to `hcloud-csi-controller-metrics` and `hcloud-csi-node-metrics`
- Update to hcloud-go 1.18.0

## v1.3.2

- Fix stuck volume terminating when the volume was already deleted

## v1.3.1

- Add correct deployment file for latest version

## v1.3.0

- Update `csi-attacher` sidecar to v2.2.0
- Update `csi-provisioner` sidecar to v1.6.0
- Update `csi-node-driver-registrar` sidecar to v1.3.0
- Add livenessProbe support
- Update Go to 1.14
- Reduce the amount of API calls from CSI driver
- Add option to configure the Action polling interval via `HCLOUD_POLLING_INTERVAL_SECONDS`
- Add option to enable the debug mode via `HCLOUD_DEBUG`

## v1.2.3

- Add missing RBAC rules required for newer k8s version
- Install `e2fsprogs-extra` for resizing
- Add better error handling and validation for certain errors related to wrong API tokens

## v1.2.2

- Fix usage of `Aborted` error code, which leads to an increasing CPU usage

## v1.2.1

- Add missing RBAC rules required for newer k8s version

## v1.2.0

- Implement volume resizing
- Implement volume statistics

## v1.1.5

- Revert fix from v1.1.2 to retry attach/detach when server is locked

## v1.1.4

- Respect minimum volume size of 10 GB

## v1.1.3

- Detach volumes before deleting them

## v1.1.2

- Fix error handling for attaching/detaching volumes in case server is locked

## v1.1.1

- Improve logging

## v1.1.0

- Implement topology awareness (supporting nodes and volumes in different locations)

## v1.0.0

- Initial release
