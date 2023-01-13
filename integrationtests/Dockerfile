FROM alpine:3.17

RUN apk add --no-cache ca-certificates e2fsprogs xfsprogs blkid xfsprogs-extra e2fsprogs-extra btrfs-progs cryptsetup
RUN apk add --no-cache coreutils

WORKDIR /test
COPY integrationtests.tests /test/integrationtests.tests
ENTRYPOINT ["/test/integrationtests.tests"]
