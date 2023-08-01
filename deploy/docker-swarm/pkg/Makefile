PLUGIN_NAME = hetznercloud/hcloud-csi-driver
PLUGIN_TAG ?= $(shell git describe --tags --exact-match 2> /dev/null || echo dev)-swarm-${ARCH}
ARCH = amd64

all: create

clean:
	@rm -rf ./plugin
	@docker container rm -vf tmp_plugin_build_${ARCH} || true

rootfs: clean
	docker image build --platform=linux/${ARCH} -f Dockerfile -t ${PLUGIN_NAME}:rootfs-${ARCH} ../../../
	mkdir -p ./plugin/rootfs
	docker container create --name tmp_plugin_build_${ARCH} ${PLUGIN_NAME}:rootfs-${ARCH}
	docker container export tmp_plugin_build_${ARCH} | tar -x -C ./plugin/rootfs
	cp config.json ./plugin/
	docker container rm -vf tmp_plugin_build_${ARCH}

create: rootfs
	docker plugin rm -f ${PLUGIN_NAME}:${PLUGIN_TAG} 2> /dev/null || true
	docker plugin create ${PLUGIN_NAME}:${PLUGIN_TAG} ./plugin

enable: create
	docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}

push: create
	docker plugin push ${PLUGIN_NAME}:${PLUGIN_TAG}

push_latest: create
	docker plugin push ${PLUGIN_NAME}:latest-swarm-${ARCH}

.PHONY: clean rootfs create enable push
