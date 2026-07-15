package app

import (
	"strings"

	"account-switcher/internal/basic"
	"account-switcher/internal/platform"
	"account-switcher/internal/security"
	"account-switcher/internal/steam"
)

// RegisterStartupAccountCounts wires per-platform account totals for GetStartup skeleton hints.
func RegisterStartupAccountCounts() {
	platform.SetStartupAccountCountResolver(resolveStartupAccountCounts)
	platform.SetStartupTagCountResolver(resolveStartupTagCounts)
}

func resolveStartupAccountCounts(platformNames []string) map[string]int {
	out := make(map[string]int, len(platformNames))
	if security.AppLocked() {
		return out
	}
	for _, name := range platformNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if strings.EqualFold(name, steam.PlatformKey) {
			out[name] = steam.CountSavedAccounts()
		} else {
			out[name] = basic.CountSavedAccounts(name)
		}
	}
	return out
}

func resolveStartupTagCounts(platformNames []string) map[string]platform.PlatformTagCountInfo {
	out := make(map[string]platform.PlatformTagCountInfo, len(platformNames))
	if security.AppLocked() {
		return out
	}
	for _, name := range platformNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = platform.PlatformTagCountInfo{
			TagCount:           basic.CountTags(name),
			TaggedAccountCount: basic.CountTaggedAccounts(name),
		}
	}
	return out
}
