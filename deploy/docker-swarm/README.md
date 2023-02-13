# Docker Swarm Hetzner CSI plugin

Currently in Beta. Please consult the Docker Swarm documentation
for cluster volumes (=CSI) support at https://github.com/moby/moby/blob/master/docs/cluster_volumes.md

## How to install the plugin

Run the following steps on all nodes (especially master nodes).
The simplest way to achieve this

1. Install the plugin 

```bash
docker plugin install --disable --alias hetznercloud/csi-driver-docker --grant-all-permissions hetznercloud/csi-driver-docker
```

2. Set HCLOUD_TOKEN

```bash
docker plugin set hetznercloud/csi-driver-docker HCLOUD_TOKEN=<your token>
```

3. Enable plugin

```bash
docker plugin enable hetznercloud/csi-driver-docker
```

## How to create a volume

Example: Create a volume wih size 50G in Nuremberg:

```bash
docker volume create --driver hetznercloud/csi-driver-docker --required-bytes 50G --type mount --sharing onewriter --scope single hcloud-debug1 --topology-required nbg1
```

We can now use this in a service:

```bash
docker service create --name hcloud-debug-serv1   --mount type=cluster,src=hcloud-debug1,dst=/srv/www   nginx:alpine
```

Note that only scope `single` is supported as Hetzner Cloud volumes can only be attached to one node at a time

We can however share the volume on multiple containers on the same host:

```bash
docker volume create --driver hetznercloud/csi-driver-docker --required-bytes 50G --type mount --sharing all --scope single hcloud-debug1 --topology-required nbg1
```

After creation we can now use this volume with `--sharing all` in more than one replica:

```bash
docker service create --name hcloud-debug-serv2  --mount type=cluster,src=hcloud-debug2,dst=/srv/www   nginx:alpine
docker service scale hcloud-debug-serv2=2
```

## How to resize a docker swarm Hetzner CSI volume

Currently, the Docker Swarm CSI support does not come with support for volume resizing.
The following explains a step by step guide on how to do this manually instead.

Please test the following on a Swarm with the same version as your target cluster
as this strongly depends on the logic of `docker volume rm -f` not deleting the cloud volume.

### Steps

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

