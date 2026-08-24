package launchers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"account-switcher/internal/gamelib"
)

// EpicPlatformKey matches the Platforms.json entry.
const EpicPlatformKey = "Epic Games"

// Epic resolves the Epic Games library.
func Epic() gamelib.Resolver {
	return gamelib.ResolverFunc{Key: EpicPlatformKey, Fn: resolveEpic}
}

// epicManifest is the subset of a .item manifest worth reading. The launcher
// writes one per installed game and keeps them machine-wide, not per account.
type epicManifest struct {
	AppName          string `json:"AppName"`
	DisplayName      string `json:"DisplayName"`
	InstallLocation  string `json:"InstallLocation"`
	InstallSize      int64  `json:"InstallSize"`
	CatalogNamespace string `json:"CatalogNamespace"`
	CatalogItemID    string `json:"CatalogItemId"`
	// LaunchExecutable is relative to InstallLocation and is the only pointer
	// Epic gives to the game's own binary, which is where its icon lives.
	LaunchExecutable string `json:"LaunchExecutable"`
	// bIsIncompleteInstall is set while a download is still running, and those
	// entries point at a folder that cannot be launched yet.
	IsIncompleteInstall bool `json:"bIsIncompleteInstall"`
}

// epicManifestDirs are the places the launcher keeps its install manifests. The
// ProgramData copy is authoritative; the per-user one exists on installs that
// were set up without administrator rights.
func epicManifestDirs() []string {
	var out []string
	if pd := programData(); pd != "" {
		out = append(out, filepath.Join(pd, "Epic", "EpicGamesLauncher", "Data", "Manifests"))
	}
	if lad := localAppData(); lad != "" {
		out = append(out, filepath.Join(lad, "EpicGamesLauncher", "Saved", "Manifests"))
	}
	return out
}

// resolveEpic reads the install manifests.
//
// Epic keeps no per-account library on disk: the manifests are machine-wide and
// the owned catalogue only exists behind an authenticated web call. So this
// resolves every installed game exactly and leaves ownership to the inference
// rule, which declines to guess once there is more than one Epic account.
func resolveEpic(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
	res := gamelib.Result{PlatformKey: EpicPlatformKey}
	b := gamelib.NewBuilder()
	var art []artSource

	found := false
	for _, dir := range epicManifestDirs() {
		if !dirExists(dir) {
			continue
		}
		found = true
		ents, err := os.ReadDir(dir)
		if err != nil {
			res.Warnings = append(res.Warnings, "Epic manifests unreadable: "+err.Error())
			continue
		}
		for _, e := range ents {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".item") {
				continue
			}
			man, ok := readEpicManifest(filepath.Join(dir, e.Name()))
			if !ok {
				continue
			}
			obs := gamelib.Observation{
				PlatformKey: EpicPlatformKey,
				GameID:      man.AppName,
				Name:        strings.TrimSpace(man.DisplayName),
				Installed:   !man.IsIncompleteInstall,
				InstallPath: strings.TrimSpace(man.InstallLocation),
				SizeOnDisk:  man.InstallSize,
				Source:      gamelib.SourceEpicManifest,
			}
			if obs.Name == "" {
				obs.Name = dirNameTitle(obs.InstallPath)
			}
			attributeInstall(&obs, opts)
			b.Observe(obs)

			// Epic ships no artwork of its own, so the game's executable icon
			// is the only image on this machine that belongs to it.
			art = append(art, artSource{
				gameID: man.AppName,
				local:  installDirIcons(obs.InstallPath),
				exe:    exeForIcon(obs.InstallPath, man.LaunchExecutable),
			})
		}
	}

	if !found {
		res.Unsupported = true
		return res, nil
	}
	if w := ambiguousOwnerWarning("Epic Games", opts); w != "" {
		res.Warnings = append(res.Warnings, w)
	}
	applyLauncherArt(ctx, b, EpicPlatformKey, art, gamelib.SourceEpicManifest, opts.AllowNetwork)
	res.Games = b.Games()
	return res, nil
}

func readEpicManifest(path string) (epicManifest, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return epicManifest{}, false
	}
	var man epicManifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return epicManifest{}, false
	}
	// AppName is the launcher's own id for the game and the only stable key;
	// a manifest without one cannot be addressed or launched.
	if strings.TrimSpace(man.AppName) == "" {
		return epicManifest{}, false
	}
	return man, true
}
