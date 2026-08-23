package gamelib

import (
	"testing"
	"time"
)

func ownerFor(t *testing.T, g Game, accountID string) Ownership {
	t.Helper()
	for _, o := range g.Owners {
		if o.AccountID == accountID {
			return o
		}
	}
	t.Fatalf("game %q has no owner %q (owners: %+v)", g.GameID, accountID, g.Owners)
	return Ownership{}
}

func gameFor(t *testing.T, games []Game, gameID string) Game {
	t.Helper()
	for _, g := range games {
		if g.GameID == gameID {
			return g
		}
	}
	t.Fatalf("no game %q in %+v", gameID, games)
	return Game{}
}

// A weak source must never demote what a strong one already established, no
// matter which order the resolver happened to read them in.
func TestObserve_StrongerClaimWinsRegardlessOfOrder(t *testing.T) {
	t.Parallel()

	forward := NewBuilder()
	forward.Observe(Observation{
		PlatformKey: "Steam", GameID: "730", AccountID: "76561198000000001",
		Source: SourceSteamAppManifest, Confidence: ConfidenceExact, InstalledBy: true,
	})
	forward.Observe(Observation{
		PlatformKey: "Steam", GameID: "730", AccountID: "76561198000000001",
		Source: SourceSteamUserdata, Confidence: ConfidenceWeak,
	})

	backward := NewBuilder()
	backward.Observe(Observation{
		PlatformKey: "Steam", GameID: "730", AccountID: "76561198000000001",
		Source: SourceSteamUserdata, Confidence: ConfidenceWeak,
	})
	backward.Observe(Observation{
		PlatformKey: "Steam", GameID: "730", AccountID: "76561198000000001",
		Source: SourceSteamAppManifest, Confidence: ConfidenceExact, InstalledBy: true,
	})

	for name, b := range map[string]*Builder{"forward": forward, "backward": backward} {
		games := b.Games()
		if len(games) != 1 {
			t.Fatalf("%s: want 1 game, got %d", name, len(games))
		}
		o := ownerFor(t, games[0], "76561198000000001")
		if o.Confidence != "exact" {
			t.Errorf("%s: confidence = %q, want exact", name, o.Confidence)
		}
		if o.Source != SourceSteamAppManifest {
			t.Errorf("%s: source = %q, want %q", name, o.Source, SourceSteamAppManifest)
		}
		if !o.InstalledBy {
			t.Errorf("%s: installedBy lost", name)
		}
	}
}

// The strongest source rarely knows everything. Facts a weaker source uniquely
// carries, like playtime, have to survive being outranked on attribution.
func TestObserve_WeakerSourceStillContributesItsOwnFacts(t *testing.T) {
	t.Parallel()

	lastPlayed := time.Date(2026, 3, 4, 5, 0, 0, 0, time.UTC)
	b := NewBuilder()
	b.Observe(Observation{
		PlatformKey: "Steam", GameID: "730", Name: "Counter-Strike 2",
		AccountID: "acct", Source: SourceSteamAppManifest,
		Confidence: ConfidenceExact, InstalledBy: true, Installed: true,
	})
	b.Observe(Observation{
		PlatformKey: "Steam", GameID: "730", AccountID: "acct",
		Source: SourceSteamLocalConfig, Confidence: ConfidenceStrong,
		PlaytimeMinutes: 4200, LastPlayed: lastPlayed,
	})

	o := ownerFor(t, b.Games()[0], "acct")
	if o.Confidence != "exact" {
		t.Errorf("confidence = %q, want exact", o.Confidence)
	}
	if o.PlaytimeMinutes != 4200 {
		t.Errorf("playtime = %d, want 4200", o.PlaytimeMinutes)
	}
	if o.LastPlayed != lastPlayed.Format(time.RFC3339) {
		t.Errorf("lastPlayed = %q, want %q", o.LastPlayed, lastPlayed.Format(time.RFC3339))
	}
}

// The point of the whole exercise: an installed game resolves to the account
// that installed it, and that account leads the picker even when another
// account has far more playtime.
func TestGames_InstallingAccountLeadsTheOwners(t *testing.T) {
	t.Parallel()

	b := NewBuilder()
	b.Observe(Observation{
		PlatformKey: "Steam", GameID: "730", Name: "CS2", Installed: true,
		AccountID: "installer", Source: SourceSteamAppManifest,
		Confidence: ConfidenceExact, InstalledBy: true,
	})
	b.Observe(Observation{
		PlatformKey: "Steam", GameID: "730", AccountID: "grinder",
		Source: SourceSteamLocalConfig, Confidence: ConfidenceStrong,
		PlaytimeMinutes: 99999,
	})

	g := b.Games()[0]
	if len(g.Owners) != 2 {
		t.Fatalf("want 2 owners, got %d", len(g.Owners))
	}
	if g.Owners[0].AccountID != "installer" {
		t.Errorf("first owner = %q, want installer", g.Owners[0].AccountID)
	}
	got, ok := g.InstalledOwner()
	if !ok || got.AccountID != "installer" {
		t.Errorf("InstalledOwner() = %+v, %v; want installer", got, ok)
	}
}

// Two accounts owning the same game is one game with two owners, not two rows.
func TestObserve_SharedGameCollapsesToOneEntry(t *testing.T) {
	t.Parallel()

	b := NewBuilder()
	for _, id := range []string{"a", "b"} {
		b.Observe(Observation{
			PlatformKey: "Steam", GameID: "570", Name: "Dota 2",
			AccountID: id, Source: SourceSteamSharedConfig, Confidence: ConfidenceStrong,
		})
	}
	games := b.Games()
	if len(games) != 1 {
		t.Fatalf("want 1 game, got %d", len(games))
	}
	if len(games[0].Owners) != 2 {
		t.Errorf("want 2 owners, got %d", len(games[0].Owners))
	}
}

// The same id on two platforms is two different games, and must not merge.
func TestObserve_SameIDOnDifferentPlatformsStaysSeparate(t *testing.T) {
	t.Parallel()

	b := NewBuilder()
	b.Observe(Observation{PlatformKey: "Steam", GameID: "1", Name: "Steam One"})
	b.Observe(Observation{PlatformKey: "GOG Galaxy", GameID: "1", Name: "GOG One"})

	if got := len(b.Games()); got != 2 {
		t.Fatalf("want 2 games, got %d", got)
	}
}

func TestObserve_DropsUnaddressableObservations(t *testing.T) {
	t.Parallel()

	b := NewBuilder()
	b.Observe(Observation{PlatformKey: "Steam", Name: "no id"})
	b.Observe(Observation{GameID: "730", Name: "no platform"})

	if got := len(b.Games()); got != 0 {
		t.Fatalf("want 0 games, got %d", got)
	}
}

// Merge is what joins per-platform results, so it has to preserve the grading
// rather than flatten everything into one undifferentiated list.
func TestMerge_PreservesConfidenceAndCollapsesDuplicates(t *testing.T) {
	t.Parallel()

	weak := []Game{{
		PlatformKey: "Steam", GameID: "730", Name: "CS2",
		Owners: []Ownership{{AccountID: "a", Source: SourceSteamUserdata, Confidence: "weak"}},
	}}
	exact := []Game{{
		PlatformKey: "Steam", GameID: "730", Name: "CS2", Installed: true,
		Owners: []Ownership{{
			AccountID: "a", Source: SourceSteamAppManifest,
			Confidence: "exact", InstalledBy: true,
		}},
	}}

	games := Merge(weak, exact)
	if len(games) != 1 {
		t.Fatalf("want 1 game, got %d", len(games))
	}
	if !games[0].Installed {
		t.Error("installed flag lost in merge")
	}
	o := ownerFor(t, games[0], "a")
	if o.Confidence != "exact" || !o.InstalledBy {
		t.Errorf("owner = %+v, want exact and installedBy", o)
	}
}

// Installed games are the ones a person can act on right now, so they sort
// ahead of a bigger pile of owned-but-absent titles.
func TestSortGames_InstalledFirstThenMostOwners(t *testing.T) {
	t.Parallel()

	games := []Game{
		{PlatformKey: "Steam", GameID: "1", Name: "Owned By Many", Owners: []Ownership{{AccountID: "a"}, {AccountID: "b"}, {AccountID: "c"}}},
		{PlatformKey: "Steam", GameID: "2", Name: "Installed", Installed: true},
	}
	SortGames(games)

	if games[0].GameID != "2" {
		t.Errorf("first = %q, want the installed game", games[0].Name)
	}
}

func TestSingleKnownAccount(t *testing.T) {
	t.Parallel()

	if _, _, ok := (Options{}).SingleKnownAccount(); ok {
		t.Error("no accounts must not resolve to one")
	}
	two := Options{KnownAccounts: map[string]string{"a": "A", "b": "B"}}
	if _, _, ok := two.SingleKnownAccount(); ok {
		t.Error("two accounts must not resolve to one")
	}
	one := Options{KnownAccounts: map[string]string{"a": "A"}}
	id, name, ok := one.SingleKnownAccount()
	if !ok || id != "a" || name != "A" {
		t.Errorf("got %q/%q/%v, want a/A/true", id, name, ok)
	}
}
