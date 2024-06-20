FROM alpine:3.13

RUN apk add --no-cache \
    blkid \
    btrfs-progs \
    ca-certificates \
    cryptsetup \
    e2fsprogs \
    e2fsprogs-extra \
    xfsprogs \
    xfsprogs-extra

RUN apk add --no-cache \
    coreutils

WORKDIR /test
COPY integration.tests /test/integration.tests
ENTRYPOINT ["/test/integration.tests"]
