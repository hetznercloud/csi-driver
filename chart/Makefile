RELEASE_NAMESPACE=--namespace kube-system
VALUES = $(wildcard *-values.yaml)
SNAPSHOTS = $(VALUES:%-values.yaml=.snapshots/%.yaml)

all: lint snapshots deploy-manifests

# ----- Snapshots -----

.PHONY: snapshots-clean
snapshots-clean:
	rm -f .snapshots/*

.snapshots/default.yaml:
	helm template hcloud-csi . $(RELEASE_NAMESPACE) | grep -v helm.sh/chart > "$@"

.snapshots/%.yaml: %-values.yaml
	helm template hcloud-csi . $(RELEASE_NAMESPACE) -f "$<" | grep -v helm.sh/chart > "$@"

.PHONY: snapshots
snapshots: snapshots-clean .snapshots/*

# ----- Deploy Manifests -----

VALUES_OVERRIDES = \
	--set metrics.enabled=true \
	--set controller.matchLabelsOverride.app=hcloud-csi-controller \
	--set controller.podLabels.app=hcloud-csi-controller \
	--set node.matchLabelsOverride.app=hcloud-csi \
	--set node.podLabels.app=hcloud-csi

.PHONY: deploy-manifests-clean
deploy-manifests-clean:
	rm -f ../deploy/kubernetes/hcloud-csi.yml

../deploy/kubernetes/hcloud-csi.yml:
	helm template hcloud-csi . \
		$(RELEASE_NAMESPACE) \
		$(VALUES_OVERRIDES) | \
		grep -v helm.sh/chart | \
		grep -v app.kubernetes.io/managed-by \
		> ../deploy/kubernetes/hcloud-csi.yml

.PHONY: deploy-manifests
deploy-manifests: deploy-manifests-clean ../deploy/kubernetes/*

# ----- Lint -----

.PHONY: lint
lint:
	helm lint .
