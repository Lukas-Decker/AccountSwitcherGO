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
func applySteamArt(ctx context.Context, b *gamelib.Builder, root string, accounts map[string]string, allowArtwork, cacheOnly bool) {
	games := b.Games()
	if len(games) == 0 {
		return
	}
	if cacheOnly {
		for _, g := range games {
			if res := gameart.Cached(PlatformKey, g.GameID); res.PublicURL != "" {
				b.Observe(gamelib.Observation{
					PlatformKey: PlatformKey,
					GameID:      g.GameID,
					ArtURL:      res.PublicURL,
					Source:      gamelib.SourceSteamAppList,
				})
			}
		}
		return
	}

	id32s := steamUserID32s(root, accounts)

	reqs := make([]gameart.Request, 0, len(games))
	for _, g := range games {
		reqs = append(reqs, steamArtRequest(root, g.GameID, g.Name, id32s))
	}

	online := allowArtwork && !appclient.IsOfflineMode()
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
// Local and remote candidates are mixed together and ranked by shape, not by
// where they live. That matters: Steam caches a bare logo.png for plenty of
// apps whose portrait capsule it never downloaded, and taking the local file
// just because it is local puts a transparent wordmark on the tile when the
// public CDN has the actual cover.
func steamArtRequest(root, appID, name string, userID32s []string) gameart.Request {
	librarycache := filepath.Join(root, "appcache", "librarycache")

	var candidates []gameart.Candidate
	candidates = append(candidates, steamGridOverrides(root, appID, userID32s)...)
	candidates = append(candidates, steamLocalArt(librarycache, appID)...)
	candidates = append(candidates, steamRemoteArt(appID)...)

	return gameart.Request{
		PlatformKey: PlatformKey,
		GameID:      appID,
		Candidates:  candidates,
		// Reached only when Steam's own cache and its store CDN both come up
		// empty, which on a real library is a few hundred apps out of several
		// thousand rather than every one of them.
		Archive: func(ctx context.Context) []gameart.Candidate {
			return gameart.ArchiveCandidates(ctx, appclient.Shared, gameart.ArchiveRef{
				SteamAppID: appID,
				Name:       name,
			})
		},
	}
}

// steamGridOverrides are the images the user set themselves from the library.
//
// Steam writes them per account under config/grid, named by app id with a
// suffix for the shape. "p" is the portrait capsule the library grid shows,
// which is the shape the games view uses, so it is preferred over the wide and
// hero variants.
func steamGridOverrides(root, appID string, userID32s []string) []gameart.Candidate {
	var out []gameart.Candidate
	for _, id32 := range userID32s {
		grid := filepath.Join(root, "userdata", id32, "config", "grid")
		for _, suffix := range []string{"p", ""} {
			for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp"} {
				out = append(out, gameart.LocalFile(gameart.TierUserPicked,
					filepath.Join(grid, appID+suffix+ext)))
			}
		}
	}
	return out
}

// steamLocalArt is what Steam has already cached on this machine.
//
// The names are the ones a current client actually writes, checked against a
// real 4500-app cache: library_600x900.jpg for the portrait, header.jpg and
// library_hero.jpg for the wide shapes, logo.png for the wordmark. Names from
// older asset schemes are not listed, because every extra name costs a stat per
// game and none of them turned up.
func steamLocalArt(librarycache, appID string) []gameart.Candidate {
	dir := filepath.Join(librarycache, appID)
	join := func(name string) string { return filepath.Join(dir, name) }

	var out []gameart.Candidate
	out = append(out, gameart.LocalFiles(gameart.TierPortrait,
		join("library_600x900.jpg"),
	)...)
	out = append(out, gameart.LocalFiles(gameart.TierWide,
		join("library_header.jpg"),
		join("header.jpg"),
		join("library_hero.jpg"),
	)...)
	out = append(out, gameart.LocalFiles(gameart.TierLogo,
		join("logo.png"),
	)...)

	// The flat layout an older client used, kept as a cheap safety net for an
	// install that has not rewritten its cache.
	flat := func(name string) string { return filepath.Join(librarycache, appID+"_"+name) }
	out = append(out, gameart.LocalFile(gameart.TierPortrait, flat("library_600x900.jpg")))
	out = append(out, gameart.LocalFile(gameart.TierWide, flat("header.jpg")))
	return out
}

// steamRemoteArt is Steam's public store artwork.
//
// No key and no session: the store serves these to a logged out browser. They
// cover every game the client never cached locally, which on a fresh install or
// for a never-launched game is most of the library.
//
// Both hosts are listed because they are not interchangeable. The
// store_item_assets path is the current scheme, and shared.steamstatic.com is
// where its cloudflare alias redirects to, so it is used directly rather than
// paying a redirect on every request. The older /steam/apps/ path still answers
// for apps that were never republished under the new one.
func steamRemoteArt(appID string) []gameart.Candidate {
	const (
		storeAssets = "https://shared.steamstatic.com/store_item_assets/steam/apps/"
		legacyCDN   = "https://cdn.cloudflare.steamstatic.com/steam/apps/"
	)
	var out []gameart.Candidate
	for _, base := range []string{legacyCDN, storeAssets} {
		out = append(out, gameart.RemoteURL(gameart.TierPortrait, base+appID+"/library_600x900.jpg"))
	}
	for _, base := range []string{legacyCDN, storeAssets} {
		out = append(out,
			gameart.RemoteURL(gameart.TierWide, base+appID+"/header.jpg"),
			gameart.RemoteURL(gameart.TierWide, base+appID+"/library_hero.jpg"),
		)
	}
	for _, base := range []string{legacyCDN, storeAssets} {
		out = append(out, gameart.RemoteURL(gameart.TierLogo, base+appID+"/logo.png"))
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
