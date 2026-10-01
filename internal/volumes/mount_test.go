package volumes

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

var _ MountService = (*LinuxMountService)(nil)

func TestPublishRefusesUnidentifiedDevice(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(attach attachFunc)
		wantErr error
	}{
		{
			name: "no device reports the volume",
			setup: func(attach attachFunc) {
				attach("sdc", vpdPage("106478890"))
				attach("sdd", nil)
			},
			wantErr: errDeviceNotFound,
		},
		{
			name: "two devices report the volume",
			setup: func(attach attachFunc) {
				attach("sdc", vpdPage("106486781"))
				attach("sdd", vpdPage("106486781"))
			},
			wantErr: errAmbiguousDevice,
		},
		{
			name: "device node does not exist yet",
			setup: func(attach attachFunc) {
				_ = os.Remove(attach("sdc", vpdPage("106486781")))
			},
			wantErr: os.ErrNotExist,
		},
		{
			name:    "context is cancelled while waiting",
			setup:   func(attachFunc) {},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(fakeNode(t))

			ctx, stopWaiting := context.WithCancel(t.Context())
			stopWaiting()

			s := NewLinuxMountService(slog.New(slog.DiscardHandler))
			err := s.Publish(ctx, filepath.Join(t.TempDir(), "mount"), "106486781", MountOpts{})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Publish() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
