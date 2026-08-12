package steam

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"account-switcher/internal/platform"
	"account-switcher/internal/security"
)

// OwnedGameDTO is one game in the games view, with the accounts known to have it.
//
// Ownership here is derived entirely from local Steam data: an account owns a
// game if Steam kept per-user data for that app under userdata/<id32>/<appid>,
// which it does once the account has run the game on this machine. That misses
// games owned but never launched here, and it needs no account credentials of
// any kind - the alternative would be asking Steam for each account's library,
// which requires a live access token per account.
type OwnedGameDTO struct {
	AppID     string   `json:"appId"`
	Name      string   `json:"name"`
	IconURL   string   `json:"iconUrl"`
	Owners    []string `json:"owners"`
	Installed bool     `json:"installed"`
}

// steamInfraAppIDs are Steam's own per-user data folders. They sit in userdata
// next to real games and would otherwise show up as games every account owns.
var steamInfraAppIDs = map[string]struct{}{
	"7":       {}, // Steam client configuration
	"760":     {}, // Screenshots
	"241100":  {}, // Steam Controller configs
	"250820":  {}, // SteamVR
	"228980":  {}, // Steamworks Common Redistributables
	"431960":  {}, // Wallpaper Engine service data
	"250900":  {}, // Steam Big Picture assets
	"1070560": {}, // Steam Linux Runtime
	"1391110": {}, // Steam Linux Runtime - Soldier
	"1493710": {}, // Proton Experimental
	"2180100": {}, // Steam Deck / runtime helper
	"223300":  {}, // Steam Hardware Survey
	"365670":  {}, // Blender (bundled tool entry)
	"1826330": {}, // Steam runtime helper
	"1887720": {}, // Proton runtime
	"858280":  {}, // Steam VR helper
	"1628350": {}, // Steam Linux Runtime - Sniper
	"2348590": {}, // Proton runtime
	"1245040": {}, // Proton runtime
	"996510":  {}, // Steam runtime
	"1006":    {}, // legacy client entry
}

// reACFName pulls the display name out of an appmanifest. AppState lists it
// before any nested section, so the first match is the game's own name.
var reACFName = regexp.MustCompile(`(?i)"name"\s+"([^"]*)"`)

// installedAppNames reads game names straight out of Steam's appmanifest files.
//
// The downloaded app catalogue is the better source when it is there, but it is
// a network fetch that can be empty on a first run or in offline mode, and then
// every game in the list would read "App 730". Steam already wrote the name to
// disk for anything installed.
func installedAppNames(steamRoot string) map[string]string {
	dirs, err := steamAppsDirs(steamRoot)
	if err != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, dir := range dirs {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			m := reAppManifest.FindStringSubmatch(e.Name())
			if len(m) < 2 {
				continue
			}
			appID := m[1]
			if _, done := out[appID]; done {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			nm := reACFName.FindSubmatch(data)
			if len(nm) < 2 {
				continue
			}
			if name := strings.TrimSpace(string(nm[1])); name != "" {
				out[appID] = name
			}
		}
	}
	return out
}

// gameIconCandidates are the library artwork names Steam writes, newest layout
// first. library_600x900 is the portrait capsule the store and library grid use;
// header is the older wide banner and only a fallback.
func gameIconCandidates(librarycache, appID string) []string {
	return []string{
		filepath.Join(librarycache, appID, "library_600x900.jpg"),
		filepath.Join(librarycache, appID+"_library_600x900.jpg"),
		filepath.Join(librarycache, appID, "header.jpg"),
		filepath.Join(librarycache, appID+"_header.jpg"),
		filepath.Join(librarycache, appID, "logo.png"),
	}
}

// copyGameIcon publishes Steam's local artwork for appID into wwwroot so the
// webview can load it, and returns the public path. Steam's own cache is outside
// wwwroot and the asset handler only serves from there.
//
// Nothing is downloaded: a game with no local artwork simply has no icon, and
// the view falls back to its name.
func copyGameIcon(librarycache, appID string) string {
	var src string
	for _, cand := range gameIconCandidates(librarycache, appID) {
		if st, err := os.Stat(cand); err == nil && !st.IsDir() && st.Size() > 0 {
			src = cand
			break
		}
	}
	if src == "" {
		return ""
	}

	wwwroot, err := platform.WwwrootDir()
	if err != nil {
		return ""
	}
	dstDir := filepath.Join(wwwroot, "img", "games")
	ext := strings.ToLower(filepath.Ext(src))
	if ext != ".jpg" && ext != ".png" {
		ext = ".jpg"
	}
	dst := filepath.Join(dstDir, appID+ext)
	public := "img/games/" + appID + ext

	// Reuse an already published icon unless Steam's copy moved on.
	srcInfo, err := os.Stat(src)
	if err != nil {
		return ""
	}
	if dstInfo, err := os.Stat(dst); err == nil && dstInfo.Size() == srcInfo.Size() {
		return public
	}

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return ""
	}
	in, err := os.Open(src)
	if err != nil {
		return ""
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return ""
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return ""
	}
	return public
}

// EnsureLocalGameIcon publishes Steam's own artwork for appID into wwwroot and
// returns the public path, or "" when Steam has nothing cached for it.
//
// Disk only, so unlike the games view it is safe to call from a path that must
// not block on the network.
func EnsureLocalGameIcon(appID string) string {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return ""
	}
	root, err := steamInstallRoot()
	if err != nil || strings.TrimSpace(root) == "" {
		return ""
	}
	return copyGameIcon(filepath.Join(root, "appcache", "librarycache"), appID)
}

// GetOwnedGames lists the games this machine knows about, each with the accounts
// that have played it, so the games view can switch to an account by game.
func (s *SteamService) GetOwnedGames() ([]OwnedGameDTO, error) {
	if err := security.RequireUnlocked(); err != nil {
		return nil, err
	}
	root, err := s.steamInstallRoot()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(root) == "" {
		return []OwnedGameDTO{}, nil
	}

	users, err := ParseLoginUsers(LoginUsersPath(root))
	if err != nil {
		return nil, err
	}

	// appID -> owning steamID64s, in account order so the view is stable.
	owners := map[string][]string{}
	for _, u := range users {
		id64 := strings.TrimSpace(u.SteamID64)
		if id64 == "" {
			continue
		}
		f, err := FormatsFromID64(id64)
		if err != nil {
			continue
		}
		for _, appID := range listNumericSubdirNames(filepath.Join(root, "userdata", f.ID32)) {
			if _, skip := steamInfraAppIDs[appID]; skip {
				continue
			}
			owners[appID] = append(owners[appID], id64)
		}
	}

	installed, err := installedAppIDs(root)
	if err != nil {
		installed = map[string]struct{}{}
	}
	for appID := range installed {
		if _, skip := steamInfraAppIDs[appID]; skip {
			continue
		}
		if _, seen := owners[appID]; !seen {
			owners[appID] = nil
		}
	}

	names, err := getSteamAppNameMapCached()
	if err != nil {
		names = map[string]string{}
	}
	localNames := installedAppNames(root)

	librarycache := filepath.Join(root, "appcache", "librarycache")
	out := make([]OwnedGameDTO, 0, len(owners))
	for appID, ids := range owners {
		name := strings.TrimSpace(names[appID])
		if name == "" {
			name = strings.TrimSpace(localNames[appID])
		}
		_, isInstalled := installed[appID]
		// Without a name and without a local install there is nothing to show but
		// a number, and userdata collects folders for things that are not games.
		if name == "" && !isInstalled {
			continue
		}
		if name == "" {
			name = "App " + appID
		}
		out = append(out, OwnedGameDTO{
			AppID:     appID,
			Name:      name,
			IconURL:   copyGameIcon(librarycache, appID),
			Owners:    ids,
			Installed: isInstalled,
		})
	}

	sortOwnedGames(out)
	return out, nil
}

// sortOwnedGames orders by how many accounts have the game, then by name, so the
// games most worth switching for float to the top.
func sortOwnedGames(games []OwnedGameDTO) {
	sort.SliceStable(games, func(i, j int) bool {
		if len(games[i].Owners) != len(games[j].Owners) {
			return len(games[i].Owners) > len(games[j].Owners)
		}
		return strings.ToLower(games[i].Name) < strings.ToLower(games[j].Name)
	})
}
