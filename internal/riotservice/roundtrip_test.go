package riotservice

import (
	"testing"

	"account-switcher/internal/basic"
	"account-switcher/internal/paths"
	"account-switcher/internal/riot"
)

// A rank survives being stored and read back only if the division and LP travel
// as their own fields. Rebuilt from the rendered string they would come back as
// "Gold, 0 LP", which looks like real data and is not.
func TestRankSurvivesTheSnapshotRoundTrip(t *testing.T) {
	paths.ResetForTest(t.TempDir())
	withoutAPIKey(t)
	s := New()
	const uid = "round-trip"

	if err := s.SetAccountLink(uid, "Hide on bush#KR1", "kr"); err != nil {
		t.Fatalf("link: %v", err)
	}
	s.storeSnapshot(uid, CardDTO{
		RiotID: "Hide on bush#KR1",
		Region: "kr",
		Level:  412,
		Ranks: []RankDTO{{
			Queue: riot.QueueSoloDuo, Tier: "GOLD", Rank: "IV",
			LeaguePoints: 34, Wins: 60, Losses: 40,
		}},
	})

	link, ok, err := basic.ReadRiotAccountLink(PlatformKey, uid)
	if err != nil || !ok {
		t.Fatalf("read back: ok=%v err=%v", ok, err)
	}
	if len(link.Ranks) != 1 {
		t.Fatalf("stored %d ranks", len(link.Ranks))
	}
	if got := link.Ranks[0]; got.Rank != "IV" || got.LeaguePoints != 34 {
		t.Errorf("stored rank = %+v, want division IV and 34 LP", got)
	}
	if link.CapturedAt.IsZero() {
		t.Error("no capture time recorded, so the age cannot be shown")
	}

	card, err := s.GetCard(uid)
	if err != nil {
		t.Fatalf("get card: %v", err)
	}
	if len(card.Ranks) != 1 || card.Ranks[0].Display != "Gold IV, 34 LP" {
		t.Fatalf("card rank = %+v", card.Ranks)
	}
	if card.Source != "snapshot" || card.CapturedAt == "" {
		t.Errorf("source=%q capturedAt=%q, want a dated snapshot", card.Source, card.CapturedAt)
	}
}

// A Riot ID the user typed must not be replaced by whatever account the client
// happens to be signed in as.
func TestManualRiotIDSurvivesCapture(t *testing.T) {
	paths.ResetForTest(t.TempDir())
	s := New()
	const uid = "manual"

	if err := s.SetAccountLink(uid, "Typed Name#EUW", "euw1"); err != nil {
		t.Fatalf("link: %v", err)
	}
	if err := basic.MergeRiotAccountSnapshot(PlatformKey, uid, basic.RiotAccountLink{
		RiotID: "Somebody Else#KR1", Level: 30,
	}); err != nil {
		t.Fatalf("merge: %v", err)
	}
	link, _, err := basic.ReadRiotAccountLink(PlatformKey, uid)
	if err != nil {
		t.Fatal(err)
	}
	if link.RiotID != "Typed Name#EUW" {
		t.Errorf("manual Riot ID was overwritten with %q", link.RiotID)
	}
	if link.Level != 30 {
		t.Errorf("captured level was discarded: %d", link.Level)
	}
}

// withoutAPIKey makes the test independent of whatever is in the machine's
// credential store, so it exercises the snapshot path rather than quietly
// calling Riot with whichever key the developer happens to have installed.
func withoutAPIKey(t *testing.T) {
	t.Helper()
	prev := apiKeySource
	apiKeySource = func() (string, error) { return "", nil }
	resetKeyProbeForTest()
	t.Cleanup(func() {
		apiKeySource = prev
		resetKeyProbeForTest()
	})
}
