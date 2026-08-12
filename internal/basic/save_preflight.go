package basic

import (
	"errors"
	"os"
	"sort"
	"strings"

	"account-switcher/internal/winutil"
)

// ErrLoginInputsMissing is returned when a save cannot produce a usable account
// because the platform's own login state is not on disk.
var ErrLoginInputsMissing = errors.New("login inputs missing")

// missingRequiredLoginInputs lists the login inputs a save needs and cannot find.
//
// Only the two unambiguous kinds are checked, a plain file path and a single
// registry value, so a preflight can never block a save that would have worked:
// globs, JSON selectors and key enumerations are left to the copy loop, which
// already reports them.
//
// The point is the message. Without this the first missing input surfaces as a
// raw "The system cannot find the file specified", which says nothing about which
// of several inputs was missing or what the user could do about it.
func missingRequiredLoginInputs(fc FlowContext) []string {
	if !fc.Descriptor.AllFilesRequired {
		return nil
	}
	var missing []string
	for liveKey := range fc.Descriptor.LoginFiles {
		liveKey = strings.TrimSpace(liveKey)
		if liveKey == "" {
			continue
		}
		if isREG(liveKey) {
			enc := stripREG(liveKey)
			if _, isAll := winutil.RegistryKeyPathForAllValuesSpecifier(enc); isAll {
				continue
			}
			if _, _, isGlob := splitRegistryKeyPathAndValueGlob(enc); isGlob {
				continue
			}
			if _, _, err := winutil.RegistryRead(enc); err != nil {
				missing = append(missing, enc)
			}
			continue
		}
		if isJSONSelect(liveKey) || isJSONEmptyValue(liveKey) {
			continue
		}
		src := expandPlatformPath(liveKey, fc.Folder, fc.PathCtx)
		if hasGlobPattern(src) {
			continue
		}
		if _, err := os.Stat(src); err != nil {
			missing = append(missing, src)
		}
	}
	sort.Strings(missing)
	return missing
}

func pluralIsAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}
