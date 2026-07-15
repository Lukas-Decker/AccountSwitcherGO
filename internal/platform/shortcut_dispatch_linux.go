//go:build linux

package platform

import pos "account-switcher/internal/platform/os/linux"

func findExeViaStartMenuShortcuts(entry PlatformEntry, exeName string) (string, bool) {
	return pos.FindExeViaShortcuts()
}
