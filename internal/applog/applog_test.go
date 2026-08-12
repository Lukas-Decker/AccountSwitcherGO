package applog

import (
	"log"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestInitWritesAndRotates(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = Close() })

	if err := Init(dir, slog.LevelInfo); err != nil {
		t.Fatalf("init: %v", err)
	}
	slog.Info("hello", "platform", "Epic Games")
	if err := Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	body, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(body), "Epic Games") {
		t.Fatalf("log line did not reach the file: %q", body)
	}

	// An oversized log is moved aside, so a run always starts with room to write
	// and exactly one previous file is kept.
	if err := os.WriteFile(Path(dir), make([]byte, maxBytes+1), 0o644); err != nil {
		t.Fatalf("grow log: %v", err)
	}
	if err := Init(dir, slog.LevelInfo); err != nil {
		t.Fatalf("re-init: %v", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(Path(dir) + ".1"); err != nil {
		t.Errorf("previous log was not kept: %v", err)
	}
	fi, err := os.Stat(Path(dir))
	if err != nil {
		t.Fatalf("stat current: %v", err)
	}
	if fi.Size() >= maxBytes {
		t.Errorf("current log was not rotated: %d bytes", fi.Size())
	}
}

// Under -H windowsgui with no console, every write to os.Stderr fails. The file
// is the whole point of this package, so a broken console must not stop it.
func TestAFailingConsoleDoesNotStopTheFile(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = Close() })

	closed, err := os.CreateTemp(dir, "closed")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	name := closed.Name()
	if err := closed.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	broken, err := os.Open(name) // read-only: every Write returns an error
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer broken.Close()

	realStderr := os.Stderr
	os.Stderr = broken
	defer func() { os.Stderr = realStderr }()

	if err := Init(dir, slog.LevelInfo); err != nil {
		t.Fatalf("init: %v", err)
	}
	slog.Info("survives a dead console")
	if err := Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	body, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(body), "survives a dead console") {
		t.Fatalf("a failing console swallowed the log line: %q", body)
	}
}

// Chunks of the codebase, the process-killing path among them, log through the
// standard logger rather than slog. main routes it through Writer; this is that
// composition, so the redirect cannot quietly stop working.
func TestStandardLoggerReachesTheFile(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = Close() })

	if err := Init(dir, slog.LevelInfo); err != nil {
		t.Fatalf("init: %v", err)
	}
	previous := log.Writer()
	log.SetOutput(Writer())
	defer log.SetOutput(previous)

	log.Printf("winutil: stopping process=%s method=%s", "EpicGamesLauncher.exe", "Combined")
	if err := Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	body, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(body), "stopping process=EpicGamesLauncher.exe") {
		t.Fatalf("standard logger output did not reach the file: %q", body)
	}
}

// Losing the log is not a reason to refuse to start.
func TestInitWithoutADataRootStillLogs(t *testing.T) {
	t.Cleanup(func() { _ = Close() })
	if err := Init("", slog.LevelInfo); err == nil {
		t.Fatal("expected an error for an empty data root")
	}
	slog.Info("still works")
}
