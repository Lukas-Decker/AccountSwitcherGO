package platform

import (
	"log/slog"
	"strings"
	"sync/atomic"
)

// debugLogging mirrors the stored preference for the running process.
var debugLogging atomic.Bool

// debugLevelHook lets main raise or lower the process log level when the
// preference changes, without this package knowing how logging is set up.
var debugLevelHook func(debug bool)

// SetDebugLevelHook registers the level switch.
func SetDebugLevelHook(fn func(debug bool)) { debugLevelHook = fn }

// ApplyDebugLogging turns verbose logging on or off now.
func ApplyDebugLogging(enabled bool) {
	debugLogging.Store(enabled)
	if debugLevelHook != nil {
		debugLevelHook(enabled)
	}
	slog.Info("debug logging changed", "enabled", enabled)
}

// DebugLoggingEnabled reports the current state, for the frontend to decide how
// much to forward.
func DebugLoggingEnabled() bool { return debugLogging.Load() }

// logFrontendLine writes a console line from the webview.
//
// Errors and warnings are kept whatever the setting, since they are rare and are
// exactly what goes missing; the chattier levels are dropped unless debugging is
// on, so an ordinary session does not write a line per render.
func logFrontendLine(level, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	log := slog.Default().With("component", "frontend")
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "error":
		log.Error(message)
	case "warn", "warning":
		log.Warn(message)
	default:
		if debugLogging.Load() {
			log.Info(message)
		}
	}
}
