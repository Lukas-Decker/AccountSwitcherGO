package launchers

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"account-switcher/internal/gamelib"
)

// resolveOriginManifests reads Origin's per-game .mfst files.
//
// Origin writes one folder per game under ProgramData\Origin\LocalContent, and
// inside it a .mfst holding a query string with the content id and the install
// path. It predates the EA app, so machines that upgraded still have games
// only recorded here.
func resolveOriginManifests(ctx context.Context, platformKey string, opts gamelib.Options) []gamelib.Game {
	pd := programData()
	if pd == "" {
		return nil
	}
	root := filepath.Join(pd, "Origin", "LocalContent")
	if !dirExists(root) {
		return nil
	}

	b := gamelib.NewBuilder()
	var art []artSource
	for _, gameDir := range listSubdirs(root) {
		dir := filepath.Join(root, gameDir)
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".mfst") {
				continue
			}
			id, installPath := parseOriginManifest(filepath.Join(dir, e.Name()))
			if id == "" {
				// The folder name is Origin's own label for the game and is a
				// usable key when the manifest body is a format we cannot read.
				id = gameDir
			}
			obs := gamelib.Observation{
				PlatformKey: platformKey,
				GameID:      id,
				Name:        dirNameTitle(gameDir),
				Installed:   installPath == "" || dirExists(installPath),
				InstallPath: installPath,
				Source:      gamelib.SourceOriginManifest,
			}
			attributeInstall(&obs, opts)
			b.Observe(obs)

			// Origin caches no artwork, so the game's own folder is all there
			// is. Games with no recorded path contribute nothing and are
			// skipped rather than probing a relative path.
			if installPath != "" {
				art = append(art, artSource{
					gameID: obs.GameID,
					local:  installDirIcons(installPath),
					exe:    exeForIcon(installPath, ""),
				})
			}
			break
		}
	}

	applyLauncherArt(ctx, b, platformKey, art, gamelib.SourceOriginManifest, opts.AllowNetwork)
	return b.Games()
}

// parseOriginManifest pulls the content id and install path out of a .mfst.
// The file is a single URL-encoded query string, so it parses as one.
func parseOriginManifest(path string) (id, installPath string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	body := strings.TrimSpace(string(raw))
	body = strings.TrimPrefix(body, "?")
	q, err := url.ParseQuery(body)
	if err != nil {
		return "", ""
	}
	id = strings.TrimSpace(q.Get("id"))
	if id == "" {
		id = strings.TrimSpace(q.Get("contentids"))
	}
	installPath = strings.TrimSpace(q.Get("dipinstallpath"))
	if installPath == "" {
		installPath = strings.TrimSpace(q.Get("installpath"))
	}
	return id, installPath
}
