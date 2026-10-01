package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	proto "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
)

func TestRequestLoggerStripsSecrets(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	req := &proto.NodePublishVolumeRequest{
		VolumeId: "106486781",
		Secrets:  map[string]string{"encryption-passphrase": "hunter2"},
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/csi.v1.Node/NodePublishVolume"}
	handler := func(context.Context, any) (any, error) {
		return nil, errors.New("publish failed")
	}

	_, _ = requestLogger(logger)(t.Context(), req, info, handler)

	if strings.Contains(logs.String(), "hunter2") {
		t.Errorf("logs contain the encryption passphrase:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "106486781") {
		t.Errorf("logs do not contain the request:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), `msg="handler failed" method=/csi.v1.Node/NodePublishVolume`) {
		t.Errorf("failed request is not logged with its method:\n%s", logs.String())
	}
}
