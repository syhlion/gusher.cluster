package main

import (
	"io"
	"log/slog"
	"os"
	"testing"
)

// TestMain initializes the package-level globals the handlers rely on
// (logger, listenChannelPrefix) exactly once, before any test starts. Tests
// must not reassign these — doing so races with in-flight server goroutines
// from earlier tests under -race.
func TestMain(m *testing.M) {
	logger = newLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	listenChannelPrefix = "gushere2e."
	os.Exit(m.Run())
}
