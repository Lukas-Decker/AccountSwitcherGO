package launchers

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"account-switcher/internal/gamelib"

	"github.com/tidwall/gjson"
)

// oculusManifestDir is where the Oculus client records installed software.
func oculusManifestDir() string {
	for _, base := range []string{
		os.Getenv("OculusBase"),
		filepath.Join(os.Getenv("ProgramFiles"), "Oculus"),
		filepath.Join(os.Getenv("ProgramW6432"), "Oculus"),
	} {
		base = strings.TrimSpace(base)
		if base == "" {
			continue
		}
		dir := filepath.Join(base, "CoreData", "Manifests")
		if dirExists(dir) {
			return dir
		}
	}
	return ""
}

// OculusPlatformKey matches the Platforms.json entry.
const OculusPlatformKey = "Oculus"

// Oculus resolves installed Oculus software from the client's manifests.
func Oculus() gamelib.Resolver {
	return gamelib.ResolverFunc{Key: OculusPlatformKey, Fn: resolveOculus}
}

func resolveOculus(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
	res := gamelib.Result{PlatformKey: OculusPlatformKey}
	dir := oculusManifestDir()
	if dir == "" {
		res.Unsupported = true
		return res, nil
	}

	ents, err := os.ReadDir(dir)
	if err != nil {
		return res, err
	}
	b := gamelib.NewBuilder()
	var art []artSource
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		appID := strings.TrimSpace(gjson.GetBytes(raw, "appId").String())
		if appID == "" {
			appID = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}
		name := strings.TrimSpace(gjson.GetBytes(raw, "canonicalName").String())
		if display := strings.TrimSpace(gjson.GetBytes(raw, "displayName").String()); display != "" {
			name = display
		}
		obs := gamelib.Observation{
			PlatformKey: OculusPlatformKey,
			GameID:      appID,
			Name:        name,
			Installed:   true,
			Source:      gamelib.SourceOculusManifest,
		}
		attributeInstall(&obs, opts)
		b.Observe(obs)

		// The manifest records where the software was installed, which is the
		// only place its icon exists on this machine.
		if dir := strings.TrimSpace(gjson.GetBytes(raw, "libraryPath").String()); dir != "" {
			art = append(art, artSource{
				gameID: appID,
				local:  installDirIcons(dir),
				exe:    exeForIcon(dir, strings.TrimSpace(gjson.GetBytes(raw, "launchFile").String())),
			})
		}
	}

	applyLauncherArt(ctx, b, OculusPlatformKey, art, gamelib.SourceOculusManifest, opts.AllowNetwork)
	games := b.Games()
	if len(games) == 0 {
		res.Unsupported = true
		return res, nil
	}
	if w := ambiguousOwnerWarning("Oculus", opts); w != "" {
		res.Warnings = append(res.Warnings, w)
	}
	res.Games = games
	return res, nil
}
