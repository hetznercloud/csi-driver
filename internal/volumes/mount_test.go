package volumes

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"path/filepath"
	"testing"
)

var _ MountService = (*LinuxMountService)(nil)

func TestPublishRefusesUnverifiedDevice(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(n *fakeNode) string
		wantErr error
	}{
		{
			name: "symlink resolves to another volume",
			setup: func(n *fakeNode) string {
				return n.link("106486781", n.attach("sdc", vpdPage("106478890")))
			},
			wantErr: errDeviceMismatch,
		},
		{
			name: "symlink does not exist yet",
			setup: func(n *fakeNode) string {
				return n.byID("106486781")
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "context is cancelled while waiting",
			setup: func(n *fakeNode) string {
				return n.byID("106486781")
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devicePath := tt.setup(newFakeNode(t))

			ctx, stopWaiting := context.WithCancel(t.Context())
			stopWaiting()

			s := NewLinuxMountService(slog.New(slog.DiscardHandler))
			err := s.Publish(ctx, filepath.Join(t.TempDir(), "mount"), devicePath, "106486781", MountOpts{})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Publish() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
