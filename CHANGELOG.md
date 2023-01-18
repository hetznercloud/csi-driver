# Changelog

## v2.1.0

### What's Changed

* fix: revert invalid topology changes #333 by @apricote in https://github.com/hetznercloud/csi-driver/pull/335


**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v2.0.1...v2.1.0

## v2.0.1

### :warning: This is a bugfix for #333, only update to this release if you are currently on `v2.0.0`. Otherwise check out the issue to learn which version you should install/upgrade to.

### What's Changed

* fix: invalid topology label on new volumes #333 by @apricote in https://github.com/hetznercloud/csi-driver/pull/334


**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v2.0.0...v2.0.1

## v2.0.0

### :stop_sign: Known Bug

:stop_sign: Version v2.0.0 contains a known bug that affects newly created volumes. Please upgrade directly to `v2.1.0`. Details are available in the issue #333.

### Breaking Changes

:warning: This release contains breaking changes from `1.6.0`. Please see the [Upgrading](https://github.com/hetznercloud/csi-driver#from-v1-to-v2) section in the README for details on the required steps.

### What's Changed

* Include more information in request logging by @samcday in https://github.com/hetznercloud/csi-driver/pull/237
* e2e test workflow improvements by @samcday in https://github.com/hetznercloud/csi-driver/pull/234
* driver: fix panic when server is not found. by @jrasell in https://github.com/hetznercloud/csi-driver/pull/249
* Use our own runners in E2E tests by @LKaemmerling in https://github.com/hetznercloud/csi-driver/pull/252
* Remove unneeded privileges from CSI sidecars by @samcday in https://github.com/hetznercloud/csi-driver/pull/235
* Use hcloud API for volume filesystem formatting by @samcday in https://github.com/hetznercloud/csi-driver/pull/238
* Kustomization support and manifest simplification by @samcday in https://github.com/hetznercloud/csi-driver/pull/223
* fix docs:  taints vs label by @jleni in https://github.com/hetznercloud/csi-driver/pull/257
* Update hcloud-csi.yml by @sui77 in https://github.com/hetznercloud/csi-driver/pull/242
* [1/???] Split deployment manifests by @samcday in https://github.com/hetznercloud/csi-driver/pull/261
* [2/???] Rework RBAC to only apply to CSI Controller by @samcday in https://github.com/hetznercloud/csi-driver/pull/262
* [3/???] Switch Controller to Deployment, plus other tweaks by @samcday in https://github.com/hetznercloud/csi-driver/pull/263
* [4/???] Remove hcloud API calls from most Node code paths by @samcday in https://github.com/hetznercloud/csi-driver/pull/264
* [5/???] Split the driver into controller + node binaries by @samcday in https://github.com/hetznercloud/csi-driver/pull/266
* [6/???] Simplify node resize by @samcday in https://github.com/hetznercloud/csi-driver/pull/267
* Rename secret for hcloud api token to `hcloud` by @LKaemmerling in https://github.com/hetznercloud/csi-driver/pull/275
* [7/???] Remove HCLOUD_TOKEN from node DaemonSet by @samcday in https://github.com/hetznercloud/csi-driver/pull/269
* [8/???] Remove Node Stage/Unstage capability by @samcday in https://github.com/hetznercloud/csi-driver/pull/270
* Allow to configure the HCLOUD API Endpoint via Environment Variables. by @LKaemmerling in https://github.com/hetznercloud/csi-driver/pull/277
* Add support for volume encryption with cryptsetup and LUKS by @choffmeister in https://github.com/hetznercloud/csi-driver/pull/279
* Implement ListVolumes Call by @LKaemmerling in https://github.com/hetznercloud/csi-driver/pull/292
* Add FSGroup to mount capabilities + update dependencies  by @4ND3R50N in https://github.com/hetznercloud/csi-driver/pull/296
* Updates Version Constraint by @mvhirsch in https://github.com/hetznercloud/csi-driver/pull/291
* Update k8s support by @4ND3R50N in https://github.com/hetznercloud/csi-driver/pull/298
* Add skaffold for local debugging + add "Local test setup" section to README.md by @4ND3R50N in https://github.com/hetznercloud/csi-driver/pull/301
* ci: publish unstable docker image from main by @EternalDeiwos in https://github.com/hetznercloud/csi-driver/pull/305
* Explicit docs: read+write API token is needed. by @guettli in https://github.com/hetznercloud/csi-driver/pull/313
* StorageClass has cluster scope by @jlgeering in https://github.com/hetznercloud/csi-driver/pull/317
* test: fix integration tests relying on specific byte amounts by @apricote in https://github.com/hetznercloud/csi-driver/pull/322
* feat: test against Kubernetes v1.25 by @apricote in https://github.com/hetznercloud/csi-driver/pull/321
* chore: upgrade all dependencies to latest version by @apricote in https://github.com/hetznercloud/csi-driver/pull/326
* [enhancement] Use native kubernetes topology region label for volumes nodeAffinity by @maksim-paskal in https://github.com/hetznercloud/csi-driver/pull/302
* fix: driver version not updated on tagged release by @apricote in https://github.com/hetznercloud/csi-driver/pull/328
* docs: update README for v2.0.0 by @apricote in https://github.com/hetznercloud/csi-driver/pull/329

### New Contributors

* @jrasell made their first contribution in https://github.com/hetznercloud/csi-driver/pull/249
* @jleni made their first contribution in https://github.com/hetznercloud/csi-driver/pull/257
* @sui77 made their first contribution in https://github.com/hetznercloud/csi-driver/pull/242
* @choffmeister made their first contribution in https://github.com/hetznercloud/csi-driver/pull/279
* @4ND3R50N made their first contribution in https://github.com/hetznercloud/csi-driver/pull/296
* @mvhirsch made their first contribution in https://github.com/hetznercloud/csi-driver/pull/291
* @EternalDeiwos made their first contribution in https://github.com/hetznercloud/csi-driver/pull/305
* @guettli made their first contribution in https://github.com/hetznercloud/csi-driver/pull/311
* @jlgeering made their first contribution in https://github.com/hetznercloud/csi-driver/pull/317
* @maksim-paskal made their first contribution in https://github.com/hetznercloud/csi-driver/pull/302

**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v1.6.0...v2.0.0
## v1.6.0

### Changelog

2ea4803 Add btrfs support
7719e45 Add exclude for blockstorage during resize (#211)
4a69641 Add k8s 1.22 to tests (#225)
beb3783 Adjust stale bot to be more userfriendly (#217)
0de9bd9 CI improvements for speed and fork-friendliness. (#221)
e07b392 Fix changelog generation
8cb0bfe Implement Instrumentation from hcloud-go (#227)
c89c462 Increase default polling interval to 3 seconds. (#230)
11c9940 Make e2e workflow friendly to running on forks. (#214)
29893db Migrate Testsuite Setup to be in line with our CCM Testsuite (#219)
4ad4d69 Prepare release v1.6.0 (#231)
cf4e7e4 Recognition of root servers (#195)
c213244 Reduce default log verbosity to info level (#224)
c74a95b Remove testing for k8s 1.18 as written in our Versioning policy. (#199)
8d1f531 Run e2e tests in parallel. (#215)
da859e8 Simplify CSI socket handling (#222)
6164eaf Update README.md (#196)
140dad9 Update hcloud-go to v1.29.1 (#218)
fb90575 Upgrade csi sidecars to latest versions. (#216)
54f573e Use Go 1.17 (#228)
5d2ac90 Use Goreleaser to publish changelog (#229)
## v1.5.2

- Update Go to 1.16
- Update csi driver container to alpine linux v3.13
- Update hcloud-go to 1.24.0
- Fix mounting idempotency issues

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
