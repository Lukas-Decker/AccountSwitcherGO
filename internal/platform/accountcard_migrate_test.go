package platform

import "testing"

// A user who never touched the display settings should end up with no
// overrides at all, following whatever card shape they later choose, rather
// than a configuration pinning every block to the value it happened to have.
func TestMigrationRecordsNothingWhenNothingWasConfigured(t *testing.T) {
	def := DefaultPlatformSettings()
	cfg := MigrateLegacyCardFlags(LegacyCardFlagsFromPlatform(def))

	if cfg.Preset != "small" {
		t.Fatalf("expected the card as it was, got preset %q", cfg.Preset)
	}
	if len(cfg.Blocks) != 0 {
		t.Fatalf("expected no overrides, got %v", cfg.Blocks)
	}
}

func TestMigrationCarriesFlagsThatWereTurnedOff(t *testing.T) {
	cfg := MigrateLegacyCardFlags(LegacyCardFlags{ShowShortNotes: false, ShowLastUsed: false})

	if v, ok := cfg.Blocks["note"]; !ok || v {
		t.Fatal("a note setting that was off should survive as an override")
	}
	if v, ok := cfg.Blocks["lastUsed"]; !ok || v {
		t.Fatal("a last-used setting that was off should survive as an override")
	}
}

// Steam's identifier lines were hidden unless asked for. Getting this direction
// wrong would start showing SteamIDs to people who had deliberately hidden
// them, which is exactly what streamer mode exists to prevent.
func TestMigrationKeepsSteamIdentifiersHiddenUnlessTheyWereShown(t *testing.T) {
	off := MigrateLegacyCardFlags(LegacyCardFlags{
		HasSteamFlags: true, ShowShortNotes: true, ShowLastUsed: true,
		ShowAccUsername: false, ShowSteamID: false, ShowVAC: true, ShowLimited: true,
	})
	if off.Blocks["platformId"] {
		t.Fatal("a SteamID that was hidden must stay hidden")
	}
	if off.Blocks["accountLogin"] {
		t.Fatal("a login name that was hidden must stay hidden")
	}

	on := MigrateLegacyCardFlags(LegacyCardFlags{
		HasSteamFlags: true, ShowShortNotes: true, ShowLastUsed: true,
		ShowAccUsername: true, ShowSteamID: true, ShowVAC: true, ShowLimited: true,
	})
	if !on.Blocks["platformId"] || !on.Blocks["accountLogin"] {
		t.Fatal("identifiers that were shown must stay shown")
	}
}

func TestMigrationKeepsWarningsAsAnAvatarBorder(t *testing.T) {
	cfg := MigrateLegacyCardFlags(LegacyCardFlags{
		HasSteamFlags: true, ShowShortNotes: true, ShowLastUsed: true, ShowVAC: true, ShowLimited: true,
	})
	if cfg.StatusBadgeStyle != "border" {
		t.Fatalf("VAC and limited were drawn as a border; got %q", cfg.StatusBadgeStyle)
	}
}

func TestEnsureOnlyTurnsCustomisationOnWhereThereIsSomethingToPreserve(t *testing.T) {
	untouched, changed := EnsurePlatformCardConfig(DefaultPlatformSettings(), LegacyCardFlagsFromPlatform(DefaultPlatformSettings()))
	if !changed {
		t.Fatal("a platform with no card config should get one")
	}
	if untouched.AccountCardCustomizationEnabled {
		t.Fatal("nothing was configured, so the platform should follow the global shape")
	}

	configured, _ := EnsurePlatformCardConfig(DefaultPlatformSettings(), LegacyCardFlags{ShowShortNotes: false, ShowLastUsed: true})
	if !configured.AccountCardCustomizationEnabled {
		t.Fatal("a platform that was configured should keep the card it had")
	}
}

func TestEnsureLeavesAnExistingConfigAlone(t *testing.T) {
	s := DefaultPlatformSettings()
	existing := AccountCardConfig{Version: CardConfigVersion, Preset: "large"}
	s.AccountCard = &existing

	out, changed := EnsurePlatformCardConfig(s, LegacyCardFlags{ShowShortNotes: false})
	if changed {
		t.Fatal("migration must not run twice over a config that already exists")
	}
	if out.AccountCard.Preset != "large" {
		t.Fatal("an existing card config must not be overwritten by the old flags")
	}
}
