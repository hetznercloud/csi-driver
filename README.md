# Container Storage Interface driver for Hetzner Cloud

[![GitHub Actions status](https://github.com/hetznercloud/csi-driver/workflows/Run%20tests/badge.svg)](https://github.com/hetznercloud/csi-driver/actions)
[![codecov](https://codecov.io/github/hetznercloud/csi-driver/graph/badge.svg?token=OHFN24A0sR)](https://codecov.io/github/hetznercloud/csi-driver/tree/main)

This is a [Container Storage Interface](https://github.com/container-storage-interface/spec) driver for Hetzner Cloud
enabling you to use ReadWriteOnce Volumes within Kubernetes & other Container
Orchestrators. Please note that this driver **requires Kubernetes 1.19 or newer**.

## Getting Started

Depending on your Container Orchestrator you need to follow different steps to
get started with the Hetzner Cloud csi-driver. You can also find other docs
relevant to that Container Orchestrator behind the link:

### [Kubernetes](./docs/kubernetes/README.md#getting-started)

### [Docker Swarm](./docs/docker-swarm/README.md)️ _⚠ Not officially supported_

### [HashiCorp Nomad](./docs/nomad/README.md)️ _⚠ Not officially supported_

## Tests

### Integration Tests

**Requirements: Docker**

The core operations like publishing and resizing can be tested locally with Docker.

```bash
go test $(go list ./... | grep integration) -v
```

### E2E Tests

> ⚠️ Kubernetes E2E Tests were recently refactored and the docs are now outdated.
> See the [GitHub Actions workflow](.github/workflows/test_e2e.yml) for an
> up-to-date script to run the e2e tests.

The Hetzner Cloud CSI Driver was tested against the official k8s e2e
tests for a specific version. You can run the tests with the following
commands. Keep in mind, that these tests run on real cloud servers and
will create volumes that will be billed.

**Test Server Setup**:

1x CPX21 (Ubuntu 18.04)

**Requirements: Docker and Go 1.17**

1. Configure your environment correctly
   ```bash
   export HCLOUD_TOKEN=<specify a project token>
   export K8S_VERSION=1.21.0 # The specific (latest) version is needed here
   export USE_SSH_KEYS=key1,key2 # Name or IDs of your SSH Keys within the Hetzner Cloud, the servers will be accessible with that keys
   ```
2. Run the tests
   ```bash
   go test $(go list ./... | grep e2e) -v -timeout 60m
   ```

The tests will now run, this will take a while (~30 min).

**If the tests fail, make sure to clean up the project with the Hetzner Cloud Console or the hcloud cli.**

### Local test setup  

> ⚠️ Local Kubernetes Dev Setup was recently refactored and the docs are now
> outdated. Check out the scripts [dev-up.sh](hack/dev-up.sh) &
> [dev-down.sh](hack/dev-down.sh) for an automatic dev setup.

This repository provides [skaffold](https://skaffold.dev/) to easily deploy / debug this driver on demand

#### Requirements
1. Install [hcloud-cli](https://github.com/hetznercloud/cli)
2. Install [k3sup](https://github.com/alexellis/k3sup)
3. Install [cilium](https://github.com/cilium/cilium-cli)
4. Install [docker](https://www.docker.com/)

You will also need to set a `HCLOUD_TOKEN` in your shell session

#### Manual Installation guide

1. Create an SSH key

Assuming you already have created an ssh key via `ssh-keygen`
```
hcloud ssh-key create --name ssh-key-csi-test --public-key-from-file ~/.ssh/id_rsa.pub 
```

2. Create a server
```
hcloud server create --name csi-test-server --image ubuntu-20.04 --ssh-key ssh-key-csi-test --type cx22 
```

3. Setup k3s on this server
```
k3sup install --ip $(hcloud server ip csi-test-server) --local-path=/tmp/kubeconfig --cluster --k3s-channel=v1.23 --k3s-extra-args='--no-flannel --no-deploy=servicelb --no-deploy=traefik --disable-cloud-controller --disable-network-policy --kubelet-arg=cloud-provider=external'
```
- The kubeconfig will be created under `/tmp/kubeconfig`
- Kubernetes version can be configured via `--k3s-channel`

4. Switch your kubeconfig to the test cluster
```
export KUBECONFIG=/tmp/kubeconfig
```

5. Install cilium + test your cluster
```
cilium install
```

6. Add your secret to the cluster
```
kubectl -n kube-system create secret generic hcloud --from-literal="token=$HCLOUD_TOKEN"
```

7. Install hcloud-cloud-controller-manager + test your cluster
```
kubectl apply -f  https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/latest/download/ccm.yaml
kubectl config set-context default
kubectl get node -o wide
```

8. Deploy your CSI driver
```
SKAFFOLD_DEFAULT_REPO=naokiii skaffold dev
```
- `docker login` required
- Skaffold is using your own dockerhub repo to push the CSI image.

On code change, skaffold will repack the image & deploy it to your test cluster again. Also, it is printing all logs from csi components.

## License

MIT license
