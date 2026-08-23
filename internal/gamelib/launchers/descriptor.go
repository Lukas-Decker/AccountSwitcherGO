package launchers

import (
	"context"
	"strings"

	"account-switcher/internal/gamelib"
	"account-switcher/internal/platform"
)

// SingleTitle returns a resolver for a platform that is one game rather than a
// library, such as Escape from Tarkov or Albion Online.
//
// These have no library to enumerate: the platform and the game are the same
// thing, and the only question is whether it is installed. Answering that from
// the descriptor's own executable path keeps them in the games view alongside
// everything else, so an account switch can be started from the game rather
// than only from the account list.
func SingleTitle(platformKey string) gamelib.Resolver {
	return gamelib.ResolverFunc{
		Key: platformKey,
		Fn: func(_ context.Context, opts gamelib.Options) (gamelib.Result, error) {
			return resolveSingleTitle(platformKey, opts)
		},
	}
}

func resolveSingleTitle(platformKey string, opts gamelib.Options) (gamelib.Result, error) {
	res := gamelib.Result{PlatformKey: platformKey}

	exe, err := descriptorExePath(platformKey)
	if err != nil {
		return res, err
	}
	if exe == "" {
		res.Unsupported = true
		return res, nil
	}

	b := gamelib.NewBuilder()
	obs := gamelib.Observation{
		PlatformKey: platformKey,
		// The platform key is the game's identity here, and it is already
		// unique across the switcher.
		GameID:      platformKey,
		Name:        platformKey,
		Installed:   true,
		InstallPath: exe,
		Source:      gamelib.SourceDescriptorExe,
	}
	attributeInstall(&obs, opts)
	b.Observe(obs)

	// Every account on a single-title platform owns the title by definition,
	// which is the one case where full per-account resolution is free.
	for id, name := range opts.KnownAccounts {
		b.Observe(gamelib.Observation{
			PlatformKey: platformKey,
			GameID:      platformKey,
			AccountID:   id,
			AccountName: name,
			Source:      gamelib.SourceDescriptorExe,
			Confidence:  gamelib.ConfidenceStrong,
		})
	}

	res.Games = b.Games()
	return res, nil
}

// descriptorExePath resolves the platform's installed executable, preferring
// the copy the user actually has over the descriptor's default guess.
func descriptorExePath(platformKey string) (string, error) {
	exeDir, err := platform.ResolveExeDir()
	if err != nil {
		return "", err
	}
	raw, err := platform.LoadPlatformsJSON(exeDir)
	if err != nil {
		return "", err
	}
	d, err := platform.ParseDescriptor(raw, platformKey)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(d.ExeLocationDefault.FirstExistingExe()), nil
}
