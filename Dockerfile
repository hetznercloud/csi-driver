FROM golang:1.13 as builder
WORKDIR /csi
ADD go.mod go.sum /csi/
RUN go mod download
ADD . /csi/
RUN CGO_ENABLED=0 go build -o driver.bin hetzner.cloud/csi/cmd/driver

FROM alpine:3.7
RUN apk add --no-cache ca-certificates e2fsprogs xfsprogs
COPY --from=builder /csi/driver.bin /bin/hcloud-csi-driver
ENTRYPOINT ["/bin/hcloud-csi-driver"]
