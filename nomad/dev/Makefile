SHELL = bash

ENV ?= dev

env.auto.tfvars:
	@echo 'hcloud_token = "$(HCLOUD_TOKEN)"' >> "$@"

.terraform:
	tofu init

validate: .terraform
	tofu validate

up: .terraform env.auto.tfvars
	tofu apply -auto-approve
	$(MAKE) port-forward

down: .terraform env.auto.tfvars
	if test -f files/registry-port-forward.sh; then files/registry-port-forward.sh down; fi
	tofu destroy -auto-approve

port-forward:
	files/registry-port-forward.sh up

clean:
	rm -Rf files/ .terraform/ terraform.tfstate* env.auto.tfvars
