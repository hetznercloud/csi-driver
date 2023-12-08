FROM golang:1.21 as builder
WORKDIR /csi
ADD go.mod go.sum /csi/
RUN go mod download
ADD . /csi/
RUN ls -al
# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o controller.bin github.com/hetznercloud/csi-driver/cmd/controller
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o node.bin github.com/hetznercloud/csi-driver/cmd/node

FROM alpine:3.19
RUN apk add --no-cache ca-certificates e2fsprogs xfsprogs blkid xfsprogs-extra e2fsprogs-extra btrfs-progs cryptsetup
ENV GOTRACEBACK=all
COPY --from=builder /csi/controller.bin /bin/hcloud-csi-driver-controller
COPY --from=builder /csi/node.bin /bin/hcloud-csi-driver-node
