FROM alpine:3.15

RUN apk add --no-cache \
    blkid \
    btrfs-progs \
    ca-certificates \
    cryptsetup \
    e2fsprogs \
    e2fsprogs-extra \
    xfsprogs \
    xfsprogs-extra

COPY ./controller.bin /bin/hcloud-csi-driver-controller
COPY ./node.bin /bin/hcloud-csi-driver-node
