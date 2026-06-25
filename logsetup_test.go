package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSetupLoggingBoth verifies that GUSHER_LOG_OUTPUT=both writes structured
// log lines to the rotation file (lumberjack) as well as stdout.
func TestSetupLoggingBoth(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "gusher.log")

	t.Setenv("GUSHER_LOG_OUTPUT", "both")
	t.Setenv("GUSHER_LOG_FORMAT", "json")
	t.Setenv("GUSHER_LOG_FILE", logFile)

	ls, err := setupLoggingFromEnv()
	if err != nil {
		t.Fatalf("setupLoggingFromEnv: %v", err)
	}
	defer ls.Close()

	ls.App.WithField("k", "v").Info("hello-both")
	ls.Slog.Info("engine-line", "n", 1)
	if err := ls.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	b, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	out := string(b)
	if !strings.Contains(out, "hello-both") || !strings.Contains(out, `"k":"v"`) {
		t.Errorf("app line missing from file:\n%s", out)
	}
	if !strings.Contains(out, "engine-line") {
		t.Errorf("engine line missing from file:\n%s", out)
	}
}

// TestSetupLoggingTextFormat verifies the text formatter path is selectable.
func TestSetupLoggingTextFormat(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "g.log")
	t.Setenv("GUSHER_LOG_OUTPUT", "file")
	t.Setenv("GUSHER_LOG_FORMAT", "text")
	t.Setenv("GUSHER_LOG_FILE", logFile)

	ls, err := setupLoggingFromEnv()
	if err != nil {
		t.Fatalf("setupLoggingFromEnv: %v", err)
	}
	ls.App.Info("text-line")
	ls.Close()

	b, _ := os.ReadFile(logFile)
	if !strings.Contains(string(b), "msg=text-line") {
		t.Errorf("expected text-format line, got:\n%s", string(b))
	}
}
