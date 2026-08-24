package basic

import (
	"encoding/json"
	"strings"
	"testing"
)

// GameStats.json is a file the user is invited to edit, so seeding it must be
// additive. It used to be written over on every load, which meant an edit
// survived until the next launch and a definition dropped from the shipped copy
// took the user's configuration for that game with it.

const userGameStats = `{
  "StatsDefinitions": {
    "Apex Legends": {"UniqueId": "AL", "Url": "https://example.invalid/apex"},
    "My Custom Game": {"UniqueId": "MCG", "Url": "https://example.invalid/mine"}
  },
  "PlatformCompatibilities": {
    "Steam": ["Apex Legends", "My Custom Game"]
  }
}`

const shippedGameStats = `{
  "StatsDefinitions": {
    "Counter-Strike 2": {"UniqueId": "CS2", "Url": "https://example.invalid/cs2"},
    "Apex Legends": {"UniqueId": "AL", "Url": ""}
  },
  "PlatformCompatibilities": {
    "Steam": ["Counter-Strike 2"],
    "BattleNet": ["Overwatch"]
  }
}`

func parseDefs(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var f struct {
		StatsDefinitions map[string]json.RawMessage `json:"StatsDefinitions"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("merged output does not parse: %v", err)
	}
	return f.StatsDefinitions
}

// A game the user configured must survive a shipped copy that no longer has it,
// and one the shipped copy defines differently must keep the user's version.
func TestMergeGameStatsDefinitions_KeepsUserDefinitions(t *testing.T) {
	t.Parallel()

	out, added, ok := mergeGameStatsDefinitions([]byte(userGameStats), []byte(shippedGameStats))
	if !ok {
		t.Fatal("merge reported the input as unusable")
	}
	if added != 1 {
		t.Fatalf("added = %d, want only Counter-Strike 2", added)
	}

	defs := parseDefs(t, out)
	for _, name := range []string{"Apex Legends", "My Custom Game", "Counter-Strike 2"} {
		if _, present := defs[name]; !present {
			t.Errorf("%q is missing from the merged file", name)
		}
	}
	// The user's Apex URL must not have been replaced by the shipped empty one.
	if !strings.Contains(string(defs["Apex Legends"]), "example.invalid/apex") {
		t.Errorf("user's Apex definition was overwritten: %s", defs["Apex Legends"])
	}
}

// Nothing new to add means nothing is written, so an untouched file keeps its
// formatting and its modification time.
func TestMergeGameStatsDefinitions_NoWriteWhenNothingIsNew(t *testing.T) {
	t.Parallel()

	_, added, ok := mergeGameStatsDefinitions([]byte(userGameStats), []byte(userGameStats))
	if !ok {
		t.Fatal("merge reported the input as unusable")
	}
	if added != 0 {
		t.Errorf("added = %d, want 0 for an identical shipped copy", added)
	}
}

// Platform compatibility entries the user set must survive too, and new
// platforms should still arrive.
func TestMergeGameStatsDefinitions_MergesCompatibility(t *testing.T) {
	t.Parallel()

	out, _, ok := mergeGameStatsDefinitions([]byte(userGameStats), []byte(shippedGameStats))
	if !ok {
		t.Fatal("merge failed")
	}
	var f struct {
		Compat map[string][]string `json:"PlatformCompatibilities"`
	}
	if err := json.Unmarshal(out, &f); err != nil {
		t.Fatal(err)
	}
	// The user's Steam list wins: theirs already had an entry for that key.
	steam := strings.Join(f.Compat["Steam"], ",")
	if !strings.Contains(steam, "My Custom Game") {
		t.Errorf("Steam compatibility lost the user's game: %v", f.Compat["Steam"])
	}
	if len(f.Compat["BattleNet"]) == 0 {
		t.Error("a platform only the shipped copy knows about was not added")
	}
}

// A file the user has hand-edited into something unparseable must be left
// exactly as they left it, not replaced.
func TestMergeGameStatsDefinitions_LeavesBrokenFilesAlone(t *testing.T) {
	t.Parallel()

	if _, _, ok := mergeGameStatsDefinitions([]byte("{ this is not json"), []byte(shippedGameStats)); ok {
		t.Error("an unparseable user file was reported as mergeable")
	}
}
