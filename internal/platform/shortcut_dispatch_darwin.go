//go:build darwin

package platform

import pos "account-switcher/internal/platform/os/darwin"

func findExeViaStartMenuShortcuts(entry PlatformEntry, exeName string) (string, bool) {
	return pos.FindExeViaShortcuts()
}
