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

# Creating compatibility wrapper scripts to avoid breaking existing installations
# that rely on separate entrypoints. This ensures upgrading only the image tag is possible.
RUN echo -e '#!/bin/sh\nexec /bin/hcloud-csi-driver -node "$@"' > /bin/hcloud-csi-driver-node && \
    echo -e '#!/bin/sh\nexec /bin/hcloud-csi-driver -controller "$@"' > /bin/hcloud-csi-driver-controller && \
    chmod +x /bin/hcloud-csi-driver-node /bin/hcloud-csi-driver-controller

ENTRYPOINT /bin/hcloud-csi-driver
