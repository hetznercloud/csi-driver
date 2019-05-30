FROM golang:1.12 as builder
WORKDIR /csi/src
ADD src/go.mod src/go.sum /csi/src/
RUN go mod download
ADD src /csi/src/
RUN CGO_ENABLED=0 go build -o driver.bin hetzner.cloud/csi/cmd/driver

FROM alpine:3.7
RUN apk add --no-cache ca-certificates e2fsprogs xfsprogs
COPY --from=builder /csi/src/driver.bin /bin/hcloud-csi-driver
ENTRYPOINT ["/bin/hcloud-csi-driver"]
