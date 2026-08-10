//go:build windows

package winutil

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

// LocateExe hunts for a platform's executable in the places Windows records an
// install, so the user is only asked to point at it by hand as a last resort.
//
// The order is cheapest and most authoritative first: the registry knows where
// an installer put things, replaying the known default layout across the other
// drives costs a handful of stat calls, and only then is anything walked.
func LocateExe(exeName string, defaultPaths []string, budget time.Duration) (string, bool) {
	exeName = strings.TrimSpace(exeName)
	if exeName == "" {
		return "", false
	}
	deadline := time.Now().Add(budget)

	if p, ok := findExeViaAppPaths(exeName); ok {
		return p, true
	}
	if p, ok := findExeViaUninstallKeys(exeName); ok {
		return p, true
	}
	if p, ok := findExeViaDefaultLayouts(exeName, defaultPaths); ok {
		return p, true
	}
	return findExeByBoundedScan(exeName, deadline)
}

// findExeViaAppPaths reads the App Paths key, which is how an installer tells
// Windows where its executable lives so "run" can find it by bare name.
func findExeViaAppPaths(exeName string) (string, bool) {
	sub := `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\` + exeName
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, access := range []uint32{registry.QUERY_VALUE, registry.QUERY_VALUE | registry.WOW64_32KEY} {
			k, err := registry.OpenKey(root, sub, access)
			if err != nil {
				continue
			}
			val, _, err := k.GetStringValue("")
			k.Close()
			if err != nil {
				continue
			}
			if p, ok := validExePath(strings.Trim(strings.TrimSpace(val), `"`)); ok {
				return p, true
			}
		}
	}
	return "", false
}

// findExeViaUninstallKeys walks the uninstall entries and looks for the exe
// under whatever folder each one claims to have installed into.
func findExeViaUninstallKeys(exeName string) (string, bool) {
	type source struct {
		root   registry.Key
		path   string
		access uint32
	}
	sources := []source{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.ENUMERATE_SUB_KEYS | registry.QUERY_VALUE},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, registry.ENUMERATE_SUB_KEYS | registry.QUERY_VALUE},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.ENUMERATE_SUB_KEYS | registry.QUERY_VALUE},
	}

	for _, src := range sources {
		k, err := registry.OpenKey(src.root, src.path, src.access)
		if err != nil {
			continue
		}
		names, err := k.ReadSubKeyNames(-1)
		k.Close()
		if err != nil {
			continue
		}
		for _, name := range names {
			sk, err := registry.OpenKey(src.root, src.path+`\`+name, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			install, _, _ := sk.GetStringValue("InstallLocation")
			icon, _, _ := sk.GetStringValue("DisplayIcon")
			sk.Close()

			if dir := strings.Trim(strings.TrimSpace(install), `"`); dir != "" {
				if p, ok := exeUnderDir(dir, exeName); ok {
					return p, true
				}
			}
			// DisplayIcon is usually the executable itself, sometimes with an
			// icon index appended.
			if ic := strings.Trim(strings.TrimSpace(icon), `"`); ic != "" {
				if idx := strings.LastIndex(ic, ","); idx > 2 {
					ic = ic[:idx]
				}
				if strings.EqualFold(filepath.Base(ic), exeName) {
					if p, ok := validExePath(ic); ok {
						return p, true
					}
				}
				if p, ok := exeUnderDir(filepath.Dir(ic), exeName); ok {
					return p, true
				}
			}
		}
	}
	return "", false
}

// findExeViaDefaultLayouts replays the known-good install layout somewhere else.
// A platform installed to D: keeps the same folder structure it would have had
// on C:, so the default path minus its drive is a strong hint.
func findExeViaDefaultLayouts(exeName string, defaultPaths []string) (string, bool) {
	roots := candidateRoots()
	for _, def := range defaultPaths {
		def = strings.TrimSpace(os.ExpandEnv(strings.ReplaceAll(def, "%", "$")))
		if def == "" {
			continue
		}
		rel := strings.TrimPrefix(def, filepath.VolumeName(def))
		rel = strings.TrimLeft(rel, `\/`)
		if rel == "" {
			continue
		}
		variants := []string{rel}
		// Also try without the leading "Program Files"-style segment, for the
		// common case of a game folder sitting at the root of another drive.
		if i := strings.IndexAny(rel, `\/`); i > 0 {
			variants = append(variants, rel[i+1:])
		}
		for _, root := range roots {
			for _, v := range variants {
				if p, ok := validExePath(filepath.Join(root, v)); ok {
					return p, true
				}
			}
		}
	}
	return "", false
}

// findExeByBoundedScan is the last resort: a shallow look through the folders
// installers actually use. It is depth and time limited because the alternative
// is walking whole drives while the user waits.
func findExeByBoundedScan(exeName string, deadline time.Time) (string, bool) {
	const maxDepth = 3
	for _, root := range candidateRoots() {
		if time.Now().After(deadline) {
			return "", false
		}
		if p, ok := scanDirForExe(root, exeName, maxDepth, deadline); ok {
			return p, true
		}
	}
	return "", false
}

func scanDirForExe(dir, exeName string, depth int, deadline time.Time) (string, bool) {
	if depth < 0 || time.Now().After(deadline) {
		return "", false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	var subdirs []string
	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			// Package caches and component stores are large and never hold a
			// platform's own executable.
			switch strings.ToLower(name) {
			case "windowsapps", "packages", "winsxs", "$recycle.bin", "system volume information", "node_modules":
				continue
			}
			subdirs = append(subdirs, filepath.Join(dir, name))
			continue
		}
		if strings.EqualFold(e.Name(), exeName) {
			if p, ok := validExePath(filepath.Join(dir, e.Name())); ok {
				return p, true
			}
		}
	}
	for _, sub := range subdirs {
		if p, ok := scanDirForExe(sub, exeName, depth-1, deadline); ok {
			return p, true
		}
	}
	return "", false
}

// candidateRoots is where installers put things: the standard program folders,
// per-user application data, and the top of every fixed drive.
func candidateRoots() []string {
	var roots []string
	seen := map[string]struct{}{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		key := strings.ToLower(filepath.Clean(p))
		if _, dup := seen[key]; dup {
			return
		}
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			return
		}
		seen[key] = struct{}{}
		roots = append(roots, p)
	}

	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "ProgramW6432", "ProgramData", "LOCALAPPDATA", "APPDATA"} {
		add(os.Getenv(env))
	}
	for drive := 'C'; drive <= 'Z'; drive++ {
		base := string(drive) + `:\`
		if st, err := os.Stat(base); err != nil || !st.IsDir() {
			continue
		}
		add(base)
		for _, sub := range []string{"Program Files", "Program Files (x86)", "Games", "SteamLibrary"} {
			add(filepath.Join(base, sub))
		}
	}
	return roots
}

func exeUnderDir(dir, exeName string) (string, bool) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", false
	}
	if p, ok := validExePath(filepath.Join(dir, exeName)); ok {
		return p, true
	}
	// Launchers often nest the executable one level in, e.g. a "Riot Client"
	// folder inside the install location the uninstaller records.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if p, ok := validExePath(filepath.Join(dir, e.Name(), exeName)); ok {
			return p, true
		}
	}
	return "", false
}

func validExePath(p string) (string, bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", false
	}
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return "", false
	}
	return p, true
}
