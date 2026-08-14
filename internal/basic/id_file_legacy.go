package basic

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// The unique-id file this fork writes used to carry the original project's name.
// Renaming it is cosmetic, but the value inside is what identifies an account, so
// a rename that misses a copy does not merely fail: ensureUniqueIDOnSave mints a
// fresh id when it cannot find the file, which silently detaches a saved account
// from the profile it belongs to.
//
// Both names are therefore accepted, with the legacy one adopted in passing. That
// is done on read rather than as a one-shot migration at startup, because the
// cached copies inside encrypted account blobs are only ever on disk while a
// restore is running and cannot be reached without the user's password.
const (
	idFileName       = "AccountSwitcher-ID.instance"
	legacyIDFileName = "TcNoAccSwitcher-ID.instance"
)

// legacyIDFileSibling returns the pre-rename path beside p, or "" when p is not
// an id file at all.
func legacyIDFileSibling(p string) string {
	if !strings.EqualFold(filepath.Base(p), idFileName) {
		return ""
	}
	return filepath.Join(filepath.Dir(p), legacyIDFileName)
}

// resolveIDFilePath returns the path to read for an id file.
//
// The current name wins. Where only the legacy name is present it is renamed
// into place, so a machine converges on the new name as accounts are used; if
// the rename cannot be done, the legacy path is returned so the caller still
// reads the right value rather than treating the account as new.
func resolveIDFilePath(p string) string {
	if _, err := os.Stat(p); err == nil {
		return p
	}
	legacy := legacyIDFileSibling(p)
	if legacy == "" {
		return p
	}
	if _, err := os.Stat(legacy); err != nil {
		return p
	}
	if err := os.Rename(legacy, p); err != nil {
		slog.Debug("could not adopt legacy id file, reading it in place",
			"legacy", legacy, "current", p, "err", err)
		return legacy
	}
	slog.Info("adopted legacy id file", "from", legacy, "to", p)
	return p
}
