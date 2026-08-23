package steam

import (
	"errors"
	"testing"
)

const communityGamesSample = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<gamesList>
	<steamID64>76561198000000001</steamID64>
	<steamID><![CDATA[Owner]]></steamID>
	<games>
		<game>
			<appID>730</appID>
			<name><![CDATA[Counter-Strike 2]]></name>
			<hoursOnRecord>1,234.5</hoursOnRecord>
		</game>
		<game>
			<appID>570</appID>
			<name><![CDATA[Dota 2]]></name>
			<hoursOnRecord>12.3</hoursOnRecord>
		</game>
		<game>
			<appID>760</appID>
			<name><![CDATA[Screenshots]]></name>
		</game>
	</games>
</gamesList>`

func TestParseCommunityGames(t *testing.T) {
	t.Parallel()

	games, err := parseCommunityGames([]byte(communityGamesSample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// The infrastructure entry is filtered out with the rest.
	if len(games) != 2 {
		t.Fatalf("want 2 games, got %d: %+v", len(games), games)
	}
	if games[0].AppID != "730" || games[0].Name != "Counter-Strike 2" {
		t.Errorf("first game = %+v", games[0])
	}
	// 1,234.5 hours is 74070 minutes, and the separator must not truncate it.
	if games[0].PlaytimeMinutes != 74070 {
		t.Errorf("playtime = %d, want 74070", games[0].PlaytimeMinutes)
	}
}

// A private profile answers with a well-formed empty list. Treating that as
// "owns nothing" would silently drop a whole account's library.
func TestParseCommunityGames_EmptyListIsReportedAsPrivate(t *testing.T) {
	t.Parallel()

	_, err := parseCommunityGames([]byte(`<gamesList><games></games></gamesList>`))
	if !errors.Is(err, errCommunityProfilePrivate) {
		t.Errorf("err = %v, want errCommunityProfilePrivate", err)
	}
}

func TestParseCommunityGames_ExplicitError(t *testing.T) {
	t.Parallel()

	_, err := parseCommunityGames([]byte(`<gamesList><error>This profile is private.</error></gamesList>`))
	if !errors.Is(err, errCommunityProfilePrivate) {
		t.Errorf("err = %v, want errCommunityProfilePrivate", err)
	}
}

func TestParseCommunityGames_RejectsGarbage(t *testing.T) {
	t.Parallel()

	if _, err := parseCommunityGames([]byte("<html>not xml at all")); err == nil {
		t.Error("garbage parsed without error")
	}
}

func TestParseHoursOnRecord(t *testing.T) {
	t.Parallel()

	cases := map[string]int64{
		"":         0,
		"0":        0,
		"0.0":      0,
		"1.0":      60,
		"12.3":     738,
		"1,234.5":  74070,
		"nonsense": 0,
	}
	for in, want := range cases {
		if got := parseHoursOnRecord(in); got != want {
			t.Errorf("parseHoursOnRecord(%q) = %d, want %d", in, got, want)
		}
	}
}
