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

// A Riot ID copied out of the League client, op.gg or a chat app arrives wrapped
// in bidi isolates, because Riot IDs mix scripts and the surrounding text would
// otherwise reorder around them. They are invisible, so a rejection for being
// too long is unexplainable to the person looking at 15 characters.
func TestParseIDAcceptsACopiedID(t *testing.T) {
	cases := map[string]string{
		"bidi isolates":      "\u2066Average Tibbers\u2069#\u2066TR1\u2069",
		"left-to-right mark": "\u200eAverage Tibbers#TR1",
		"zero width space":   "Average Tibbers\u200b#TR1",
		"byte order mark":    "\ufeffAverage Tibbers#TR1",
	}
	for name, in := range cases {
		id, err := ParseID(in)
		if err != nil {
			t.Errorf("%s: ParseID(%q) = %v", name, in, err)
			continue
		}
		if got := id.String(); got != "Average Tibbers#TR1" {
			t.Errorf("%s: ParseID(%q) = %q", name, in, got)
		}
	}
}

// A non-breaking space is not the space Riot stores, so a name pasted with one
// would be kept as something no lookup can match.
func TestParseIDNormalisesExoticSpaces(t *testing.T) {
	id, err := ParseID("Average\u00a0Tibbers#TR1")
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	if got := id.String(); got != "Average Tibbers#TR1" {
		t.Errorf("ParseID = %q, want a plain space", got)
	}
}

// Stripping invisibles must not smuggle a genuinely over-long name through.
func TestParseIDStillEnforcesTheLimits(t *testing.T) {
	if _, err := ParseID("\u2066ThisNameIsFarTooLong\u2069#TR1"); err == nil {
		t.Error("a 20-character name was accepted")
	}
	if _, err := ParseID("Average Tibbers#\u2066TOOLONG\u2069"); err == nil {
		t.Error("a 7-character tagline was accepted")
	}
}
