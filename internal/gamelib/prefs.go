package gamelib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"account-switcher/internal/fsutil"
	"account-switcher/internal/paths"
)

// Per-game choices the user has made, kept beside the rest of the app's data
// rather than in the account settings file.
//
// These are decisions about a library view, not about an account: which games
// to stop showing, and which artwork to keep when the chain picked one the user
// did not want. Keeping them separate means a library of several thousand
// entries never bloats the settings file that every screen reads at startup.

// GamePrefs is one platform's stored choices.
type GamePrefs struct {
	// Hidden lists game ids the user has hidden. A hidden game stays resolved,
	// because unhiding has to be possible without a rescan, and is filtered out
	// of the view unless the filter is turned off.
	Hidden []string `json:"hidden,omitempty"`
	// ArtOverride pins a game to one artwork source, by the public URL that
	// source produced. Empty means the chain decides.
	ArtOverride map[string]string `json:"artOverride,omitempty"`
	// NSFWOverride records the user's own answer for a game the catalogues
	// disagree about, or never labelled. true forces it behind the filter,
	// false forces it in front.
	NSFWOverride map[string]bool `json:"nsfwOverride,omitempty"`
}

type prefsFile struct {
	Version   int                   `json:"version"`
	Platforms map[string]*GamePrefs `json:"platforms"`
}

var (
	prefsMu     sync.RWMutex
	prefsCache  *prefsFile
	prefsLoaded bool
)

func prefsPath() (string, error) {
	root, err := paths.DataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "Settings", "GamePrefs.json"), nil
}

// loadPrefsLocked reads the file once and keeps it in memory.
//
// A missing or unreadable file is an empty set rather than an error: these are
// preferences, and refusing to show a library because one of them could not be
// read would be a worse failure than forgetting a hidden game.
func loadPrefsLocked() *prefsFile {
	if prefsLoaded && prefsCache != nil {
		return prefsCache
	}
	prefsLoaded = true
	prefsCache = &prefsFile{Version: 1, Platforms: map[string]*GamePrefs{}}

	p, err := prefsPath()
	if err != nil {
		return prefsCache
	}
	raw, err := os.ReadFile(p)
	if err != nil || len(raw) == 0 {
		return prefsCache
	}
	var parsed prefsFile
	if json.Unmarshal(raw, &parsed) != nil {
		return prefsCache
	}
	if parsed.Platforms == nil {
		parsed.Platforms = map[string]*GamePrefs{}
	}
	parsed.Version = 1
	prefsCache = &parsed
	return prefsCache
}

func savePrefsLocked() error {
	p, err := prefsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(prefsCache, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(p, append(raw, '\n'), 0o644)
}

// platformPrefsLocked returns the entry for a platform, creating it.
func platformPrefsLocked(platformKey string) *GamePrefs {
	f := loadPrefsLocked()
	gp, ok := f.Platforms[platformKey]
	if !ok || gp == nil {
		gp = &GamePrefs{}
		f.Platforms[platformKey] = gp
	}
	return gp
}

// PrefsFor returns a copy of one platform's choices.
func PrefsFor(platformKey string) GamePrefs {
	prefsMu.RLock()
	defer prefsMu.RUnlock()
	f := loadPrefsLocked()
	gp, ok := f.Platforms[strings.TrimSpace(platformKey)]
	if !ok || gp == nil {
		return GamePrefs{}
	}
	out := GamePrefs{
		Hidden:       append([]string(nil), gp.Hidden...),
		ArtOverride:  map[string]string{},
		NSFWOverride: map[string]bool{},
	}
	for k, v := range gp.ArtOverride {
		out.ArtOverride[k] = v
	}
	for k, v := range gp.NSFWOverride {
		out.NSFWOverride[k] = v
	}
	return out
}

// SetHidden hides or unhides one game.
func SetHidden(platformKey, gameID string, hidden bool) error {
	platformKey, gameID = strings.TrimSpace(platformKey), strings.TrimSpace(gameID)
	if platformKey == "" || gameID == "" {
		return nil
	}
	prefsMu.Lock()
	defer prefsMu.Unlock()

	gp := platformPrefsLocked(platformKey)
	set := map[string]struct{}{}
	for _, id := range gp.Hidden {
		set[id] = struct{}{}
	}
	if hidden {
		set[gameID] = struct{}{}
	} else {
		delete(set, gameID)
	}
	gp.Hidden = gp.Hidden[:0]
	for id := range set {
		gp.Hidden = append(gp.Hidden, id)
	}
	sort.Strings(gp.Hidden)
	return savePrefsLocked()
}

// SetArtOverride pins a game's artwork to one already published URL, or clears
// the pin when url is empty.
func SetArtOverride(platformKey, gameID, url string) error {
	platformKey, gameID = strings.TrimSpace(platformKey), strings.TrimSpace(gameID)
	if platformKey == "" || gameID == "" {
		return nil
	}
	prefsMu.Lock()
	defer prefsMu.Unlock()

	gp := platformPrefsLocked(platformKey)
	if gp.ArtOverride == nil {
		gp.ArtOverride = map[string]string{}
	}
	if url = strings.TrimSpace(url); url == "" {
		delete(gp.ArtOverride, gameID)
	} else {
		gp.ArtOverride[gameID] = url
	}
	return savePrefsLocked()
}

// SetNSFWOverride records the user's own answer about a game's rating.
func SetNSFWOverride(platformKey, gameID string, nsfw *bool) error {
	platformKey, gameID = strings.TrimSpace(platformKey), strings.TrimSpace(gameID)
	if platformKey == "" || gameID == "" {
		return nil
	}
	prefsMu.Lock()
	defer prefsMu.Unlock()

	gp := platformPrefsLocked(platformKey)
	if gp.NSFWOverride == nil {
		gp.NSFWOverride = map[string]bool{}
	}
	if nsfw == nil {
		delete(gp.NSFWOverride, gameID)
	} else {
		gp.NSFWOverride[gameID] = *nsfw
	}
	return savePrefsLocked()
}

// resetPrefsForTest drops the cache so a test can point at its own data root.
func resetPrefsForTest() {
	prefsMu.Lock()
	prefsCache = nil
	prefsLoaded = false
	prefsMu.Unlock()
}
