package launchers

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"account-switcher/internal/gamelib"
	"account-switcher/internal/winutil"

	"github.com/tidwall/gjson"
)

// BattleNetPlatformKey matches the Platforms.json entry.
const BattleNetPlatformKey = "BattleNet"

// BattleNet resolves the Battle.net library.
func BattleNet() gamelib.Resolver {
	return gamelib.ResolverFunc{Key: BattleNetPlatformKey, Fn: resolveBattleNet}
}

// battleNetProducts maps Blizzard's internal product codes to readable names.
//
// Battle.net.config names games only by these codes, and the agent database
// that holds the display names is a protobuf blob not worth decoding for a
// fixed, short catalogue. Unknown codes fall through as themselves rather than
// being dropped, so a newly released game still appears.
var battleNetProducts = map[string]string{
	"agent":       "Battle.net Agent",
	"bna":         "Battle.net",
	"d3":          "Diablo III",
	"d3cn":        "Diablo III (China)",
	"d4":          "Diablo IV",
	"fenris":      "Diablo IV",
	"hero":        "Heroes of the Storm",
	"hs":          "Hearthstone",
	"osi":         "Diablo II: Resurrected",
	"pro":         "Overwatch 2",
	"prometheus":  "Overwatch 2",
	"rtro":        "Blizzard Arcade Collection",
	"s1":          "StarCraft Remastered",
	"s2":          "StarCraft II",
	"viper":       "Call of Duty: Black Ops 4",
	"odin":        "Call of Duty: Modern Warfare",
	"lazr":        "Call of Duty: Modern Warfare II",
	"zeus":        "Call of Duty: Black Ops Cold War",
	"fore":        "Call of Duty: Vanguard",
	"auks":        "Call of Duty",
	"w3":          "Warcraft III: Reforged",
	"wlby":        "Crash Bandicoot 4",
	"wow":         "World of Warcraft",
	"wow_classic": "World of Warcraft Classic",
	"anbs":        "Diablo Immortal",
}

// battleNetConfigPath is the launcher's JSON settings file, which lists both
// the saved account names and every game the client knows on this machine.
func battleNetConfigPath() string {
	appdata := roamingAppData()
	if appdata == "" {
		return ""
	}
	return filepath.Join(appdata, "Battle.net", "Battle.net.config")
}

// resolveBattleNet reads the launcher config for known products and the
// registry for their install paths.
//
// Battle.net records nothing about which account installed a game, so like
// Epic this resolves installs exactly and leaves the owner to inference.
func resolveBattleNet(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
	res := gamelib.Result{PlatformKey: BattleNetPlatformKey}
	b := gamelib.NewBuilder()
	var art []artSource

	installs := battleNetRegistryInstalls()
	seen := map[string]struct{}{}

	if cfg := battleNetConfigPath(); fileExists(cfg) {
		raw, err := os.ReadFile(cfg)
		if err == nil {
			gjson.GetBytes(raw, "Games").ForEach(func(key, value gjson.Result) bool {
				code := strings.TrimSpace(key.String())
				if code == "" || code == "battle_net" {
					return true
				}
				seen[code] = struct{}{}
				installPath := installs[code]
				// ServerUid is set once the client has actually provisioned the
				// game; a bare entry is a placeholder for something never used.
				hasServer := strings.TrimSpace(value.Get("ServerUid").String()) != ""
				obs := gamelib.Observation{
					PlatformKey: BattleNetPlatformKey,
					GameID:      code,
					Name:        battleNetName(code),
					Installed:   installPath != "" && dirExists(installPath),
					InstallPath: installPath,
					Source:      gamelib.SourceBattleNetConfig,
				}
				if !obs.Installed && !hasServer {
					return true
				}
				attributeInstall(&obs, opts)
				b.Observe(obs)
				art = append(art, battleNetArt(code, installPath))
				return true
			})
		}
	}

	// Registry installs the config did not mention: a game installed by an
	// older client, or one whose config entry was pruned.
	for code, path := range installs {
		if _, done := seen[code]; done {
			continue
		}
		obs := gamelib.Observation{
			PlatformKey: BattleNetPlatformKey,
			GameID:      code,
			Name:        battleNetName(code),
			Installed:   dirExists(path),
			InstallPath: path,
			Source:      gamelib.SourceBattleNetReg,
		}
		attributeInstall(&obs, opts)
		b.Observe(obs)
		art = append(art, battleNetArt(code, path))
	}

	applyLauncherArt(ctx, b, BattleNetPlatformKey, art, gamelib.SourceBattleNetConfig, opts)
	games := b.Games()
	if len(games) == 0 {
		res.Unsupported = true
		return res, nil
	}
	if w := ambiguousOwnerWarning("Battle.net", opts); w != "" {
		res.Warnings = append(res.Warnings, w)
	}
	res.Games = games
	return res, nil
}

// battleNetArt collects a product's art candidates.
//
// Blizzard installs carry a .ico beside the launcher stub for most titles, and
// the game executable itself otherwise. Nothing is fetched remotely: Blizzard
// serves its store art from paths that need a product id this resolver does not
// have.
func battleNetArt(code, installPath string) artSource {
	src := artSource{gameID: code, name: battleNetName(code)}
	if strings.TrimSpace(installPath) == "" {
		return src
	}
	src.local = installDirIcons(installPath)
	src.exe = exeForIcon(installPath, "")
	return src
}

func battleNetName(code string) string {
	if name, ok := battleNetProducts[strings.ToLower(code)]; ok {
		return name
	}
	return code
}

// battleNetRegistryInstalls maps product code to install path.
//
// Blizzard writes one uninstall entry per game with the product code in the key
// name, which is the only machine-readable link between the config codes and
// where the games actually live.
func battleNetRegistryInstalls() map[string]string {
	out := map[string]string{}
	parents := []string{
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	for _, parent := range parents {
		names, err := winutil.RegistrySubKeyNames(parent)
		if err != nil {
			continue
		}
		for _, name := range names {
			// Blizzard's uninstall keys are named "Battle.net <code>" or
			// "<Game Name> <code>"; the publisher value is the reliable filter.
			keyPath := parent + `\` + name
			publisher := strings.TrimSpace(winutil.RegistryStringValue(keyPath, "Publisher"))
			if !strings.Contains(strings.ToLower(publisher), "blizzard") {
				continue
			}
			path := strings.TrimSpace(winutil.RegistryStringValue(keyPath, "InstallLocation"))
			if path == "" {
				continue
			}
			code := battleNetCodeFromKeyName(name)
			if code == "" {
				continue
			}
			if _, seen := out[code]; !seen {
				out[code] = path
			}
		}
	}
	return out
}

// battleNetCodeFromKeyName pulls the product code out of an uninstall key name,
// which Blizzard formats as "<Display Name> <code>".
func battleNetCodeFromKeyName(name string) string {
	fields := strings.Fields(strings.TrimSpace(name))
	if len(fields) == 0 {
		return ""
	}
	last := strings.ToLower(fields[len(fields)-1])
	if _, known := battleNetProducts[last]; known {
		return last
	}
	// An unknown trailing token is still the code for a game released after
	// this table was written, as long as it looks like one.
	if len(last) <= 12 && last != "" && !strings.ContainsAny(last, `\/:."'`) {
		return last
	}
	return ""
}
