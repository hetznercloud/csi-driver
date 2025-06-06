FROM alpine:3.22

RUN apk add --no-cache \
    blkid \
    btrfs-progs \
    ca-certificates \
    cryptsetup \
    e2fsprogs \
    e2fsprogs-extra \
    xfsprogs \
    xfsprogs-extra

COPY ./hcloud-csi-driver /bin/hcloud-csi-driver
