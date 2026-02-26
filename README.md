# Container Storage Interface driver for Hetzner Cloud

[![Tests](https://github.com/hetznercloud/csi-driver/actions/workflows/test.yml/badge.svg)](https://github.com/hetznercloud/csi-driver/actions/workflows/test.yml)
[![codecov](https://codecov.io/github/hetznercloud/csi-driver/graph/badge.svg?token=OHFN24A0sR)](https://codecov.io/github/hetznercloud/csi-driver/tree/main)

This is a [Container Storage Interface](https://github.com/container-storage-interface/spec) driver for Hetzner Cloud
enabling you to use ReadWriteOnce Volumes within Kubernetes & other Container
Orchestrators. Please note that this driver **requires Kubernetes 1.19 or newer**.

## Kubernetes Docs

- :rocket: See the [quick start guide](docs/kubernetes/quickstart.md) to get you started.
- :book: See the [CSI driver architecture](docs/kubernetes/csi-driver-architecture.md) for the involved components.

### Other Container Orchestrators

**These environments are not officially supported**. However, thanks to their compatibility with the CSI spec and community contributions, quick start guides are available for the following:

- [Docker Swarm](./docs/docker-swarm/README.md)️
- [HashiCorp Nomad](./docs/nomad/README.md)️

## Development

### Setup a development environment

To setup a development environment, make sure you installed the following tools:

- [tofu](https://opentofu.org/)
- [k3sup](https://github.com/alexellis/k3sup)
- [docker](https://www.docker.com/)
- [skaffold](https://skaffold.dev/)

1. Configure a `HCLOUD_TOKEN` in your shell session.

> [!WARNING]
> The development environment runs on Hetzner Cloud servers which will induce costs.

2. Deploy the development cluster:

```sh
make -C dev up
```

3. Load the generated configuration to access the development cluster:

```sh
source dev/files/env.sh
```

4. Check that the development cluster is healthy:

```sh
kubectl get nodes -o wide
```

5. Start developing the CSI driver in the development cluster:

```sh
skaffold dev
```

On code change, skaffold will rebuild the image, redeploy it and print all logs from csi components.

⚠️ Do not forget to clean up the development cluster once are finished:

```sh
make -C dev down
```

### Run the docker e2e tests

To run the integrations tests, make sure you installed the following tools:

- [docker](https://www.docker.com/)

1. Run the following command to run the integrations tests:

```sh
go test -v -tags integration ./test/integration
```

### Run the kubernetes e2e tests

The Hetzner Cloud CSI driver is tested against the official kubernetes e2e tests.

Before running the integrations tests, make sure you followed the [Setup a development environment](#setup-a-development-environment) steps.

1. Run the kubernetes e2e tests using the following command:

```sh
make -C test/e2e/kubernetes test
```

## License

MIT license
