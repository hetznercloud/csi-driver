FROM alpine:3.19
RUN apk add --no-cache ca-certificates e2fsprogs xfsprogs blkid xfsprogs-extra e2fsprogs-extra btrfs-progs cryptsetup
COPY ./controller.bin /bin/hcloud-csi-driver-controller
COPY ./node.bin /bin/hcloud-csi-driver-node
