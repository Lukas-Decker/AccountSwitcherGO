// Package launchers resolves game libraries for the non-Steam platforms.
//
// What is reachable varies enormously between them. GOG Galaxy keeps a full
// SQLite library with the owning user on every row; the Epic launcher writes a
// manifest per installed game and records no account at all; Ubisoft and
// Rockstar only leave a registry key per install. Each resolver here reports
// what its launcher actually stores, graded honestly, rather than inventing an
// owner to fill the column.
package launchers

import (
	"os"
	"path/filepath"
	"strings"

	"account-switcher/internal/gamelib"
)

// programData resolves %ProgramData%, where the machine-wide launcher state
// lives. The environment variable is the documented way to find it and it is
// set on every supported Windows install.
func programData() string {
	if v := strings.TrimSpace(os.Getenv("ProgramData")); v != "" {
		return v
	}
	return ""
}

func localAppData() string {
	return strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
}

func roamingAppData() string {
	return strings.TrimSpace(os.Getenv("APPDATA"))
}

// dirExists reports whether path is a directory that can be listed.
func dirExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// fileExists reports whether path is a regular file with content.
func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}

// attributeInstall assigns an owner to a game whose launcher records none.
//
// The only defensible guess is a platform with exactly one known account: that
// account installed everything, because there is nobody else. With two or more
// accounts the game is left unowned so the view offers a picker, since guessing
// the account currently signed in would confidently name the wrong one for
// every game the other account installed.
func attributeInstall(obs *gamelib.Observation, opts gamelib.Options) {
	if obs.AccountID != "" {
		return
	}
	id, name, ok := opts.SingleKnownAccount()
	if !ok {
		return
	}
	obs.AccountID = id
	obs.AccountName = name
	obs.Confidence = gamelib.ConfidenceInferred
	obs.InstalledBy = true
}

// ambiguousOwnerWarning explains an empty owner column for a launcher that
// keeps no per-account records, so the view can say why rather than looking
// broken.
func ambiguousOwnerWarning(platform string, opts gamelib.Options) string {
	if len(opts.KnownAccounts) < 2 {
		return ""
	}
	return platform + " does not record which account installed a game; pick an account when launching"
}

// dirNameTitle turns an install folder into a readable name, for launchers that
// store a path and nothing else. It is a fallback, not a catalogue: the folder
// is what the publisher chose, so it is usually the game's name with the
// punctuation stripped out.
func dirNameTitle(installPath string) string {
	base := filepath.Base(strings.TrimRight(strings.TrimSpace(installPath), `\/`))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	// Folder names run words together with separators far more often than a
	// real title does, and a name is only useful if a person can read it.
	for _, sep := range []string{"_", "-"} {
		if strings.Count(base, sep) > 0 && !strings.Contains(base, " ") {
			base = strings.ReplaceAll(base, sep, " ")
		}
	}
	return strings.Join(strings.Fields(base), " ")
}

// listSubdirs returns the immediate subdirectory names of dir.
func listSubdirs(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}
