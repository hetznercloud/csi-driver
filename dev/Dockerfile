FROM golang:1.25 as builder

WORKDIR /build

ADD go.mod go.sum /build/
RUN go mod download

ADD . /build/
RUN ls -al

ARG GOOS=linux
ARG GOARCH=amd64
ARG CGO_ENABLED=0
# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
ARG GO_BUILDFLAGS

RUN --mount=type=cache,target=/root/.cache/go-build go build -gcflags="$SKAFFOLD_GO_GCFLAGS" $GO_BUILDFLAGS -o hcloud-csi-driver github.com/hetznercloud/csi-driver/cmd

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

ENV GOTRACEBACK=all
COPY --from=builder /build/hcloud-csi-driver /bin/hcloud-csi-driver

# Creating compatibility wrapper scripts to avoid breaking existing installations
# that rely on separate entrypoints. This ensures upgrading only the image tag is possible.
RUN echo -e '#!/bin/sh\nexec /bin/hcloud-csi-driver -node "$@"' > /bin/hcloud-csi-driver-node && \
    echo -e '#!/bin/sh\nexec /bin/hcloud-csi-driver -controller "$@"' > /bin/hcloud-csi-driver-controller && \
    chmod +x /bin/hcloud-csi-driver-node /bin/hcloud-csi-driver-controller

ENTRYPOINT ["/bin/hcloud-csi-driver"]
