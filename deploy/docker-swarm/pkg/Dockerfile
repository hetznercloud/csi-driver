FROM --platform=$TARGETPLATFORM golang:1.21 as builder

WORKDIR /build

ADD go.mod go.sum /build/
RUN go mod download

ADD . /build/
RUN ls -al

ARG CGO_ENABLED=0
RUN go build -o aio.bin github.com/hetznercloud/csi-driver/cmd/aio

FROM --platform=$TARGETPLATFORM alpine:3.20

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
RUN mkdir -p /plugin
COPY --from=builder /build/aio.bin /plugin

ENTRYPOINT [ "/plugin/aio.bin" ]
