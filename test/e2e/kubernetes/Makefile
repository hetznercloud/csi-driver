SHELL = bash

K8S_TEST_VERSION ?= v1.37.1
ENCRYPTION ?= false

GINKGO = bin/ginkgo
GINKGO_ARGS = -v --flake-attempts=2
GINKGO_EXTRA_ARGS ?=

TESTDRIVER = $(CURDIR)/testdriver.yaml
TESTDRIVER_ENCRYPTED = $(CURDIR)/bin/testdriver-encrypted.yaml
SETUP =

ifeq ($(ENCRYPTION),true)
TESTDRIVER = $(TESTDRIVER_ENCRYPTED)
SETUP = encryption-setup
endif

E2E = bin/e2e.test
E2E_ARGS = -storage.testdriver=$(TESTDRIVER)

.PHONY: parallel serial test encryption-setup

bin:
	mkdir -p bin
	curl -sSL "https://dl.k8s.io/$(K8S_TEST_VERSION)/kubernetes-test-linux-amd64.tar.gz" | \
		tar --strip-components=3 -C bin -zxf - \
			kubernetes/test/bin/e2e.test \
			kubernetes/test/bin/ginkgo

$(TESTDRIVER_ENCRYPTED): testdriver.yaml | bin
	sed 's/FromName: true/FromExistingClassName: hcloud-volumes-encrypted/' $< > $@

encryption-setup: $(TESTDRIVER_ENCRYPTED)
	kubectl apply -f encryption.yaml

parallel: bin $(TESTDRIVER) $(SETUP)
	$(GINKGO) $(GINKGO_ARGS) $(GINKGO_EXTRA_ARGS) \
		-nodes=12 \
		-focus='External.Storage' \
		-skip='\[Feature:|\[Disruptive\]|\[Serial\]' \
		$(E2E) -- $(E2E_ARGS)

serial: bin $(TESTDRIVER) $(SETUP)
	$(GINKGO) $(GINKGO_ARGS) $(GINKGO_EXTRA_ARGS) \
		-focus='External.Storage.*(\[Feature:|\[Serial\])' \
		$(E2E) -- $(E2E_ARGS)

test: parallel serial
