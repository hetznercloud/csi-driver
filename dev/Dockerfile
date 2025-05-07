FROM golang:1.24 as builder

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

RUN --mount=type=cache,target=/root/.cache/go-build go build -gcflags="$SKAFFOLD_GO_GCFLAGS" $GO_BUILDFLAGS -o controller.bin github.com/hetznercloud/csi-driver/cmd/controller
RUN --mount=type=cache,target=/root/.cache/go-build go build -gcflags="$SKAFFOLD_GO_GCFLAGS" $GO_BUILDFLAGS -o node.bin github.com/hetznercloud/csi-driver/cmd/node

FROM alpine:3.21

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
COPY --from=builder /build/controller.bin /bin/hcloud-csi-driver-controller
COPY --from=builder /build/node.bin /bin/hcloud-csi-driver-node
