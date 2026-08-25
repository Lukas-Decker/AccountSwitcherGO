package launchers

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"account-switcher/internal/appclient"
	"account-switcher/internal/gameart"
	"account-switcher/internal/gamelib"
	"account-switcher/internal/winutil"
)

// artSource is one game's art candidates, collected by a resolver as it walks
// its launcher's records.
//
// local holds files the launcher or publisher left on disk. These are icons and
// loose images rather than store capsules, so they rank as logo art: real cover
// art from a launcher that has any beats them, and they in turn beat an icon
// pulled out of an executable.
type artSource struct {
	gameID string
	// name is the game's title, used to look it up in the artwork archive.
	// These launchers ship no store art of their own, so for most of them the
	// archive is the only thing standing between the tile and an exe icon.
	name  string
	local []string
	// portrait holds remote URLs known to be cover-shaped, which outrank
	// anything local. GOG is the only launcher here that supplies them.
	portrait []string
	// remote holds other remote images of unknown or wide shape.
	remote []string
	exe    string
}

// applyLauncherArt resolves art for a batch of games and folds the results back
// into the builder.
//
// Kept separate from the resolve loop for the same reason as Steam's: art is a
// per-game lookup that says nothing about ownership, and batching it bounds how
// many downloads run at once.
func applyLauncherArt(ctx context.Context, b *gamelib.Builder, platformKey string, sources []artSource, src gamelib.Source, opts gamelib.Options) {
	if len(sources) == 0 {
		return
	}
	if opts.ArtFromCacheOnly {
		for _, s := range sources {
			if res := gameart.Cached(platformKey, s.gameID); res.PublicURL != "" {
				b.Observe(gamelib.Observation{
					PlatformKey: platformKey,
					GameID:      s.gameID,
					ArtURL:      res.PublicURL,
					Source:      src,
				})
			}
		}
		return
	}
	reqs := make([]gameart.Request, 0, len(sources))
	for _, s := range sources {
		var candidates []gameart.Candidate
		candidates = append(candidates, gameart.RemoteURLs(gameart.TierPortrait, s.portrait...)...)
		candidates = append(candidates, gameart.LocalFiles(gameart.TierLogo, s.local...)...)
		candidates = append(candidates, gameart.RemoteURLs(gameart.TierWide, s.remote...)...)
		title := s.name
		reqs = append(reqs, gameart.Request{
			PlatformKey: platformKey,
			GameID:      s.gameID,
			Candidates:  candidates,
			IconExe:     s.exe,
			Archive: func(ctx context.Context) []gameart.Candidate {
				return gameart.ArchiveCandidates(ctx, appclient.Shared, gameart.ArchiveRef{Name: title})
			},
		})
	}

	online := opts.AllowArtwork && !appclient.IsOfflineMode()
	for gameID, res := range gameart.ResolveMany(ctx, appclient.Shared, reqs, online) {
		b.Observe(gamelib.Observation{
			PlatformKey: platformKey,
			GameID:      gameID,
			ArtURL:      res.PublicURL,
			Source:      src,
		})
	}
}

// installDirIcons are image files sitting at the root of a game's install
// folder. Publishers routinely drop the game's own icon there next to the
// executable, and it is the only artwork most of these launchers ever produce.
func installDirIcons(installPath string) []string {
	installPath = strings.TrimSpace(installPath)
	if installPath == "" {
		return nil
	}
	ents, err := os.ReadDir(installPath)
	if err != nil {
		return nil
	}
	var icons []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".ico", ".png", ".jpg", ".jpeg":
			icons = append(icons, filepath.Join(installPath, e.Name()))
		}
	}
	return icons
}

// exeForIcon picks an executable to take an icon from.
//
// An explicit path from the launcher is used when there is one. Otherwise the
// install root is only used when it holds exactly one executable, because a
// folder with several is as likely to yield a crash reporter or an installer
// icon as the game's own, and a wrong icon is worse than none.
func exeForIcon(installPath, explicitExe string) string {
	if explicitExe = strings.TrimSpace(explicitExe); explicitExe != "" {
		if !filepath.IsAbs(explicitExe) && strings.TrimSpace(installPath) != "" {
			explicitExe = filepath.Join(installPath, explicitExe)
		}
		if fileExists(explicitExe) {
			return explicitExe
		}
	}
	installPath = strings.TrimSpace(installPath)
	if installPath == "" {
		return ""
	}
	ents, err := os.ReadDir(installPath)
	if err != nil {
		return ""
	}
	found := ""
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".exe") {
			continue
		}
		if found != "" {
			return ""
		}
		found = filepath.Join(installPath, e.Name())
	}
	return found
}

// registryDisplayIcon reads an uninstall entry's DisplayIcon.
//
// The value is a path optionally followed by a comma and an icon index, which
// is how Windows addresses one icon inside an executable. The index is dropped:
// the extractor takes the default icon, which is the one the shell shows.
func registryDisplayIcon(keyPath string) string {
	raw := strings.TrimSpace(winutil.RegistryStringValue(keyPath, "DisplayIcon"))
	if raw == "" {
		return ""
	}
	raw = strings.Trim(raw, `"`)
	if comma := strings.LastIndex(raw, ","); comma > 0 {
		if _, err := strconv.Atoi(strings.TrimSpace(raw[comma+1:])); err == nil {
			raw = strings.TrimSpace(raw[:comma])
		}
	}
	if !fileExists(raw) {
		return ""
	}
	return raw
}

// displayIconCandidates splits a DisplayIcon into the right bucket: an image
// file can be published directly, while an executable has to be unpacked.
func displayIconCandidates(keyPath string) (local []string, exe string) {
	icon := registryDisplayIcon(keyPath)
	if icon == "" {
		return nil, ""
	}
	switch strings.ToLower(filepath.Ext(icon)) {
	case ".ico", ".png", ".jpg", ".jpeg":
		return []string{icon}, ""
	case ".exe", ".dll":
		return nil, icon
	}
	return nil, ""
}
