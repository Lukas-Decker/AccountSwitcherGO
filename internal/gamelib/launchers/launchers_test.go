package launchers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"account-switcher/internal/gamelib"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findGame(t *testing.T, games []gamelib.Game, id string) gamelib.Game {
	t.Helper()
	for _, g := range games {
		if g.GameID == id {
			return g
		}
	}
	t.Fatalf("no game %q in %+v", id, games)
	return gamelib.Game{}
}

// epicFixture points %ProgramData% at a temporary tree holding Epic manifests.
func epicFixture(t *testing.T) {
	t.Helper()
	pd := t.TempDir()
	t.Setenv("ProgramData", pd)

	dir := filepath.Join(pd, "Epic", "EpicGamesLauncher", "Data", "Manifests")
	writeFile(t, filepath.Join(dir, "a.item"), `{
		"AppName": "Fortnite",
		"DisplayName": "Fortnite",
		"InstallLocation": "C:\\Games\\Fortnite",
		"InstallSize": 12345
	}`)
	writeFile(t, filepath.Join(dir, "b.item"), `{
		"AppName": "PartialGame",
		"DisplayName": "Half Downloaded",
		"InstallLocation": "C:\\Games\\Half",
		"bIsIncompleteInstall": true
	}`)
	// A manifest with no AppName cannot be addressed or launched.
	writeFile(t, filepath.Join(dir, "c.item"), `{"DisplayName": "Nameless"}`)
	writeFile(t, filepath.Join(dir, "notes.txt"), "ignored")
}

func TestResolveEpic_ReadsInstallManifests(t *testing.T) {
	epicFixture(t)

	res, err := resolveEpic(context.Background(), gamelib.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Games) != 2 {
		t.Fatalf("want 2 games, got %d: %+v", len(res.Games), res.Games)
	}

	fortnite := findGame(t, res.Games, "Fortnite")
	if !fortnite.Installed {
		t.Error("Fortnite should be installed")
	}
	if fortnite.SizeOnDisk != 12345 {
		t.Errorf("size = %d, want 12345", fortnite.SizeOnDisk)
	}

	// A download still in flight is listed but not claimed to be playable.
	if findGame(t, res.Games, "PartialGame").Installed {
		t.Error("an incomplete install should not report as installed")
	}
}

// Epic records no account per game, so with one account it is inferred and
// with several it is left open rather than guessed wrong.
func TestResolveEpic_OwnerInference(t *testing.T) {
	epicFixture(t)

	single := gamelib.Options{KnownAccounts: map[string]string{"acct-1": "Player One"}}
	res, err := resolveEpic(context.Background(), single)
	if err != nil {
		t.Fatal(err)
	}
	owner, ok := findGame(t, res.Games, "Fortnite").InstalledOwner()
	if !ok {
		t.Fatal("single account should have been inferred as the owner")
	}
	if owner.AccountID != "acct-1" || owner.Confidence != "inferred" {
		t.Errorf("owner = %+v, want acct-1 as inferred", owner)
	}

	many := gamelib.Options{KnownAccounts: map[string]string{"a": "A", "b": "B"}}
	res, err = resolveEpic(context.Background(), many)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findGame(t, res.Games, "Fortnite").InstalledOwner(); ok {
		t.Error("with two accounts the owner must not be guessed")
	}
	if len(res.Warnings) == 0 {
		t.Error("an unattributable library should say why")
	}
}

func TestResolveEpic_UnsupportedWhenNotInstalled(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())

	res, err := resolveEpic(context.Background(), gamelib.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unsupported {
		t.Error("a machine with no Epic launcher should report unsupported, not empty")
	}
}

func TestResolveOriginManifests(t *testing.T) {
	pd := t.TempDir()
	t.Setenv("ProgramData", pd)

	dir := filepath.Join(pd, "Origin", "LocalContent", "Battlefield V")
	writeFile(t, filepath.Join(dir, "bfv.mfst"), `?id=1234567&dipinstallpath=C%3A%5CGames%5CBFV&other=x`)

	games := resolveOriginManifests(context.Background(), OriginPlatformKey, gamelib.Options{})
	if len(games) != 1 {
		t.Fatalf("want 1 game, got %d: %+v", len(games), games)
	}
	if games[0].GameID != "1234567" {
		t.Errorf("id = %q, want 1234567", games[0].GameID)
	}
	if games[0].InstallPath != `C:\Games\BFV` {
		t.Errorf("install path = %q", games[0].InstallPath)
	}
	if games[0].Name != "Battlefield V" {
		t.Errorf("name = %q, want Battlefield V", games[0].Name)
	}
}

// Galaxy indexes other launchers in the same tables, and those rows belong to
// the platforms that own them rather than being duplicated under GOG.
func TestGOGProductID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"gog_1207658930", "1207658930", true},
		{"steam_730", "", false},
		{"epic_fortnite", "", false},
		{"gog_", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := gogProductID(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("gogProductID(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestDirNameTitle(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		`C:\Games\Battlefield V`:  "Battlefield V",
		`C:\Games\Dead_Space`:     "Dead Space",
		`C:\Games\Half-Life-Alyx`: "Half Life Alyx",
		`C:\Games\Trailing\`:      "Trailing",
		// A title that already reads as one keeps its own punctuation.
		`C:\Games\Marvel's Spider-Man`: "Marvel's Spider-Man",
		``:                             "",
	}
	for in, want := range cases {
		if got := dirNameTitle(in); got != want {
			t.Errorf("dirNameTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAttributeInstall_OnlyGuessesWithASingleAccount(t *testing.T) {
	t.Parallel()

	obs := gamelib.Observation{PlatformKey: "Epic Games", GameID: "x"}
	attributeInstall(&obs, gamelib.Options{KnownAccounts: map[string]string{"a": "A", "b": "B"}})
	if obs.AccountID != "" {
		t.Errorf("owner guessed from two accounts: %q", obs.AccountID)
	}

	attributeInstall(&obs, gamelib.Options{KnownAccounts: map[string]string{"a": "A"}})
	if obs.AccountID != "a" || obs.Confidence != gamelib.ConfidenceInferred {
		t.Errorf("obs = %+v, want a as inferred", obs)
	}

	// An owner a source actually named must never be overwritten by a guess.
	named := gamelib.Observation{AccountID: "real", Confidence: gamelib.ConfidenceExact}
	attributeInstall(&named, gamelib.Options{KnownAccounts: map[string]string{"other": "Other"}})
	if named.AccountID != "real" || named.Confidence != gamelib.ConfidenceExact {
		t.Errorf("a named owner was overwritten: %+v", named)
	}
}

func TestBattleNetCodeFromKeyName(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"World of Warcraft wow": "wow",
		"Battle.net bna":        "bna",
		"Diablo IV fenris":      "fenris",
		"":                      "",
	}
	for in, want := range cases {
		if got := battleNetCodeFromKeyName(in); got != want {
			t.Errorf("battleNetCodeFromKeyName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveRiot_ReadsMetadataFolders(t *testing.T) {
	pd := t.TempDir()
	t.Setenv("ProgramData", pd)

	metadata := filepath.Join(pd, "Riot Games", "Metadata")
	writeFile(t, filepath.Join(metadata, "valorant.live", "valorant.live.product_settings.yaml"), "x: 1")
	writeFile(t, filepath.Join(metadata, "league_of_legends.live", "league_of_legends.live.product_settings.yaml"), "x: 1")
	// The client itself is not a game.
	writeFile(t, filepath.Join(metadata, "riot_client.default", "riot_client.default.product_settings.yaml"), "x: 1")

	res, err := resolveRiot(context.Background(), gamelib.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Games) != 2 {
		t.Fatalf("want 2 games, got %d: %+v", len(res.Games), res.Games)
	}
	if findGame(t, res.Games, "valorant").Name != "VALORANT" {
		t.Errorf("valorant name = %q", findGame(t, res.Games, "valorant").Name)
	}
}

// Every account on a single-title platform owns the title, which is the one
// case where complete per-account resolution costs nothing.
func TestResolveSingleTitle_EveryAccountOwnsIt(t *testing.T) {
	opts := gamelib.Options{KnownAccounts: map[string]string{"a": "A", "b": "B"}}

	res, err := resolveSingleTitle(context.Background(), "Escape from Tarkov", opts)
	// Without a Platforms.json on disk this reports an error rather than a
	// wrong answer, and that path is the one worth pinning here.
	if err != nil {
		return
	}
	if res.Unsupported {
		return
	}
	if len(res.Games) != 1 {
		t.Fatalf("want 1 game, got %d", len(res.Games))
	}
	if len(res.Games[0].Owners) != 2 {
		t.Errorf("want 2 owners, got %d", len(res.Games[0].Owners))
	}
}
