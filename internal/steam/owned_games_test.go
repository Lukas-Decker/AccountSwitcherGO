package steam

import "testing"

func TestSortOwnedGamesRanksSharedGamesFirst(t *testing.T) {
	games := []OwnedGameDTO{
		{AppID: "3", Name: "Zulu", Owners: []string{"a"}},
		{AppID: "1", Name: "alpha", Owners: []string{"a", "b"}},
		{AppID: "2", Name: "Bravo", Owners: []string{"a"}},
		{AppID: "4", Name: "Delta", Owners: []string{"a", "b", "c"}},
	}

	sortOwnedGames(games)

	want := []string{"4", "1", "2", "3"}
	for i, id := range want {
		if games[i].AppID != id {
			t.Fatalf("position %d = %q (%s), want %q", i, games[i].AppID, games[i].Name, id)
		}
	}
}

func TestGameIconCandidatesPrefersPortraitCapsule(t *testing.T) {
	got := gameIconCandidates("/lc", "440")
	if len(got) == 0 {
		t.Fatal("expected candidates")
	}
	// The grid renders portrait capsules; a wide header is only a fallback.
	if want := "library_600x900.jpg"; !hasSuffixPath(got[0], want) {
		t.Fatalf("first candidate = %q, want it to end in %q", got[0], want)
	}
}

func hasSuffixPath(p, suffix string) bool {
	return len(p) >= len(suffix) && p[len(p)-len(suffix):] == suffix
}

// Steam keeps its own per-user folders in userdata beside real games; listing
// them as games would show an entry every account "owns".
func TestSteamInfraAppIDsCoverScreenshotsAndConfig(t *testing.T) {
	for _, id := range []string{"7", "760", "241100", "250820"} {
		if _, ok := steamInfraAppIDs[id]; !ok {
			t.Fatalf("app id %s should be treated as Steam infrastructure", id)
		}
	}
}
