FROM alpine:3.13

RUN apk add --no-cache ca-certificates e2fsprogs xfsprogs blkid xfsprogs-extra e2fsprogs-extra btrfs-progs cryptsetup
RUN apk add --no-cache coreutils

WORKDIR /test
COPY integration.tests /test/integration.tests
ENTRYPOINT ["/test/integration.tests"]
