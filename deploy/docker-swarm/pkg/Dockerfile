FROM --platform=$TARGETPLATFORM golang:1.25 as builder

WORKDIR /build

ADD go.mod go.sum /build/
RUN go mod download

ADD . /build/
RUN ls -al

ARG CGO_ENABLED=0
RUN go build -o hcloud-csi-driver github.com/hetznercloud/csi-driver/cmd

FROM --platform=$TARGETPLATFORM alpine:3.22

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
COPY --from=builder /build/hcloud-csi-driver /plugin

ENTRYPOINT [ "/plugin/hcloud-csi-driver" ]
CMD [ "-controller", "-node" ]
