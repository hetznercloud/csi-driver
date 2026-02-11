SHELL = bash

K8S_TEST_VERSION ?= v1.35.1

GINKGO = bin/ginkgo
GINKGO_ARGS = -v --flake-attempts=2
GINKGO_EXTRA_ARGS ?= ""

E2E = bin/e2e.test
E2E_ARGS = -storage.testdriver=$(CURDIR)/testdriver.yaml

bin:
	mkdir -p bin
	curl -sSL "https://dl.k8s.io/$(K8S_TEST_VERSION)/kubernetes-test-linux-amd64.tar.gz" | \
		tar --strip-components=3 -C bin -zxf - \
			kubernetes/test/bin/e2e.test \
			kubernetes/test/bin/ginkgo

parallel: bin
	$(GINKGO) $(GINKGO_ARGS) $(GINKGO_EXTRA_ARGS) \
		-nodes=12 \
		-focus='External.Storage' \
		-skip='\[Feature:|\[Disruptive\]|\[Serial\]' \
		$(E2E) -- $(E2E_ARGS)

serial: bin
	$(GINKGO) $(GINKGO_ARGS) $(GINKGO_EXTRA_ARGS) \
		-focus='External.Storage.*(\[Feature:|\[Serial\])' \
		$(E2E) -- $(E2E_ARGS)

test: parallel serial
