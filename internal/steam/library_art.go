package steam

import (
	"context"
	"path/filepath"
	"strings"

	"account-switcher/internal/appclient"
	"account-switcher/internal/gameart"
	"account-switcher/internal/gamelib"
)

// applySteamArt resolves cover art for every game in one batched pass.
//
// Done after the sources have been read rather than while reading them: the
// same app turns up from several sources, and resolving art inline would repeat
// the work and, once the CDN is in play, the round trip with it.
func applySteamArt(ctx context.Context, b *gamelib.Builder, root string, accounts map[string]string, allowNetwork bool) {
	games := b.Games()
	if len(games) == 0 {
		return
	}
	id32s := steamUserID32s(root, accounts)

	reqs := make([]gameart.Request, 0, len(games))
	for _, g := range games {
		reqs = append(reqs, steamArtRequest(root, g.GameID, id32s))
	}

	online := allowNetwork && !appclient.IsOfflineMode()
	for appID, res := range gameart.ResolveMany(ctx, appclient.Shared, reqs, online) {
		b.Observe(gamelib.Observation{
			PlatformKey: PlatformKey,
			GameID:      appID,
			ArtURL:      res.PublicURL,
			Source:      gamelib.SourceSteamAppList,
		})
	}
}

// steamArtRequest builds the candidate chain for one Steam app.
//
// Steam keeps several artwork layouts side by side, because it has changed the
// scheme twice and never removed the old files, and it also lets the user
// replace any of it from the library page. Both matter: the old layout is all
// there is for a game installed years ago, and the user's own grid image is the
// one they expect to see.
func steamArtRequest(root, appID string, userID32s []string) gameart.Request {
	librarycache := filepath.Join(root, "appcache", "librarycache")
	return gameart.Request{
		PlatformKey: PlatformKey,
		GameID:      appID,
		UserFiles:   steamGridOverrides(root, appID, userID32s),
		LocalFiles:  steamLocalArtFiles(librarycache, appID),
		RemoteURLs:  steamRemoteArtURLs(appID),
	}
}

// steamGridOverrides are the images the user set themselves from the library.
//
// Steam writes them per account under config/grid, named by app id with a
// suffix for the shape. "p" is the portrait capsule the library grid shows,
// which is the shape the games view uses, so it is preferred over the wide and
// hero variants.
func steamGridOverrides(root, appID string, userID32s []string) []string {
	var out []string
	for _, id32 := range userID32s {
		grid := filepath.Join(root, "userdata", id32, "config", "grid")
		for _, suffix := range []string{"p", ""} {
			for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp"} {
				out = append(out, filepath.Join(grid, appID+suffix+ext))
			}
		}
	}
	return out
}

// steamLocalArtFiles are the capsules Steam has already cached on this machine,
// best shape first.
//
// The newer client stores artwork in a per-app folder; the older one used a
// flat directory with the app id in the filename. Both are checked because an
// upgraded install keeps whichever it wrote at the time.
func steamLocalArtFiles(librarycache, appID string) []string {
	// Portrait shapes first: the grid renders 2:3 tiles, so a wide header has
	// to be cropped and loses the title art it was designed around.
	names := []string{
		"library_600x900_2x.jpg",
		"library_600x900.jpg",
		"portrait.png",
		"library_capsule.jpg",
		"capsule_616x353.jpg",
		"header.jpg",
		"library_hero.jpg",
		"logo.png",
	}
	var out []string
	for _, n := range names {
		out = append(out, filepath.Join(librarycache, appID, n))
	}
	for _, n := range names {
		out = append(out, filepath.Join(librarycache, appID+"_"+n))
	}
	return out
}

// steamRemoteArtURLs are Steam's public CDN paths for an app's store artwork.
//
// These need no key and no session: the store serves the same files to a logged
// out browser. They cover every game the client has not cached locally, which
// on a fresh install or for a never-launched game is most of the library.
func steamRemoteArtURLs(appID string) []string {
	const (
		storeAssets = "https://shared.cloudflare.steamstatic.com/store_item_assets/steam/apps/"
		legacyCDN   = "https://cdn.cloudflare.steamstatic.com/steam/apps/"
	)
	var out []string
	// The store_item_assets host is the current one and the only place the
	// newer portrait capsules exist; the legacy path still answers for older
	// apps that were never re-published.
	for _, base := range []string{storeAssets, legacyCDN} {
		out = append(out,
			base+appID+"/library_600x900_2x.jpg",
			base+appID+"/library_600x900.jpg",
			base+appID+"/header.jpg",
		)
	}
	return out
}

// steamUserID32s lists the account folders under userdata, which is where the
// per-account grid overrides live.
func steamUserID32s(root string, accounts map[string]string) []string {
	var out []string
	seen := map[string]struct{}{}
	for id64 := range accounts {
		f, err := FormatsFromID64(id64)
		if err != nil || strings.TrimSpace(f.ID32) == "" {
			continue
		}
		if _, done := seen[f.ID32]; done {
			continue
		}
		seen[f.ID32] = struct{}{}
		out = append(out, f.ID32)
	}
	return out
}
