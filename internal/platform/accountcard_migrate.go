package platform

// Migration from the per-platform display flags to the card configuration.
//
// Before the card had a shape, what it drew was decided by a handful of
// booleans scattered through each platform's settings: ShowShortNotes,
// ShowLastUsed, and on Steam a further set of Steam_Show* keys. Every one of
// them maps onto a block, so nothing is lost and nothing needs inventing.
//
// The old keys are deliberately left where they are. SavePlatformSettings
// patches rather than replaces, so they cost nothing to keep, and keeping them
// means a downgrade still finds the settings it expects.

// LegacyCardFlags is what the old booleans said, read from whichever settings
// struct a platform uses. Absent flags default to how the card behaved before:
// notes and last-used shown, the Steam-only lines hidden.
type LegacyCardFlags struct {
	ShowShortNotes bool
	ShowLastUsed   bool

	// Steam only. HasSteamFlags separates "this platform has no such setting"
	// from "it has one and it is off", which decide different things.
	HasSteamFlags   bool
	ShowAccUsername bool
	ShowSteamID     bool
	ShowVAC         bool
	ShowLimited     bool
	ShowAvatarFrame bool
}

// LegacyCardFlagsFromPlatform reads the flags a non-Steam platform carries.
func LegacyCardFlagsFromPlatform(s PlatformSettings) LegacyCardFlags {
	return LegacyCardFlags{
		ShowShortNotes: s.ShowShortNotes,
		ShowLastUsed:   s.ShowLastUsed,
	}
}

// MigrateLegacyCardFlags turns the old booleans into a card configuration.
//
// The result is the small preset, because that is the card as it was: every
// block on its own line, no icons, the same geometry. Only the decisions that
// differ from the preset's defaults are recorded, so a user who never changed
// anything gets a configuration with no overrides at all rather than one
// pinning every block to the value it happened to have.
func MigrateLegacyCardFlags(flags LegacyCardFlags) AccountCardConfig {
	cfg := AccountCardConfig{Version: CardConfigVersion, Preset: "small"}
	blocks := map[string]bool{}

	// The small preset shows both of these, so only "off" is worth recording.
	if !flags.ShowShortNotes {
		blocks["note"] = false
	}
	if !flags.ShowLastUsed {
		blocks["lastUsed"] = false
	}

	if flags.HasSteamFlags {
		// These two were off by default on the card, so both directions matter.
		blocks["accountLogin"] = flags.ShowAccUsername
		blocks["platformId"] = flags.ShowSteamID

		// VAC and limited used to be drawn as a border round the avatar and had
		// no other presentation. Preserve that, and only carry the warnings over
		// as a block if the user had asked to see them at all.
		if !flags.ShowVAC && !flags.ShowLimited {
			blocks["badges"] = false
		}
		cfg.StatusBadgeStyle = "border"
	}

	if len(blocks) > 0 {
		cfg.Blocks = blocks
	}
	return cfg
}

// EnsurePlatformCardConfig fills in a platform's card configuration from its
// old display flags the first time it is needed, and leaves an existing one
// alone. Returns the settings and whether anything changed.
//
// Customisation is switched on only where the old flags actually said
// something: a platform whose display settings were all left at their defaults
// has nothing worth preserving and is better off following the global shape,
// where it will pick up whatever the user chooses later. A platform that was
// configured keeps exactly the card it had.
func EnsurePlatformCardConfig(s PlatformSettings, flags LegacyCardFlags) (PlatformSettings, bool) {
	if s.AccountCard != nil {
		return s, false
	}
	migrated := MigrateLegacyCardFlags(flags)
	s.AccountCard = &migrated
	s.AccountCardCustomizationEnabled = len(migrated.Blocks) > 0
	return s, true
}
