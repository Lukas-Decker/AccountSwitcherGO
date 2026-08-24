package shortcuts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"account-switcher/internal/paths"
	"account-switcher/internal/profileimage"
	"account-switcher/internal/winutil"
)

// platformSwitcherLnkPath returns the Desktop path for the shortcut that opens
// this platform's page in the app.
func platformSwitcherLnkPath(platformKey string) (string, error) {
	return platformSwitcherLnkPathNamed(platformKey, false)
}

// platformSwitcherLnkPathNamed builds either the current name or the one used
// before the app was rebranded.
func platformSwitcherLnkPathNamed(platformKey string, legacy bool) (string, error) {
	platformKey = strings.TrimSpace(platformKey)
	if platformKey == "" {
		return "", fmt.Errorf("missing platform")
	}
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	if desktop == "" || strings.TrimSpace(os.Getenv("USERPROFILE")) == "" {
		return "", fmt.Errorf("desktop path unknown")
	}
	name := sanitizeShortcutFileName(platformKey)
	base := "Account Switcher - " + name
	if legacy {
		base = "TcNo - " + name + " Switcher"
	}
	return filepath.Join(desktop, base+".lnk"), nil
}

// platformSwitcherLnkPaths returns the current path first, then any older name
// still worth looking at.
//
// A shortcut created before the rename still sits on the user's Desktop under
// the old name. Without checking for it, the settings toggle would report the
// shortcut as missing, creating a second one beside the first, and turning the
// toggle off again would leave the old one behind for good.
func platformSwitcherLnkPaths(platformKey string) ([]string, error) {
	current, err := platformSwitcherLnkPathNamed(platformKey, false)
	if err != nil {
		return nil, err
	}
	legacy, err := platformSwitcherLnkPathNamed(platformKey, true)
	if err != nil {
		return []string{current}, nil
	}
	return []string{current, legacy}, nil
}

// PlatformShortcutExists reports whether the platform switcher .lnk exists on the user's Desktop.
func PlatformShortcutExists(platformKey string) (bool, error) {
	candidates, err := platformSwitcherLnkPaths(platformKey)
	if err != nil {
		return false, err
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}
	return false, nil
}

// CreatePlatformShortcut writes a Desktop .lnk targeting this exe; arguments open the platform page in the app.
func CreatePlatformShortcut(platformKey string) (string, error) {
	platformKey = strings.TrimSpace(platformKey)
	if platformKey == "" {
		return "", fmt.Errorf("missing platform")
	}

	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	self = filepath.Clean(self)

	outPath, err := platformSwitcherLnkPath(platformKey)
	if err != nil {
		return "", err
	}

	icon := ""
	if root, err := paths.DataRoot(); err == nil {
		cacheDir := filepath.Join(root, "IconCache")
		if err := os.MkdirAll(cacheDir, 0o755); err == nil {
			icoName := profileimage.PlatformFolder(platformKey) + "_platform.ico"
			icoPath := filepath.Join(cacheDir, icoName)
			if err := winutil.BuildPlatformIcon(platformKey, icoPath); err == nil {
				icon = icoPath + ",0"
			}
		}
	}

	workDir := filepath.Dir(self)
	desc := fmt.Sprintf("Account Switcher - %s", platformKey)
	argv := platformKey
	appID := winutil.ShortcutAppUserModelID("platform", platformKey)
	if err := winutil.WriteShortcutLnk(outPath, self, argv, workDir, desc, icon, appID); err != nil {
		return "", err
	}
	// Replace rather than duplicate: a user who had the old shortcut wants one
	// shortcut, not one under each name.
	if legacy, err := platformSwitcherLnkPathNamed(platformKey, true); err == nil && legacy != outPath {
		_ = os.Remove(legacy)
	}
	return outPath, nil
}

// DeletePlatformShortcut removes the Desktop .lnk for this platform if it exists.
func DeletePlatformShortcut(platformKey string) error {
	candidates, err := platformSwitcherLnkPaths(platformKey)
	if err != nil {
		return err
	}
	for _, p := range candidates {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
