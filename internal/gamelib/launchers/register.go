package launchers

import "account-switcher/internal/gamelib"

// singleTitlePlatforms are the Platforms.json entries that are one game, or one
// application, rather than a library with games inside it.
//
// They are listed rather than detected because the distinction is editorial:
// Discord and OBS are not games at all but they are platforms the switcher
// manages accounts for, and leaving them out of the games view would make it
// look as though resolution had failed for them.
var singleTitlePlatforms = []string{
	"Albion Online",
	"Arena Breakout: Infinite",
	"Delta Force",
	"Discord",
	"Discord Canary",
	"Discord PTB",
	"Escape from Tarkov",
	"GeForce Now",
	"Jagex",
	"Magic Arena",
	"Meta Horizon Link",
	"OBS Studio",
	"PS Remote Play",
	"Roblox",
}

// RegisterAll registers every non-Steam resolver.
//
// Steam registers itself from its own package, since it needs internals that
// are not exported.
func RegisterAll() {
	gamelib.Register(Epic())
	gamelib.Register(GOG())
	gamelib.Register(EADesktop())
	gamelib.Register(Origin())
	gamelib.Register(Ubisoft())
	gamelib.Register(Rockstar())
	gamelib.Register(BattleNet())
	gamelib.Register(Riot())
	gamelib.Register(Oculus())

	for _, key := range singleTitlePlatforms {
		gamelib.Register(SingleTitle(key))
	}
}
