package gamelib

import (
	"testing"

	"account-switcher/internal/paths"
)

func withPrefsRoot(t *testing.T) {
	t.Helper()
	paths.InitDataRoot(t.TempDir())
	resetPrefsForTest()
	t.Cleanup(resetPrefsForTest)
}

// Hiding has to survive a restart, which means it has to reach the file.
func TestSetHidden_RoundTrips(t *testing.T) {
	withPrefsRoot(t)

	if err := SetHidden("Steam", "730", true); err != nil {
		t.Fatal(err)
	}
	resetPrefsForTest()
	if got := PrefsFor("Steam").Hidden; len(got) != 1 || got[0] != "730" {
		t.Fatalf("hidden = %v, want [730] after a reload", got)
	}

	if err := SetHidden("Steam", "730", false); err != nil {
		t.Fatal(err)
	}
	resetPrefsForTest()
	if got := PrefsFor("Steam").Hidden; len(got) != 0 {
		t.Errorf("hidden = %v, want empty after unhiding", got)
	}
}

// A hidden game is marked, not dropped: the view offers to show hidden games
// again and that must not need a rescan.
func TestApplyPrefs_MarksRatherThanRemoves(t *testing.T) {
	withPrefsRoot(t)
	if err := SetHidden("Steam", "730", true); err != nil {
		t.Fatal(err)
	}

	games := applyPrefs("Steam", []Game{
		{PlatformKey: "Steam", GameID: "730", Name: "Counter-Strike 2"},
		{PlatformKey: "Steam", GameID: "570", Name: "Dota 2"},
	})
	if len(games) != 2 {
		t.Fatalf("got %d games, want both kept", len(games))
	}
	byID := map[string]Game{}
	for _, g := range games {
		byID[g.GameID] = g
	}
	if !byID["730"].Hidden {
		t.Error("730 was not marked hidden")
	}
	if byID["570"].Hidden {
		t.Error("570 was marked hidden")
	}
}

// Pinning artwork replaces whatever the chain chose, and says so.
func TestSetArtOverride_PinsAndClears(t *testing.T) {
	withPrefsRoot(t)

	if err := SetArtOverride("Steam", "730", "/img/games/steam/730@v2t3.jpg"); err != nil {
		t.Fatal(err)
	}
	games := applyPrefs("Steam", []Game{{PlatformKey: "Steam", GameID: "730", ArtURL: "/img/games/steam/730@v2t4.jpg"}})
	if games[0].ArtURL != "/img/games/steam/730@v2t3.jpg" || !games[0].ArtPinned {
		t.Fatalf("got %+v, want the pinned art", games[0])
	}

	if err := SetArtOverride("Steam", "730", ""); err != nil {
		t.Fatal(err)
	}
	games = applyPrefs("Steam", []Game{{PlatformKey: "Steam", GameID: "730", ArtURL: "/chain.jpg"}})
	if games[0].ArtPinned || games[0].ArtURL != "/chain.jpg" {
		t.Errorf("got %+v, want the chain's choice back", games[0])
	}
}

// The bar for guessing adult content is deliberately high: a false positive
// hides a game somebody owns and leaves them hunting for it.
func TestLooksAdult(t *testing.T) {
	t.Parallel()

	adult := []string{
		"Hentai Puzzle", "NSFW Simulator", "Some Game [18+]",
		"Nude Raider", "Strip Poker Night", "Uncensored Edition",
	}
	for _, name := range adult {
		if !looksAdult(name) {
			t.Errorf("looksAdult(%q) = false, want true", name)
		}
	}

	// Whole-word matching: these all contain an adult substring and none of
	// them are adult games.
	safe := []string{
		"Assassin's Creed", "Analogue: A Hate Story", "Cuphead",
		"Grand Theft Auto V", "Sexy Brutale", "Prison Architect",
		"Half-Life", "Middle-earth: Shadow of Mordor", "",
	}
	for _, name := range safe {
		if looksAdult(name) {
			t.Errorf("looksAdult(%q) = true, want false", name)
		}
	}
}

// Once somebody has told the app what a game is, there is no reason to keep
// guessing about it.
func TestApplyRatings_UserAnswerWins(t *testing.T) {
	withPrefsRoot(t)

	yes := true
	if err := SetNSFWOverride("Steam", "570", &yes); err != nil {
		t.Fatal(err)
	}
	no := false
	if err := SetNSFWOverride("Steam", "999", &no); err != nil {
		t.Fatal(err)
	}

	games := applyRatings("Steam", []Game{
		{GameID: "570", Name: "Dota 2"},        // not adult by the guess
		{GameID: "999", Name: "Hentai Puzzle"}, // adult by the guess
		{GameID: "730", Name: "Counter-Strike 2"},
	})
	byID := map[string]Game{}
	for _, g := range games {
		byID[g.GameID] = g
	}
	if !byID["570"].Adult || !byID["570"].AdultOverridden {
		t.Errorf("570 = %+v, want forced adult", byID["570"])
	}
	if byID["999"].Adult || !byID["999"].AdultOverridden {
		t.Errorf("999 = %+v, want forced not adult", byID["999"])
	}
	if byID["730"].Adult || byID["730"].AdultOverridden {
		t.Errorf("730 = %+v, want the default", byID["730"])
	}

	// Clearing returns it to the guess.
	if err := SetNSFWOverride("Steam", "999", nil); err != nil {
		t.Fatal(err)
	}
	games = applyRatings("Steam", []Game{{GameID: "999", Name: "Hentai Puzzle"}})
	if !games[0].Adult || games[0].AdultOverridden {
		t.Errorf("got %+v, want the guess back", games[0])
	}
}

// Preferences are per platform: hiding a Steam appid must not hide an Epic
// game that happens to share the id.
func TestPrefs_ArePerPlatform(t *testing.T) {
	withPrefsRoot(t)
	if err := SetHidden("Steam", "1", true); err != nil {
		t.Fatal(err)
	}
	if got := PrefsFor("Epic Games").Hidden; len(got) != 0 {
		t.Errorf("Epic hidden = %v, want empty", got)
	}
}
