# Nomad Development

There is no official support from our side for Nomad. Nonetheless, we would still like to offer a satisfying developer experience and have automated e2e tests to avoid breaking Nomad accidentally.

## Nomad Development Environment

As a prerequisite for developing Nomad, a setup of the [`nomad-dev-env`](https://github.com/hetznercloud/nomad-dev-env) is necessary, which is located in `nomad/dev`.

1. Setup the `HCLOUD_TOKEN` environment variable
2. Deploy the development cluster:

```bash
make -C nomad/dev up
```

3. Load the generated configuration to access the development cluster:

```bash
source nomad/dev/files/env.sh
```

4. Check that the cluster is healthy:

```bash
nomad node status
```

## Skaffold

Skaffold commands should be executed from the `csi-driver` root directory and use the `-f` flag to point to the Nomad specific `skaffold.yaml`.

```bash
skaffold -f nomad/skaffold.yaml build
```

Skaffold does not offer any native support for Nomad. For this reason we use the Nomad post build hooks to deploy/redeploy the csi plugin. To delete the csi plugin a manual execution of `stop_nomad.sh` is necessary.

```bash
bash ./nomad/stop_nomad.sh
```

## E2E Tests

The nomad e2e tests are located in `test/e2e/nomad` and need a working development environment.

1. Deploy the csi-driver

```bash
skaffold -f nomad/skaffold.yaml build
```

2. Run the e2e tests

```bash
go test -v -tags e2e ./test/e2e/nomad/...
```
