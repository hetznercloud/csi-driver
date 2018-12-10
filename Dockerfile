FROM golang:1.11 as builder
WORKDIR /csi
ADD . /csi
WORKDIR /csi/src
RUN rm go.sum # workaround for https://github.com/golang/go/issues/27925
RUN CGO_ENABLED=0 go build -o driver.bin hetzner.cloud/csi/cmd/driver

FROM alpine:3.7
RUN apk add --no-cache ca-certificates e2fsprogs
COPY --from=builder /csi/src/driver.bin /bin/hcloud-csi-driver
ENTRYPOINT ["/bin/hcloud-csi-driver"]
