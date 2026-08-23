package launchers

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"account-switcher/internal/gamelib"
)

// RiotPlatformKey matches the Platforms.json entry.
const RiotPlatformKey = "Riot Games"

// Riot resolves installed Riot titles.
func Riot() gamelib.Resolver {
	return gamelib.ResolverFunc{Key: RiotPlatformKey, Fn: resolveRiot}
}

// riotProducts maps the client's internal product codes to their titles. Riot
// ships a fixed, small catalogue, and the metadata folders are named by code.
var riotProducts = map[string]string{
	"league_of_legends": "League of Legends",
	"valorant":          "VALORANT",
	"bacon":             "Legends of Runeterra",
	"wildrift":          "Wild Rift",
	"riot_client":       "",
}

// resolveRiot reads the per-product metadata folders the Riot Client writes.
//
// Riot keeps one folder per installed product under ProgramData, named
// "<product>.<patchline>". Nothing there records an account, and the client
// only holds one session at a time, so ownership falls to inference.
func resolveRiot(_ context.Context, opts gamelib.Options) (gamelib.Result, error) {
	res := gamelib.Result{PlatformKey: RiotPlatformKey}
	pd := programData()
	if pd == "" {
		res.Unsupported = true
		return res, nil
	}
	metadata := filepath.Join(pd, "Riot Games", "Metadata")
	if !dirExists(metadata) {
		res.Unsupported = true
		return res, nil
	}

	b := gamelib.NewBuilder()
	for _, dir := range listSubdirs(metadata) {
		// The folder is "<product>.<patchline>", and the patchline is the
		// release channel, not a separate game.
		product := dir
		if idx := strings.Index(dir, "."); idx > 0 {
			product = dir[:idx]
		}
		product = strings.ToLower(strings.TrimSpace(product))
		if product == "" {
			continue
		}
		name, known := riotProducts[product]
		// The Riot Client itself has a metadata folder and is not a game.
		if known && name == "" {
			continue
		}
		if name == "" {
			name = dirNameTitle(product)
		}
		obs := gamelib.Observation{
			PlatformKey: RiotPlatformKey,
			GameID:      product,
			Name:        name,
			Installed:   riotProductInstalled(filepath.Join(metadata, dir)),
			Source:      gamelib.SourceRiotMetadata,
		}
		attributeInstall(&obs, opts)
		b.Observe(obs)
	}

	games := b.Games()
	if len(games) == 0 {
		res.Unsupported = true
		return res, nil
	}
	if w := ambiguousOwnerWarning("the Riot Client", opts); w != "" {
		res.Warnings = append(res.Warnings, w)
	}
	res.Games = games
	return res, nil
}

// riotProductInstalled reports whether the metadata folder describes a live
// install rather than a leftover. Riot writes the product settings file once
// the install completes and removes the folder contents on uninstall.
func riotProductInstalled(dir string) bool {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range ents {
		if !e.IsDir() && strings.Contains(strings.ToLower(e.Name()), "product_settings") {
			return true
		}
	}
	return len(ents) > 0
}
