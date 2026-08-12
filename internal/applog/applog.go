// Package applog gives slog somewhere to write.
//
// The app is linked with -H windowsgui, so it has no console and stderr goes
// nowhere: every log line the program produced was discarded before anyone could
// read it. This writes them to a file under the user data directory instead, and
// keeps writing to stderr as well for the case where a console is attached.
//
// The file is local and is never uploaded anywhere.
package applog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

const (
	// DirName is the folder the log lives in, under the user data directory.
	DirName = "Logs"
	// FileName is the current log. The previous run's log keeps the same name
	// with ".1" appended.
	FileName = "app.log"

	// maxBytes is when the current log is rotated. Big enough to hold a long
	// session at debug level, small enough to paste somewhere.
	maxBytes = 4 << 20
)

var (
	mu     sync.Mutex
	handle *os.File
)

// Path reports where the log for dataRoot lives, without creating anything.
func Path(dataRoot string) string {
	return filepath.Join(dataRoot, DirName, FileName)
}

// Init points slog at a file under dataRoot, and at stderr.
//
// A failure to open the file is reported but not fatal: losing the log is not a
// reason to refuse to start, and stderr still works when a console is attached.
func Init(dataRoot string, level slog.Level) error {
	sink, err := open(dataRoot)
	if err != nil {
		slog.SetDefault(newLogger(os.Stderr, level))
		return err
	}
	slog.SetDefault(newLogger(fanout{file: sink}, level))
	return nil
}

// fanout writes to the log file, and to the console as a courtesy.
//
// Not io.MultiWriter: that returns on the first writer's error, and under
// -H windowsgui with no console attached os.Stderr is an invalid handle whose
// every write fails. The file would then never be written at all, which is the
// exact failure this package exists to remove.
type fanout struct{ file io.Writer }

func (f fanout) Write(p []byte) (int, error) {
	_, _ = os.Stderr.Write(p)
	return f.file.Write(p)
}

// Close flushes and releases the log file. Safe to call when Init failed.
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if handle == nil {
		return nil
	}
	err := handle.Close()
	handle = nil
	return err
}

func newLogger(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}

func open(dataRoot string) (io.Writer, error) {
	if dataRoot == "" {
		return nil, errors.New("applog: empty data root")
	}
	dir := filepath.Join(dataRoot, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("applog: create %s: %w", dir, err)
	}
	current := filepath.Join(dir, FileName)
	rotateIfLarge(current)

	f, err := os.OpenFile(current, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("applog: open %s: %w", current, err)
	}

	mu.Lock()
	prev := handle
	handle = f
	mu.Unlock()
	if prev != nil {
		_ = prev.Close()
	}
	return f, nil
}

// rotateIfLarge moves an oversized log aside so the new run starts clean, keeping
// exactly one previous file. Errors are ignored on purpose: a log that cannot be
// rotated is still better appended to than dropped.
func rotateIfLarge(current string) {
	fi, err := os.Stat(current)
	if err != nil || fi.IsDir() || fi.Size() < maxBytes {
		return
	}
	previous := current + ".1"
	_ = os.Remove(previous)
	_ = os.Rename(current, previous)
}

// Writer returns the sink Init opened, so a component that builds its own logger
// (Wails keeps a separate one) lands in the same file. Falls back to stderr alone
// when there is no file.
func Writer() io.Writer {
	mu.Lock()
	defer mu.Unlock()
	if handle == nil {
		return os.Stderr
	}
	return fanout{file: handle}
}
