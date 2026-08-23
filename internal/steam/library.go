package steam

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"account-switcher/internal/appclient"
	"account-switcher/internal/gamelib"

	"github.com/Jleagle/steam-go/steamvdf"
)

// GameResolver returns the Steam library resolver for registration with
// [gamelib.Register].
func GameResolver() gamelib.Resolver {
	return gamelib.ResolverFunc{Key: PlatformKey, Fn: resolveSteamLibrary}
}

// resolveSteamLibrary reads every local record Steam keeps about who owns what,
// strongest source first, and optionally tops it up from the public profile.
//
// Steam spreads ownership over five places and none of them is complete on its
// own: appmanifest names the installing account but only for installed games,
// localconfig has playtime but only for games launched by that account here,
// sharedconfig has the account's categories including for games it never ran,
// and the userdata folders are leftovers that outlive an uninstall. Reading all
// of them and grading each claim is the only way to get both coverage and a
// trustworthy owner.
func resolveSteamLibrary(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
	root, err := steamInstallRoot()
	if err != nil {
		return gamelib.Result{PlatformKey: PlatformKey}, err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return gamelib.Result{
			PlatformKey: PlatformKey,
			Warnings:    []string{"Steam install folder is not set"},
		}, nil
	}
	return resolveSteamLibraryAt(ctx, root, opts)
}

// resolveSteamLibraryAt does the work against an explicit Steam root, so the
// sources can be exercised against a fixture tree.
func resolveSteamLibraryAt(ctx context.Context, root string, opts gamelib.Options) (gamelib.Result, error) {
	res := gamelib.Result{PlatformKey: PlatformKey}

	accounts := steamLibraryAccounts(root, opts)
	b := gamelib.NewBuilder()

	// Source 1, the firm one: every appmanifest on disk, with LastOwner naming
	// the account that installed the game.
	manifests := readAllAppManifests(root)
	librarycache := filepath.Join(root, "appcache", "librarycache")
	for _, m := range manifests {
		if _, skip := steamInfraAppIDs[m.AppID]; skip {
			continue
		}
		obs := gamelib.Observation{
			PlatformKey: PlatformKey,
			GameID:      m.AppID,
			Name:        m.Name,
			ArtURL:      copyGameIcon(librarycache, m.AppID),
			Installed:   true,
			InstallPath: m.InstallPath,
			SizeOnDisk:  m.SizeOnDisk,
			Source:      gamelib.SourceSteamAppManifest,
			LastPlayed:  m.LastPlayed,
		}
		// LastOwner is 0 on manifests Steam wrote before it tracked the owner,
		// and on games installed by an account that has since been removed.
		if owner := normalizeSteamID64(m.LastOwner); owner != "" {
			obs.AccountID = owner
			obs.AccountName = accounts[owner]
			obs.Confidence = gamelib.ConfidenceExact
			obs.InstalledBy = true
		} else if id, name, ok := opts.SingleKnownAccount(); ok {
			// Only one account has ever logged in here, so it installed this.
			obs.AccountID = id
			obs.AccountName = name
			obs.Confidence = gamelib.ConfidenceInferred
			obs.InstalledBy = true
		}
		b.Observe(obs)
	}

	// Sources 2 to 4: the per-account records under userdata.
	for id64 := range accounts {
		f, err := FormatsFromID64(id64)
		if err != nil {
			continue
		}
		userdata := filepath.Join(root, "userdata", f.ID32)
		observeLocalConfigApps(b, userdata, id64, accounts[id64])
		observeSharedConfigApps(b, userdata, id64, accounts[id64])
		observeUserdataFolders(b, userdata, id64, accounts[id64])
	}

	// Source 5, optional: the public profile, the only view of games owned but
	// never installed on this machine.
	if opts.AllowNetwork && !appclient.IsOfflineMode() {
		for id64, name := range accounts {
			games, err := fetchCommunityGames(ctx, id64)
			if err != nil {
				steamLog.Debug("steam community game list unavailable",
					slog.String("steamId64", id64), slog.Any("err", err))
				continue
			}
			for _, g := range games {
				b.Observe(gamelib.Observation{
					PlatformKey:     PlatformKey,
					GameID:          g.AppID,
					Name:            g.Name,
					AccountID:       id64,
					AccountName:     name,
					Source:          gamelib.SourceSteamCommunityXML,
					Confidence:      gamelib.ConfidenceExact,
					PlaytimeMinutes: g.PlaytimeMinutes,
				})
			}
		}
	}

	// Names last, so any source that knows the real name has already set one and
	// the catalogue only fills the gaps.
	applyCatalogueNames(ctx, b, manifests, librarycache, opts.AllowNetwork)

	res.Games = dropNamelessSteamGames(b.Games())
	return res, nil
}

// steamLibraryAccounts maps SteamID64 to a display name for every account this
// machine has logged in, falling back to the accounts the switcher knows when
// loginusers.vdf is unreadable.
func steamLibraryAccounts(root string, opts gamelib.Options) map[string]string {
	out := map[string]string{}
	users, err := ParseLoginUsers(LoginUsersPath(root))
	if err == nil {
		for _, u := range users {
			id := normalizeSteamID64(u.SteamID64)
			if id == "" {
				continue
			}
			name := strings.TrimSpace(u.PersonaName)
			if name == "" {
				name = strings.TrimSpace(u.AccountName)
			}
			out[id] = name
		}
	}
	for id, name := range opts.KnownAccounts {
		id = normalizeSteamID64(id)
		if id == "" {
			continue
		}
		if existing := strings.TrimSpace(out[id]); existing == "" {
			out[id] = name
		}
	}
	return out
}

func normalizeSteamID64(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 17 || !isAllDigitRunes(s) {
		return ""
	}
	if s == "0" {
		return ""
	}
	return s
}

// appManifest is the subset of appmanifest_<id>.acf worth reading.
type appManifest struct {
	AppID       string
	Name        string
	LastOwner   string
	InstallPath string
	SizeOnDisk  int64
	LastPlayed  time.Time
}

// readAllAppManifests parses every appmanifest across every Steam library
// folder. Parsing beats the old regex scan because LastOwner and SizeOnDisk sit
// beside nested sections whose keys repeat, and a regex picks the wrong one.
func readAllAppManifests(root string) []appManifest {
	dirs, err := steamAppsDirs(root)
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []appManifest
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
			if _, done := seen[appID]; done {
				continue
			}
			man, ok := parseAppManifest(filepath.Join(dir, e.Name()), dir)
			if !ok {
				continue
			}
			seen[appID] = struct{}{}
			out = append(out, man)
		}
	}
	return out
}

func parseAppManifest(path, steamappsDir string) (appManifest, bool) {
	kv, err := readVDFFile(path)
	if err != nil {
		return appManifest{}, false
	}
	// Steam writes the file with a single AppState root, but a manifest half
	// written by an interrupted update can lose it.
	state := kv
	if child, ok := childCI(kv, "AppState"); ok {
		state = child
	}

	appID := strings.TrimSpace(childValueCI(state, "appid"))
	if appID == "" || !isAllDigitRunes(appID) {
		return appManifest{}, false
	}
	man := appManifest{
		AppID:      appID,
		Name:       strings.TrimSpace(childValueCI(state, "name")),
		LastOwner:  strings.TrimSpace(childValueCI(state, "LastOwner")),
		SizeOnDisk: parseInt64(childValueCI(state, "SizeOnDisk")),
	}
	if dir := strings.TrimSpace(childValueCI(state, "installdir")); dir != "" {
		man.InstallPath = filepath.Join(steamappsDir, "common", dir)
	}
	if ts := parseUnixSeconds(childValueCI(state, "LastPlayed")); !ts.IsZero() {
		man.LastPlayed = ts
	}
	return man, true
}

// steamUserAppsPath walks the wrapper keys Steam nests its per-app data under.
// The store name differs between localconfig and sharedconfig, so the caller
// passes the root it expects and the rest of the path is shared.
func steamUserAppsPath(kv steamvdf.KeyValue, storeKey string) (steamvdf.KeyValue, bool) {
	node := kv
	if child, ok := childCI(node, storeKey); ok {
		node = child
	}
	for _, key := range []string{"Software", "Valve", "Steam", "apps"} {
		child, ok := childCI(node, key)
		if !ok {
			return steamvdf.KeyValue{}, false
		}
		node = child
	}
	return node, true
}

// observeLocalConfigApps reads the account's own play records: every app it has
// launched on this machine, with playtime and a last-played stamp.
func observeLocalConfigApps(b *gamelib.Builder, userdata, id64, name string) {
	kv, err := readVDFFile(filepath.Join(userdata, "config", "localconfig.vdf"))
	if err != nil {
		return
	}
	apps, ok := steamUserAppsPath(kv, "UserLocalConfigStore")
	if !ok {
		return
	}
	for _, app := range apps.Children {
		appID := strings.TrimSpace(app.Key)
		if appID == "" || !isAllDigitRunes(appID) {
			continue
		}
		if _, skip := steamInfraAppIDs[appID]; skip {
			continue
		}
		b.Observe(gamelib.Observation{
			PlatformKey:     PlatformKey,
			GameID:          appID,
			AccountID:       id64,
			AccountName:     name,
			Source:          gamelib.SourceSteamLocalConfig,
			Confidence:      gamelib.ConfidenceStrong,
			PlaytimeMinutes: parseInt64(childValueCI(app, "Playtime")),
			LastPlayed:      parseUnixSeconds(childValueCI(app, "LastPlayed")),
		})
	}
}

// observeSharedConfigApps reads the account's roaming library settings. An app
// appears here once the account has categorised, favourited, or hidden it,
// which happens for games it owns but has never installed on this machine.
func observeSharedConfigApps(b *gamelib.Builder, userdata, id64, name string) {
	kv, err := readVDFFile(filepath.Join(userdata, "7", "remote", "sharedconfig.vdf"))
	if err != nil {
		return
	}
	apps, ok := steamUserAppsPath(kv, "UserRoamingConfigStore")
	if !ok {
		return
	}
	for _, app := range apps.Children {
		appID := strings.TrimSpace(app.Key)
		if appID == "" || !isAllDigitRunes(appID) {
			continue
		}
		if _, skip := steamInfraAppIDs[appID]; skip {
			continue
		}
		b.Observe(gamelib.Observation{
			PlatformKey: PlatformKey,
			GameID:      appID,
			AccountID:   id64,
			AccountName: name,
			Source:      gamelib.SourceSteamSharedConfig,
			Confidence:  gamelib.ConfidenceStrong,
		})
	}
}

// observeUserdataFolders is the old signal, kept as the weakest source: a bare
// numeric folder under the account's userdata. It survives an uninstall and it
// is created for things that are not games, so on its own it proves only that
// the account ran the app here once.
func observeUserdataFolders(b *gamelib.Builder, userdata, id64, name string) {
	for _, appID := range listNumericSubdirNames(userdata) {
		if _, skip := steamInfraAppIDs[appID]; skip {
			continue
		}
		b.Observe(gamelib.Observation{
			PlatformKey: PlatformKey,
			GameID:      appID,
			AccountID:   id64,
			AccountName: name,
			Source:      gamelib.SourceSteamUserdata,
			Confidence:  gamelib.ConfidenceWeak,
		})
	}
}

// applyCatalogueNames fills names and art for everything still unnamed after
// the per-account pass, using the downloaded app catalogue first and the names
// Steam itself wrote into the manifests as the offline fallback.
func applyCatalogueNames(ctx context.Context, b *gamelib.Builder, manifests []appManifest, librarycache string, allowNetwork bool) {
	catalogue, err := getSteamAppNameMapCached()
	if err != nil {
		// A missing catalogue is normal on a first run. Downloading it is a
		// blocking fetch, so it only happens on a pass that already accepted
		// network cost; the manifests still name everything installed either
		// way, which is what the games view most needs to label.
		if !allowNetwork {
			catalogue = map[string]string{}
		} else if catalogue, err = ensureAppNameMap(ctx); err != nil {
			catalogue = map[string]string{}
		}
	}
	manifestNames := make(map[string]string, len(manifests))
	for _, m := range manifests {
		if m.Name != "" {
			manifestNames[m.AppID] = m.Name
		}
	}

	for _, g := range b.Games() {
		name := strings.TrimSpace(catalogue[g.GameID])
		if name == "" {
			name = strings.TrimSpace(manifestNames[g.GameID])
		}
		art := g.ArtURL
		if art == "" {
			art = copyGameIcon(librarycache, g.GameID)
		}
		if name == "" && art == "" {
			continue
		}
		b.Observe(gamelib.Observation{
			PlatformKey: PlatformKey,
			GameID:      g.GameID,
			Name:        name,
			ArtURL:      art,
			Source:      gamelib.SourceSteamAppList,
		})
	}
}

// dropNamelessSteamGames removes entries that resolved to nothing but a number
// on nothing but a leftover folder.
//
// The userdata folders collect directories for tools, redistributables, and
// delisted apps that no catalogue can name, and a grid of "App 1826330" tiles
// helps nobody. But an app the account categorised or actually played is a real
// library entry, so a strong claim keeps it even while the catalogue is missing
// its name, which is the normal state on a first run and in offline mode.
func dropNamelessSteamGames(games []gamelib.Game) []gamelib.Game {
	out := games[:0]
	for _, g := range games {
		if g.Name == g.GameID && !g.Installed && !hasStrongOwner(g) {
			continue
		}
		if g.Name == "" || g.Name == g.GameID {
			g.Name = "App " + g.GameID
		}
		out = append(out, g)
	}
	return out
}

// hasStrongOwner reports whether any account's claim on the game came from
// something better than a leftover folder.
func hasStrongOwner(g gamelib.Game) bool {
	for _, o := range g.Owners {
		switch o.Confidence {
		case gamelib.ConfidenceStrong.String(), gamelib.ConfidenceExact.String():
			return true
		}
	}
	return false
}

func childCI(kv steamvdf.KeyValue, key string) (steamvdf.KeyValue, bool) {
	for _, ch := range kv.Children {
		if strings.EqualFold(ch.Key, key) {
			return ch, true
		}
	}
	return steamvdf.KeyValue{}, false
}

func childValueCI(kv steamvdf.KeyValue, key string) string {
	ch, ok := childCI(kv, key)
	if !ok {
		return ""
	}
	return ch.Value
}

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// parseUnixSeconds reads a Steam timestamp. Steam writes 0 for "never", which
// would otherwise become 1970 and sort ahead of every real date.
func parseUnixSeconds(s string) time.Time {
	n := parseInt64(s)
	if n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

// ResolveLibrary exposes the resolver for callers that hold a service, so the
// games view can refresh without reaching into the registry.
func (s *SteamService) ResolveLibrary(ctx context.Context, allowNetwork bool) (gamelib.Result, error) {
	root, err := s.steamInstallRoot()
	if err != nil {
		return gamelib.Result{PlatformKey: PlatformKey}, fmt.Errorf("steam install root: %w", err)
	}
	if strings.TrimSpace(root) == "" {
		return gamelib.Result{PlatformKey: PlatformKey, Games: []gamelib.Game{}}, nil
	}
	return gamelib.ResolvePlatform(ctx, PlatformKey, allowNetwork)
}
