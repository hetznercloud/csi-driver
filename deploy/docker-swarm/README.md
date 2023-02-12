# How to resize a docker swarm Hetzner CSI volume

Currently, the Docker Swarm CSI support does not come with support for volume resizing.
The following explains a step by step guide on how to do this manually instead.

## Steps

1. Drain Volume

```
docker volume update <volume-name> --availability drain
```

This way, we ensure that all services stop using the volume.

2. Force remove volume on cluster

```
docker volume rm -f <volume-name>
```

4. Resize Volume in Hetzner UI
5. Recreate Volume with new size to make it known to Swarm again

```
docker volume create   --driver hetznercloud/csi-driver-docker:dev --required-bytes <new-size>  --type mount   --sharing onewriter   --scope single <volume-name>
```

6. Verify that volume exists again:

```
docker volume ls --cluster
```

