FROM golang:1.17 as builder
WORKDIR /csi
ADD go.mod go.sum /csi/
RUN go mod download
ADD . /csi/
RUN CGO_ENABLED=0 go build -o driver.bin github.com/hetznercloud/csi-driver/cmd/driver

FROM alpine:3.13
RUN apk add --no-cache ca-certificates e2fsprogs xfsprogs blkid xfsprogs-extra e2fsprogs-extra btrfs-progs
COPY --from=builder /csi/driver.bin /bin/hcloud-csi-driver
ENTRYPOINT ["/bin/hcloud-csi-driver"]
