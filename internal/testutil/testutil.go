package testutil

import (
	"io"
	"log/slog"
)

func NewNopLogger() *slog.Logger {
	th := slog.NewTextHandler(io.Discard, nil)
	return slog.New(th)
}
