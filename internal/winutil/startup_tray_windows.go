//go:build windows

package winutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RunValueNameStartupTray is the HKCU Run value used for "start with Windows" (-tray).
const RunValueNameStartupTray = "AccountSwitcher"

// legacyRunValueNameStartupTray is the value name used before the rename. It is
// still removed whenever the setting is written, because leaving it behind
// would start a second copy of the app at every login, and the settings toggle
// would no longer control it.
const legacyRunValueNameStartupTray = "TcNoAccSwitcher"

const runKeyBase = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run:`

const runKeyStartupTray = runKeyBase + RunValueNameStartupTray

const legacyRunKeyStartupTray = runKeyBase + legacyRunValueNameStartupTray

// RunCommandTrayArgs is appended after the quoted executable for startup-tray mode.
const RunCommandTrayArgs = "-tray"

// RunAtStartupTrayCommand returns the full Run registry string: "path\to\exe" -tray
func RunAtStartupTrayCommand(exePath string) string {
	exePath = filepath.Clean(strings.TrimSpace(exePath))
	if exePath == "" {
		return ""
	}
	return fmt.Sprintf(`"%s" `+RunCommandTrayArgs, exePath)
}

// SetRunAtStartupTray registers or removes the current-user Run entry for tray startup.
func SetRunAtStartupTray(exePath string, enabled bool) error {
	// Clear the pre-rename entry either way, so the two names can never both be
	// registered and start the app twice.
	_ = RegistryWrite(legacyRunKeyStartupTray, "")
	if !enabled {
		return RegistryWrite(runKeyStartupTray, "")
	}
	cmd := RunAtStartupTrayCommand(exePath)
	if cmd == "" {
		return fmt.Errorf("empty executable path")
	}
	return RegistryWrite(runKeyStartupTray, cmd)
}

// SyncRunAtStartupTray ensures the Run entry matches want (idempotent).
func SyncRunAtStartupTray(exePath string, want bool) error {
	return SetRunAtStartupTray(exePath, want)
}
