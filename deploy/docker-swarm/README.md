# How to resize a docker swarm Hetzner CSI volume

Currently, the Docker Swarm CSI support does not come with support for volume resizing.
The following explains a step by step guide on how to do this manually instead.

Please test the following on a Swarm with the same version as your target cluster
as this strongly depends on the logic of `docker volume rm -f` not deleting the cloud volume.

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
5. Attach Volume to temporary server manually
6. Run resize2fs manually
7. Detach Volume from temporary server manually
8. Recreate Volume with new size to make it known to Swarm again

```
docker volume create   --driver hetznercloud/csi-driver-docker:dev --required-bytes <new-size>  --type mount   --sharing onewriter   --scope single <volume-name>
```

9. Verify that volume exists again:

```
docker volume ls --cluster
```

