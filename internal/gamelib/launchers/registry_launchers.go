package launchers

import (
	"context"
	"fmt"
	"strings"

	"account-switcher/internal/gamelib"
	"account-switcher/internal/winutil"
)

// Platform keys for the launchers whose libraries live in the registry.
const (
	UbisoftPlatformKey  = "Ubisoft"
	RockstarPlatformKey = "Rockstar"
	EAPlatformKey       = "EA Desktop"
	OriginPlatformKey   = "Origin"
)

// regInstallScan describes a launcher that records one registry subkey per
// installed game. The shape is identical across Ubisoft, Rockstar, and EA: a
// parent key, one child per game, and a value naming the install folder.
type regInstallScan struct {
	// parents are tried in order; the WOW6432Node copy is where 32-bit
	// launchers write on a 64-bit Windows, which is nearly all of them.
	parents []string
	// installValues are the value names that may hold the install folder,
	// since these launchers have renamed it across versions.
	installValues []string
	// nameValues optionally hold a display name; most of these launchers do not
	// store one and the folder name has to stand in.
	nameValues []string
	source     gamelib.Source
	// uninstallKeyFmt builds the uninstall entry for a game id, for launchers
	// whose install key holds no icon of its own. %s takes the game id.
	uninstallKeyFmt []string
}

var ubisoftScan = regInstallScan{
	parents: []string{
		`HKLM\SOFTWARE\WOW6432Node\Ubisoft\Launcher\Installs`,
		`HKLM\SOFTWARE\Ubisoft\Launcher\Installs`,
	},
	installValues: []string{"InstallDir"},
	source:        gamelib.SourceUbisoftReg,
	// The Installs key holds only a path, but the installer also writes a
	// normal uninstall entry keyed by the same product id, and that one carries
	// the game's icon.
	uninstallKeyFmt: []string{
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\Uplay Install %s`,
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Uplay Install %s`,
	},
}

var rockstarScan = regInstallScan{
	parents: []string{
		`HKLM\SOFTWARE\WOW6432Node\Rockstar Games`,
		`HKLM\SOFTWARE\Rockstar Games`,
	},
	installValues: []string{"InstallFolder", "InstallLocation"},
	nameValues:    []string{"DisplayName"},
	source:        gamelib.SourceRockstarReg,
}

var eaScan = regInstallScan{
	parents: []string{
		`HKLM\SOFTWARE\WOW6432Node\EA Games`,
		`HKLM\SOFTWARE\EA Games`,
		`HKLM\SOFTWARE\WOW6432Node\Electronic Arts`,
		`HKLM\SOFTWARE\Electronic Arts`,
	},
	installValues: []string{"Install Dir", "InstallDir", "Install Location"},
	nameValues:    []string{"DisplayName", "Name"},
	source:        gamelib.SourceEAInstallReg,
}

// Ubisoft resolves Ubisoft Connect installs. The launcher keys games by their
// numeric Ubisoft product id and stores no name, so names come from the folder.
func Ubisoft() gamelib.Resolver {
	return gamelib.ResolverFunc{
		Key: UbisoftPlatformKey,
		Fn: func(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
			return resolveRegInstalls(ctx, UbisoftPlatformKey, "Ubisoft Connect", ubisoftScan, opts)
		},
	}
}

// Rockstar resolves Rockstar Games Launcher installs.
func Rockstar() gamelib.Resolver {
	return gamelib.ResolverFunc{
		Key: RockstarPlatformKey,
		Fn: func(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
			return resolveRegInstalls(ctx, RockstarPlatformKey, "the Rockstar launcher", rockstarScan, opts)
		},
	}
}

// EADesktop resolves EA app and Origin installs. Both write the same registry
// layout, and a machine that has migrated from one to the other keeps entries
// from both, so they are read together and deduplicated by game key.
func EADesktop() gamelib.Resolver {
	return gamelib.ResolverFunc{
		Key: EAPlatformKey,
		Fn: func(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
			res, err := resolveRegInstalls(ctx, EAPlatformKey, "the EA app", eaScan, opts)
			if err != nil {
				return res, err
			}
			origin := resolveOriginManifests(ctx, EAPlatformKey, opts)
			if len(origin) > 0 {
				res.Unsupported = false
				res.Games = gamelib.Merge(res.Games, origin)
			}
			return res, nil
		},
	}
}

// Origin resolves the standalone Origin platform entry, which the switcher
// still lists separately from the EA app.
func Origin() gamelib.Resolver {
	return gamelib.ResolverFunc{
		Key: OriginPlatformKey,
		Fn: func(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
			res := gamelib.Result{PlatformKey: OriginPlatformKey}
			res.Games = resolveOriginManifests(ctx, OriginPlatformKey, opts)
			if len(res.Games) == 0 {
				res.Unsupported = true
				return res, nil
			}
			if w := ambiguousOwnerWarning("Origin", opts); w != "" {
				res.Warnings = append(res.Warnings, w)
			}
			return res, nil
		},
	}
}

// resolveRegInstalls walks one launcher's install keys.
func resolveRegInstalls(ctx context.Context, platformKey, launcherName string, scan regInstallScan, opts gamelib.Options) (gamelib.Result, error) {
	res := gamelib.Result{PlatformKey: platformKey}
	b := gamelib.NewBuilder()
	var art []artSource

	found := false
	for _, parent := range scan.parents {
		names, err := winutil.RegistrySubKeyNames(parent)
		if err != nil || len(names) == 0 {
			continue
		}
		found = true
		for _, name := range names {
			keyPath := parent + `\` + name

			installPath := ""
			for _, v := range scan.installValues {
				if p := strings.TrimSpace(winutil.RegistryStringValue(keyPath, v)); p != "" {
					installPath = p
					break
				}
			}
			// A key with no install path is launcher bookkeeping, not a game.
			if installPath == "" {
				continue
			}

			display := ""
			for _, v := range scan.nameValues {
				if n := strings.TrimSpace(winutil.RegistryStringValue(keyPath, v)); n != "" {
					display = n
					break
				}
			}
			if display == "" {
				display = dirNameTitle(installPath)
			}
			if display == "" {
				display = name
			}

			obs := gamelib.Observation{
				PlatformKey: platformKey,
				GameID:      name,
				Name:        display,
				Installed:   dirExists(installPath),
				InstallPath: installPath,
				Source:      scan.source,
			}
			attributeInstall(&obs, opts)
			b.Observe(obs)

			// The launcher's own key first, then the uninstall entry it also
			// wrote, then whatever the publisher left in the install folder.
			iconFiles, iconExe := displayIconCandidates(keyPath)
			for _, format := range scan.uninstallKeyFmt {
				if iconExe != "" || len(iconFiles) > 0 {
					break
				}
				iconFiles, iconExe = displayIconCandidates(fmt.Sprintf(format, name))
			}
			if iconExe == "" {
				iconExe = exeForIcon(installPath, "")
			}
			art = append(art, artSource{
				gameID: name,
				local:  append(iconFiles, installDirIcons(installPath)...),
				exe:    iconExe,
			})
		}
	}

	if !found {
		res.Unsupported = true
		return res, nil
	}
	if w := ambiguousOwnerWarning(launcherName, opts); w != "" {
		res.Warnings = append(res.Warnings, w)
	}
	applyLauncherArt(ctx, b, platformKey, art, scan.source, opts.AllowNetwork)
	res.Games = b.Games()
	return res, nil
}
