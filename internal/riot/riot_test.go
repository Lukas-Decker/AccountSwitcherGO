package riot

import "testing"

// The same account is written three different ways depending on the target, and
// every one of them needs the real characters. Slugifying once loses information
// that cannot be recovered, so the encoders are the thing most worth pinning.
func TestRiotIDEncodesPerTarget(t *testing.T) {
	id, err := ParseID("Hide on bush#KR1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := id.OPGGSlug(); got != "Hide%20on%20bush-KR1" {
		t.Errorf("op.gg slug = %q", got)
	}
	if got := id.TrackerSegment(); got != "Hide%20on%20bush%23KR1" {
		t.Errorf("tracker segment = %q", got)
	}
	// PathEscape, not QueryEscape: a '+' here is read as a literal plus and 404s.
	if name, tag := id.APIPathSegments(); name != "Hide%20on%20bush" || tag != "KR1" {
		t.Errorf("api segments = %q / %q", name, tag)
	}

	// Non-Latin names are legal and must survive intact.
	jp, err := ParseID("クラウド#JP1")
	if err != nil {
		t.Fatalf("parse non-Latin: %v", err)
	}
	if back, err := SplitOPGGSlug(jp.OPGGSlug()); err != nil || back != jp {
		t.Errorf("non-Latin round trip = %+v, %v", back, err)
	}
}

// A game name may contain a dash, so an op.gg slug is ambiguous on its face.
// Splitting on the last one is what makes "Bo-Bo#EUW" survive the trip.
func TestOPGGSlugSplitsOnTheLastDash(t *testing.T) {
	id, err := ParseID("Bo-Bo#EUW")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := id.OPGGSlug(); got != "Bo-Bo-EUW" {
		t.Fatalf("slug = %q", got)
	}
	back, err := SplitOPGGSlug("Bo-Bo-EUW")
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if back.GameName != "Bo-Bo" || back.TagLine != "EUW" {
		t.Errorf("split = %q / %q, want Bo-Bo / EUW", back.GameName, back.TagLine)
	}
}

func TestParseIDRejectsMalformed(t *testing.T) {
	for _, s := range []string{
		"",             // nothing
		"NoTagHere",    // no separator
		"ab#EUW",       // game name too short
		"Name#X",       // tagline too short
		"Name#TOOLONG", // tagline too long
		"A#B#C",        // '#' is only ever a separator
	} {
		if _, err := ParseID(s); err == nil {
			t.Errorf("ParseID(%q) was accepted", s)
		}
	}
}

// Riot serves account-v1 from a regional cluster and summoner/league from a
// platform host. Sending either to the other's host 404s with nothing to say
// why, so the pairing is worth asserting.
func TestRegionRoutingHosts(t *testing.T) {
	r, err := LookupRegion("euw1")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if r.RouteHost() != "europe.api.riotgames.com" {
		t.Errorf("account-v1 host = %s", r.RouteHost())
	}
	if r.Host() != "euw1.api.riotgames.com" {
		t.Errorf("platform host = %s", r.Host())
	}
	// A region pasted from an op.gg URL has to resolve to the same place.
	if alias, err := LookupRegion("euw"); err != nil || alias != r {
		t.Errorf("alias euw = %+v, %v", alias, err)
	}
	if _, err := LookupRegion("nowhere"); err == nil {
		t.Error("unknown region was accepted")
	}
}

// Master and above have no divisions, so printing the numeral would invent one.
func TestLeagueEntryDisplay(t *testing.T) {
	cases := []struct {
		entry LeagueEntry
		want  string
	}{
		{LeagueEntry{Tier: "GOLD", Rank: "IV", LeaguePoints: 34}, "Gold IV, 34 LP"},
		{LeagueEntry{Tier: "CHALLENGER", Rank: "I", LeaguePoints: 1204}, "Challenger, 1204 LP"},
		{LeagueEntry{}, ""},
	}
	for _, c := range cases {
		if got := c.entry.Display(); got != c.want {
			t.Errorf("Display() = %q, want %q", got, c.want)
		}
	}
}
