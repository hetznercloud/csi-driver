# Changelog

## [v2.17.0](https://github.com/hetznercloud/csi-driver/releases/tag/v2.17.0)

### Features

- **helm**: add extra annotations for StorageClass  (#1042)

## [v2.16.0](https://github.com/hetznercloud/csi-driver/releases/tag/v2.16.0)

We’ve merged our two separate binaries into a single, unified binary. This change reduces the overall container image size. Additionally, the new binary is now published as a release artifact.

The Docker image includes two compatibility scripts to support seamless image tag upgrades. However, it is still recommended to update your manifests or Helm chart to reflect the changes.

### Features

- single binary (#996)
- publish binary (#1003)

## [v2.15.0](https://github.com/hetznercloud/csi-driver/releases/tag/v2.15.0)

In release v2.14.0, we introduced default labels that are automatically applied to every newly created volume. However, we have identified a bug in the Hetzner Cloud API that allowed invalid label keys during Volume creation. This issue affects the following labels:

1. Labels with the key prefix `csi.hetzner.cloud/` are not permitted.
2. Labels using the pattern `csi.storage.k8s.io/*/*` are invalid due to the use of double slashes in the key.

These labels are informational only and not used by the CSI driver itself. Volumes with incorrect or broken label values will still function as expected; however, editing or updating the labels may not work correctly. To comply with API requirements, Hetzner Cloud will edit the labels for every Volume and replace them with the new labels:

- `csi.hetzner.cloud/created-by` - `managed-by`
- `csi.storage.k8s.io/pvc/name` - `pvc-name`
- `csi.storage.k8s.io/pvc/namespace` - `pvc-namespace`
- `csi.storage.k8s.io/pv/name` - `pv-name`

### Features

- rename default labels on Volumes (#988)

## [v2.14.0](https://github.com/hetznercloud/csi-driver/releases/tag/v2.14.0)

> [!CAUTION]
> Please do not install the version v2.14.0 of the `csi-driver`, install the version v2.15.0 or later instead.

With this release, you can now assign Hetzner Cloud labels to all newly created volumes. For a brief guide, visit [this page](https://github.com/hetznercloud/csi-driver/tree/main/docs/kubernetes#volume-labels).

By default, all volumes are labeled as follows:

- `csi.storage.k8s.io/pvc/name`
- `csi.storage.k8s.io/pvc/namespace`
- `csi.storage.k8s.io/pv/name`
- `csi.hetzner.cloud/created-by=csi-driver`

### Features

- allow to set labels for all volumes (#932)
- add Kubernetes v1.33 and remove EOL version v1.29 (#971)

### Bug Fixes

- **helm**: node extraVolumeMounts are added to wrong container (#957)

## [v2.13.0](https://github.com/hetznercloud/csi-driver/releases/tag/v2.13.0)

### Features

- **helm-chart**: add extraParam for StorageClass (#865)

## [v2.12.0](https://github.com/hetznercloud/csi-driver/releases/tag/v2.12.0)

We are pleased to announce improvements to our Nomad support. While official support is not available, we have developed end-to-end tests that cover the core features of a CSI driver. These tests are integrated into our CI pipeline and validate compatibility with the latest Nomad version.

### Features

- **nomad**: improved nomad support (#798)
- support ReadWriteOncePod (#857)
- add support and tests for Kubernetes 1.32 (#844)

### Bug Fixes

- use extraVolumeMounts for node in daemonset (#843)

## [v2.11.0](https://github.com/hetznercloud/csi-driver/releases/tag/v2.11.0)

### Features

- Added new option enableProvidedByTopology (#780)
- drop tests for kubernetes v1.28 (#796)

### Bug Fixes

- prefer scheduling the csi controller on cloud nodes (#786)

### Kubernetes Support

This version was tested with Kubernetes 1.29 - 1.31. Furthermore, we dropped v1.28 support.

## [v2.10.1](https://github.com/hetznercloud/csi-driver/releases/tag/v2.10.1)

### ⚠️ Removed Feature from v2.10.0

We have reverted a workaround for an upstream issue in the Kubernetes scheduler where nodes without the CSI Plugin (e.g. Robot servers) would still be considered for scheduling, but then creating and attaching the volume fails with no automatic reconciliation of the this error.

Due to variations in the CSI specification implementation, these changes disrupted Nomad clusters, requiring us to revert them. We are actively working on placing this workaround behind a feature flag, allowing Kubernetes users to bypass the upstream issue.

This affects you, if you have set the Helm value `allowedTopologyCloudServer` in v2.10.0. If you are affected by the Kubernetes upstream issue, we will provide a fix in the next minor version v2.11.0.

Learn more about this in [#400](https://github.com/hetznercloud/csi-driver/issues/400) and [#771](https://github.com/hetznercloud/csi-driver/issues/771).

### Bug Fixes

- reverted NodeGetInfo response as it breaks Nomad clusters (#776)

## [2.10.0](https://github.com/hetznercloud/csi-driver/compare/v2.9.0...v2.10.0) (2024-10-29)


### Features

* add support & tests for Kubernetes 1.31 ([#721](https://github.com/hetznercloud/csi-driver/issues/721)) ([85035b9](https://github.com/hetznercloud/csi-driver/commit/85035b906613c40b9cff28e75429a70275b93c8b))
* allow arbitrary length API tokens ([#724](https://github.com/hetznercloud/csi-driver/issues/724)) ([61c3a0e](https://github.com/hetznercloud/csi-driver/commit/61c3a0eb5309cdeac2613da5396d23fb86942ceb))
* allow passing mkfs format options via storage class parameters ([#747](https://github.com/hetznercloud/csi-driver/issues/747)) ([4b9aa4e](https://github.com/hetznercloud/csi-driver/commit/4b9aa4efb65eb938600772b005699eed12b0555f))
* change XFS default options to support older kernels ([#747](https://github.com/hetznercloud/csi-driver/issues/747)) ([4b9aa4e](https://github.com/hetznercloud/csi-driver/commit/4b9aa4efb65eb938600772b005699eed12b0555f))
* drop tests for Kubernetes 1.27 ([#722](https://github.com/hetznercloud/csi-driver/issues/722)) ([d46a54b](https://github.com/hetznercloud/csi-driver/commit/d46a54b232336a4ed02458c0c40c8c7d6a8f003d))
* force pods with volumes to be scheduled on Cloud servers ([#743](https://github.com/hetznercloud/csi-driver/issues/743)) ([702fe01](https://github.com/hetznercloud/csi-driver/commit/702fe01a6650ff3ca336b29b15d8b3291a4f5bd8))
* fstype is directly passed to mkfs: mkfs.&lt;fstype&gt; ([#749](https://github.com/hetznercloud/csi-driver/issues/749)) ([173bf2f](https://github.com/hetznercloud/csi-driver/commit/173bf2f46e9d7ba482b46f4bc5446b20cdc2e5f5))
* support for SELinux mount ([#756](https://github.com/hetznercloud/csi-driver/issues/756)) ([719247e](https://github.com/hetznercloud/csi-driver/commit/719247e4ae8ce7fe7f74064ed7073491f1129459)), closes [#582](https://github.com/hetznercloud/csi-driver/issues/582)
* Support SINGLE_NODE_MULTI_WRITER capability ([#725](https://github.com/hetznercloud/csi-driver/issues/725)) ([cd53c23](https://github.com/hetznercloud/csi-driver/commit/cd53c23d500d555bf53ba423ea4081a9348e19ca)), closes [#327](https://github.com/hetznercloud/csi-driver/issues/327)
* **swarm:** removed workaround support for mock staging/unstaging ([#746](https://github.com/hetznercloud/csi-driver/issues/746)) ([465ec21](https://github.com/hetznercloud/csi-driver/commit/465ec21147cba7a85deed0164b600d012fa9a8ae))


### Bug Fixes

* do not log sensitive mount options ([#755](https://github.com/hetznercloud/csi-driver/issues/755)) ([0b6e860](https://github.com/hetznercloud/csi-driver/commit/0b6e8602c5bc1efccab2132dece916936b53d67b))

## [2.9.0](https://github.com/hetznercloud/csi-driver/compare/v2.8.0...v2.9.0) (2024-08-08)


### Features

* truncate poll exponential backoff function to max 10s ([#674](https://github.com/hetznercloud/csi-driver/issues/674)) ([5785418](https://github.com/hetznercloud/csi-driver/commit/57854187478ee5bfb5f6eff91173dc54f7f04f39))


### Bug Fixes

* xfs options compatibility between alpine and node kernel ([#689](https://github.com/hetznercloud/csi-driver/issues/689)) ([13d853b](https://github.com/hetznercloud/csi-driver/commit/13d853bfe821dc155df8c9eb30074107f732fa2e)), closes [#648](https://github.com/hetznercloud/csi-driver/issues/648)

## [2.8.0](https://github.com/hetznercloud/csi-driver/compare/v2.7.1...v2.8.0) (2024-06-24)


### Features

* **chart:** separate image name and tag ([#616](https://github.com/hetznercloud/csi-driver/issues/616)) ([0300b14](https://github.com/hetznercloud/csi-driver/commit/0300b14dc0a02444ce47a51066b2c5818ad90df4))
* **k8s:** drop Kubernetes 1.26 from tests ([93fe229](https://github.com/hetznercloud/csi-driver/commit/93fe229dfe87e0e6fc0972c6d0d25306e934bd57))
* **k8s:** test against Kubernetes 1.30 ([93fe229](https://github.com/hetznercloud/csi-driver/commit/93fe229dfe87e0e6fc0972c6d0d25306e934bd57))
* read HCLOUD_TOKEN from file ([#617](https://github.com/hetznercloud/csi-driver/issues/617)) ([4e1cd9d](https://github.com/hetznercloud/csi-driver/commit/4e1cd9d47f7056fdb90f481d08f3060e8cf9cb30))

## [2.7.1](https://github.com/hetznercloud/csi-driver/compare/v2.7.0...v2.7.1) (2024-06-13)


### Bug Fixes

* **resize:** recover from part-way failed resizes ([#623](https://github.com/hetznercloud/csi-driver/issues/623)) ([f9016af](https://github.com/hetznercloud/csi-driver/commit/f9016af982c134f391de4cd46c1189fc314411bd))

## [2.7.0](https://github.com/hetznercloud/csi-driver/compare/v2.6.0...v2.7.0) (2024-05-10)


### Features

* add support & tests for Kubernetes 1.29 ([#560](https://github.com/hetznercloud/csi-driver/issues/560)) ([b620486](https://github.com/hetznercloud/csi-driver/commit/b620486ac05de11336a5c8342966a699503b9f86))
* drop tests for kubernetes v1.25 ([#559](https://github.com/hetznercloud/csi-driver/issues/559)) ([5d19ae8](https://github.com/hetznercloud/csi-driver/commit/5d19ae876f1abd9391791ee8f8fa8106394a4901))


### Bug Fixes

* **helm:** missing rbac for leader election / leases ([#538](https://github.com/hetznercloud/csi-driver/issues/538)) ([5da7a14](https://github.com/hetznercloud/csi-driver/commit/5da7a14e9e8dfc6716ede0b72b5aaeac6b8f1306)), closes [#531](https://github.com/hetznercloud/csi-driver/issues/531)

## [2.6.0](https://github.com/hetznercloud/csi-driver/compare/v2.5.1...v2.6.0) (2023-11-08)


### Features

* **helm:** add value to customize kubeletDir ([#516](https://github.com/hetznercloud/csi-driver/issues/516)) ([501fbe3](https://github.com/hetznercloud/csi-driver/commit/501fbe3c226f860fcfa9fed33d92cb994a0d589c))

## [2.5.1](https://github.com/hetznercloud/csi-driver/compare/v2.5.0...v2.5.1) (2023-09-29)


### Bug Fixes

* release workflow fails publishing with missing Git author ([#512](https://github.com/hetznercloud/csi-driver/issues/512)) ([c45b40f](https://github.com/hetznercloud/csi-driver/commit/c45b40f9dd095aa29e8e729859de42bcc8325a55))

## [2.5.0](https://github.com/hetznercloud/csi-driver/compare/v2.4.0...v2.5.0) (2023-09-29)


### Features

* build with Go 1.21 ([#503](https://github.com/hetznercloud/csi-driver/issues/503)) ([2a82c3b](https://github.com/hetznercloud/csi-driver/commit/2a82c3ba9a78802cb82ae6a2d03264d06782e842))
* **k8s:** add Helm Chart ([#500](https://github.com/hetznercloud/csi-driver/issues/500)) ([c39b138](https://github.com/hetznercloud/csi-driver/commit/c39b138421a664b22bdb5b2257c3a50c1d26d0d7))
* test against Kubernetes 1.28 and drop 1.24 ([#491](https://github.com/hetznercloud/csi-driver/issues/491)) ([b331348](https://github.com/hetznercloud/csi-driver/commit/b33134844b5d53bc70e57b30e33a38bdfb33d03e))

## [2.4.0](https://github.com/hetznercloud/csi-driver/compare/v1.6.0...v2.4.0) (2023-08-25)


### Features

* add nomad installation docs ([#467](https://github.com/hetznercloud/csi-driver/issues/467)) ([631b117](https://github.com/hetznercloud/csi-driver/commit/631b1175b874bdf966f05c1e5409f6efacba14f4))
* **docker-swarm:** configure builds for arm64 ([#471](https://github.com/hetznercloud/csi-driver/issues/471)) ([2b9c067](https://github.com/hetznercloud/csi-driver/commit/2b9c067c7bde28eaf894e232aeee371d41129ef2))
* upgrade to hcloud-go v2 ([3622bcc](https://github.com/hetznercloud/csi-driver/commit/3622bccee2e961e39d5ef9bff9c500eb055c3cc5))


### Bug Fixes

* btrfs volumes can not be formatted ([#421](https://github.com/hetznercloud/csi-driver/issues/421)) ([#424](https://github.com/hetznercloud/csi-driver/issues/424)) ([be3fbc9](https://github.com/hetznercloud/csi-driver/commit/be3fbc9be876939b0a0b2197c99193ca52559fcd))
* improve handling of metadata service connectivity ([#466](https://github.com/hetznercloud/csi-driver/issues/466)) ([8ffa5fe](https://github.com/hetznercloud/csi-driver/commit/8ffa5fec6b07515156f5ee2cd1e978f0b1c877c3))

## v2.3.2

Affordable, sustainable & powerful! 🚀You can now get one of our Arm64 CAX servers to optimize your operations while minimizing your costs!
Discover Ampere’s efficient and robust Arm64 architecture and be ready to get blown away with its performance. 😎

Learn more: https://www.hetzner.com/news/arm64-cloud

### What's Changed

* fix(ci): correct image tag in generated k8s manifests by @samcday in https://github.com/hetznercloud/csi-driver/pull/407

**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v2.3.1...v2.3.2

## v2.3.1

Affordable, sustainable & powerful! 🚀You can now get one of our Arm64 CAX servers to optimize your operations while minimizing your costs!
Discover Ampere’s efficient and robust Arm64 architecture and be ready to get blown away with its performance. 😎

Learn more: https://www.hetzner.com/news/arm64-cloud

### What's Changed

* fix(ci): qemu binfmt wrappers during release by @samcday in https://github.com/hetznercloud/csi-driver/pull/406


**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v2.3.0...v2.3.1

## v2.3.0

Affordable, sustainable & powerful! 🚀You can now get one of our Arm64 CAX servers to optimize your operations while minimizing your costs!
Discover Ampere’s efficient and robust Arm64 architecture and be ready to get blown away with its performance. 😎

Learn more: https://www.hetzner.com/news/arm64-cloud

### What's Changed

* docs: Update README for v2.2.0 by @apricote in https://github.com/hetznercloud/csi-driver/pull/388
* chore(ci): run e2e workflow on public workers by @samcday in https://github.com/hetznercloud/csi-driver/pull/393
* Fix typo in Script: Remove quotes. by @guettli in https://github.com/hetznercloud/csi-driver/pull/394
* Add volumeattachment script macos compatibility by @laurigates in https://github.com/hetznercloud/csi-driver/pull/396
* Update README.md by @laurigates in https://github.com/hetznercloud/csi-driver/pull/395
* Fix volumeattachment array by @laurigates in https://github.com/hetznercloud/csi-driver/pull/397
* feat: new dev/test environment by @apricote in https://github.com/hetznercloud/csi-driver/pull/401
* feat(goreleaser): produce OCI manifest images by @apricote in https://github.com/hetznercloud/csi-driver/pull/402
* ci: fix goreleaser arguments by @apricote in https://github.com/hetznercloud/csi-driver/pull/403
* feat: publish ARM container images by @apricote in https://github.com/hetznercloud/csi-driver/pull/405

### New Contributors

* @laurigates made their first contribution in https://github.com/hetznercloud/csi-driver/pull/396

**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v2.2.0...v2.3.0

## v2.3.0-rc.0

### What's Changed

* docs: Update README for v2.2.0 by @apricote in https://github.com/hetznercloud/csi-driver/pull/388
* chore(ci): run e2e workflow on public workers by @samcday in https://github.com/hetznercloud/csi-driver/pull/393
* Fix typo in Script: Remove quotes. by @guettli in https://github.com/hetznercloud/csi-driver/pull/394
* Add volumeattachment script macos compatibility by @laurigates in https://github.com/hetznercloud/csi-driver/pull/396
* Update README.md by @laurigates in https://github.com/hetznercloud/csi-driver/pull/395
* Fix volumeattachment array by @laurigates in https://github.com/hetznercloud/csi-driver/pull/397
* feat: new dev/test environment by @apricote in https://github.com/hetznercloud/csi-driver/pull/401
* feat(goreleaser): produce OCI manifest images by @apricote in https://github.com/hetznercloud/csi-driver/pull/402
* ci: fix goreleaser arguments by @apricote in https://github.com/hetznercloud/csi-driver/pull/403

### New Contributors

* @laurigates made their first contribution in https://github.com/hetznercloud/csi-driver/pull/396

**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v2.2.0...v2.3.0-rc.0

## v2.2.0

### What's Changed

#### Notable Changes

* feat(k8s): pull CSI sidecar images from registry.k8s.io #368 by @apricote in https://github.com/hetznercloud/csi-driver/pull/371
* feat: add experimental Docker Swarm support by @s4ke in https://github.com/hetznercloud/csi-driver/pull/376
* feat: use exponential backoff for action polling by @apricote in https://github.com/hetznercloud/csi-driver/pull/380
* feat: add Grafana dashboard and ServiceMonitors by @apricote in https://github.com/hetznercloud/csi-driver/pull/378
* feat: Separate server ID from location for controller by @NotTheEvilOne in https://github.com/hetznercloud/csi-driver/pull/385

#### Other

* docs: update README for v2.1.1 by @apricote in https://github.com/hetznercloud/csi-driver/pull/367
* Relax volume integration test by @choffmeister in https://github.com/hetznercloud/csi-driver/pull/366
* feat: set allow discards on luksOpen by @apricote in https://github.com/hetznercloud/csi-driver/pull/370
* docs(v2): describe process to prepare VolumeAttachments by @apricote in https://github.com/hetznercloud/csi-driver/pull/375
* deps: bump hcloud-go to 1.40.0 by @apricote in https://github.com/hetznercloud/csi-driver/pull/379
* chore(deps): bump golang.org/x/net from 0.5.0 to 0.7.0 by @dependabot in https://github.com/hetznercloud/csi-driver/pull/383
* ci(e2e): improve behaviour in spite of flakiness by @apricote in https://github.com/hetznercloud/csi-driver/pull/386
* ci(e2e): fail fast on worker setup errors by @apricote in https://github.com/hetznercloud/csi-driver/pull/387

### New Contributors

* @s4ke made their first contribution in https://github.com/hetznercloud/csi-driver/pull/376
* @NotTheEvilOne made their first contribution in https://github.com/hetznercloud/csi-driver/pull/385

**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v2.1.1...v2.2.0

## v2.1.1

### What's Changed

* Build the Docker image on any platform by @hakman in https://github.com/hetznercloud/csi-driver/pull/309
* test: fix e2e suite for NixOS by @apricote in https://github.com/hetznercloud/csi-driver/pull/338
* docs: document procedure to fix PVs broken with v2.0.0 by @apricote in https://github.com/hetznercloud/csi-driver/pull/340
* docs: add missing link to script by @apricote in https://github.com/hetznercloud/csi-driver/pull/341
* encrypted storageclass had the wrong names. reference to cryptsetup by @JustinGuese in https://github.com/hetznercloud/csi-driver/pull/342
* fix(node): check for empty devicePath by @apricote in https://github.com/hetznercloud/csi-driver/pull/344
* docs: describe how to activate debug logs by @apricote in https://github.com/hetznercloud/csi-driver/pull/345
* refactor(skaffold): update manifests to match v2+ resources by @apricote in https://github.com/hetznercloud/csi-driver/pull/348
* feat: test against Kubernetes v1.26 by @apricote in https://github.com/hetznercloud/csi-driver/pull/350
* chore(deps): upgrade kubernetes-csi sidecars by @apricote in https://github.com/hetznercloud/csi-driver/pull/351
* Update k8s sig-storage containers by @choffmeister in https://github.com/hetznercloud/csi-driver/pull/354
* fix: Block volumes are formatted #349 by @apricote in https://github.com/hetznercloud/csi-driver/pull/353
* Verify hcloud-csi.yml is up-to-date by @choffmeister in https://github.com/hetznercloud/csi-driver/pull/356
* chore(e2e): update testdriver definition by @apricote in https://github.com/hetznercloud/csi-driver/pull/360
* chore(deps): update golang.org/x/ dependencies by @apricote in https://github.com/hetznercloud/csi-driver/pull/364
* chore(deps): upgrade go dependencies by @apricote in https://github.com/hetznercloud/csi-driver/pull/365

### New Contributors

* @hakman made their first contribution in https://github.com/hetznercloud/csi-driver/pull/309
* @JustinGuese made their first contribution in https://github.com/hetznercloud/csi-driver/pull/342

**Full Changelog**: https://github.com/hetznercloud/csi-driver/compare/v2.1.0...v2.1.1

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
