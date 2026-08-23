package steam

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"account-switcher/internal/gamelib"
)

const (
	testOwnerID64 = "76561198000000001"
	testOtherID64 = "76561198000000002"
)

// id32For converts through the same code the resolver uses, so a fixture can
// never drift from the folder name Steam would actually write.
func id32For(t *testing.T, id64 string) string {
	t.Helper()
	f, err := FormatsFromID64(id64)
	if err != nil {
		t.Fatalf("FormatsFromID64(%q): %v", id64, err)
	}
	return f.ID32
}

func writeFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// steamFixture builds a Steam root with two accounts and enough of the real
// layout that every source has something to read.
func steamFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFixture(t, LoginUsersPath(root), `"users"
{
	"`+testOwnerID64+`"
	{
		"AccountName"		"owner"
		"PersonaName"		"Owner"
		"MostRecent"		"1"
	}
	"`+testOtherID64+`"
	{
		"AccountName"		"other"
		"PersonaName"		"Other"
	}
}`)

	// Installed and owned by the first account.
	writeFixture(t, filepath.Join(root, "steamapps", "appmanifest_730.acf"), `"AppState"
{
	"appid"		"730"
	"name"		"Counter-Strike 2"
	"installdir"		"Counter-Strike Global Offensive"
	"LastOwner"		"`+testOwnerID64+`"
	"SizeOnDisk"		"35000000000"
	"LastUpdated"		"1700000000"
	"UserConfig"
	{
		"name"		"should not be read as the app name"
	}
}`)

	// Installed by the second account, in a second library folder.
	lib2 := filepath.Join(root, "..", "Library2")
	writeFixture(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), `"libraryfolders"
{
	"0"
	{
		"path"		"`+filepath.ToSlash(root)+`"
	}
	"1"
	{
		"path"		"`+filepath.ToSlash(lib2)+`"
	}
}`)
	writeFixture(t, filepath.Join(lib2, "steamapps", "appmanifest_570.acf"), `"AppState"
{
	"appid"		"570"
	"name"		"Dota 2"
	"installdir"		"dota 2 beta"
	"LastOwner"		"`+testOtherID64+`"
	"SizeOnDisk"		"50000000000"
}`)

	// The first account has playtime for a game it never installed here.
	writeFixture(t, filepath.Join(root, "userdata", id32For(t, testOwnerID64), "config", "localconfig.vdf"), `"UserLocalConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"730"
					{
						"LastPlayed"		"1700000000"
						"Playtime"		"1234"
					}
					"440"
					{
						"LastPlayed"		"1600000000"
						"Playtime"		"60"
					}
				}
			}
		}
	}
}`)

	// The first account has categorised a game it has neither played nor
	// installed, which is the only local trace of owning it.
	writeFixture(t, filepath.Join(root, "userdata", id32For(t, testOwnerID64), "7", "remote", "sharedconfig.vdf"), `"UserRoamingConfigStore"
{
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"620"
					{
						"tags"
						{
							"0"		"Favorite"
						}
					}
				}
			}
		}
	}
}`)

	// A leftover userdata folder for the second account, and an infrastructure
	// folder that must not be reported as a game.
	if err := os.MkdirAll(filepath.Join(root, "userdata", id32For(t, testOtherID64), "730"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "userdata", id32For(t, testOtherID64), "760"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func resolveFixture(t *testing.T, root string) []gamelib.Game {
	t.Helper()
	res, err := resolveSteamLibraryAt(context.Background(), root, gamelib.Options{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return res.Games
}

func findGame(t *testing.T, games []gamelib.Game, appID string) gamelib.Game {
	t.Helper()
	for _, g := range games {
		if g.GameID == appID {
			return g
		}
	}
	t.Fatalf("no game %q in %d resolved games", appID, len(games))
	return gamelib.Game{}
}

func hasGame(games []gamelib.Game, appID string) bool {
	for _, g := range games {
		if g.GameID == appID {
			return true
		}
	}
	return false
}

// The headline case: an installed game resolves to the account that installed
// it, named by LastOwner, not to whoever happens to have a userdata folder.
func TestResolveSteamLibrary_InstalledGameResolvesToItsInstaller(t *testing.T) {
	games := resolveFixture(t, steamFixture(t))

	cs2 := findGame(t, games, "730")
	if !cs2.Installed {
		t.Error("730 should be installed")
	}
	if cs2.Name != "Counter-Strike 2" {
		t.Errorf("name = %q, want Counter-Strike 2", cs2.Name)
	}
	if cs2.SizeOnDisk != 35000000000 {
		t.Errorf("size = %d, want 35000000000", cs2.SizeOnDisk)
	}
	owner, ok := cs2.InstalledOwner()
	if !ok {
		t.Fatalf("730 has no installing owner: %+v", cs2.Owners)
	}
	if owner.AccountID != testOwnerID64 {
		t.Errorf("installer = %q, want %q", owner.AccountID, testOwnerID64)
	}
	if owner.Confidence != "exact" {
		t.Errorf("confidence = %q, want exact", owner.Confidence)
	}
	if owner.AccountName != "Owner" {
		t.Errorf("account name = %q, want Owner", owner.AccountName)
	}
}

// A second library folder is where a good part of most libraries lives, so the
// scan has to follow libraryfolders.vdf rather than only reading the root.
func TestResolveSteamLibrary_FindsGamesInSecondaryLibraryFolders(t *testing.T) {
	games := resolveFixture(t, steamFixture(t))

	dota := findGame(t, games, "570")
	if !dota.Installed {
		t.Error("570 should be installed")
	}
	owner, ok := dota.InstalledOwner()
	if !ok || owner.AccountID != testOtherID64 {
		t.Errorf("installer = %+v, want %q", owner, testOtherID64)
	}
}

// A nested UserConfig section repeats the "name" key, and reading it instead of
// the app's own name was the bug that made a regex scan unusable here.
func TestResolveSteamLibrary_IgnoresNestedNameKeys(t *testing.T) {
	games := resolveFixture(t, steamFixture(t))

	if got := findGame(t, games, "730").Name; got == "should not be read as the app name" {
		t.Errorf("name came from the nested UserConfig section: %q", got)
	}
}

// Playtime and last-played only exist per account, and a game played but never
// installed here still belongs in the list.
func TestResolveSteamLibrary_LocalConfigContributesPlaytime(t *testing.T) {
	games := resolveFixture(t, steamFixture(t))

	cs2 := findGame(t, games, "730")
	var found bool
	for _, o := range cs2.Owners {
		if o.AccountID != testOwnerID64 {
			continue
		}
		found = true
		if o.PlaytimeMinutes != 1234 {
			t.Errorf("playtime = %d, want 1234", o.PlaytimeMinutes)
		}
		if o.LastPlayed == "" {
			t.Error("lastPlayed missing")
		}
	}
	if !found {
		t.Fatalf("730 has no owner %q", testOwnerID64)
	}

	tf2 := findGame(t, games, "440")
	if tf2.Installed {
		t.Error("440 is not installed and should not claim to be")
	}
	if len(tf2.Owners) != 1 || tf2.Owners[0].AccountID != testOwnerID64 {
		t.Errorf("440 owners = %+v, want just %q", tf2.Owners, testOwnerID64)
	}
}

// sharedconfig is the only local source that sees a game owned but never run,
// which is exactly the gap the old userdata-only scan left.
func TestResolveSteamLibrary_SharedConfigFindsNeverPlayedGames(t *testing.T) {
	games := resolveFixture(t, steamFixture(t))

	portal2 := findGame(t, games, "620")
	if len(portal2.Owners) != 1 || portal2.Owners[0].AccountID != testOwnerID64 {
		t.Fatalf("620 owners = %+v, want %q", portal2.Owners, testOwnerID64)
	}
	if portal2.Owners[0].Confidence != "strong" {
		t.Errorf("confidence = %q, want strong", portal2.Owners[0].Confidence)
	}
}

// A userdata folder proves only that the account ran the app, so it must stay
// the weakest claim and must not be mistaken for having installed the game.
func TestResolveSteamLibrary_UserdataFolderIsAWeakClaim(t *testing.T) {
	games := resolveFixture(t, steamFixture(t))

	cs2 := findGame(t, games, "730")
	for _, o := range cs2.Owners {
		if o.AccountID != testOtherID64 {
			continue
		}
		if o.Confidence != "weak" {
			t.Errorf("confidence = %q, want weak", o.Confidence)
		}
		if o.InstalledBy {
			t.Error("a userdata folder must not imply the account installed the game")
		}
		return
	}
	t.Fatalf("730 has no owner %q from its userdata folder", testOtherID64)
}

// Steam keeps its own per-user folders next to the games; screenshots are not
// a game every account owns.
func TestResolveSteamLibrary_SkipsSteamInfrastructureApps(t *testing.T) {
	games := resolveFixture(t, steamFixture(t))

	if hasGame(games, "760") {
		t.Error("760 (Screenshots) resolved as a game")
	}
}

// With exactly one account there is nobody else it could belong to, so an
// ownerless manifest is attributed, but only as a guess.
func TestResolveSteamLibrary_InfersOwnerOnlyForASingleAccount(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, LoginUsersPath(root), `"users"
{
	"`+testOwnerID64+`"
	{
		"AccountName"		"owner"
		"PersonaName"		"Owner"
	}
}`)
	writeFixture(t, filepath.Join(root, "steamapps", "appmanifest_730.acf"), `"AppState"
{
	"appid"		"730"
	"name"		"Counter-Strike 2"
	"LastOwner"		"0"
}`)

	opts := gamelib.Options{KnownAccounts: map[string]string{testOwnerID64: "Owner"}}
	res, err := resolveSteamLibraryAt(context.Background(), root, opts)
	if err != nil {
		t.Fatal(err)
	}
	g := findGame(t, res.Games, "730")
	owner, ok := g.InstalledOwner()
	if !ok {
		t.Fatalf("730 has no owner: %+v", g.Owners)
	}
	if owner.AccountID != testOwnerID64 {
		t.Errorf("owner = %q, want %q", owner.AccountID, testOwnerID64)
	}
	if owner.Confidence != "inferred" {
		t.Errorf("confidence = %q, want inferred", owner.Confidence)
	}
}

// With two accounts, guessing would name the wrong one for half the library.
func TestResolveSteamLibrary_LeavesOwnerlessInstallsUnattributedWithManyAccounts(t *testing.T) {
	root := steamFixture(t)
	writeFixture(t, filepath.Join(root, "steamapps", "appmanifest_271590.acf"), `"AppState"
{
	"appid"		"271590"
	"name"		"Grand Theft Auto V"
	"LastOwner"		"0"
}`)

	games := resolveFixture(t, root)
	g := findGame(t, games, "271590")
	if _, ok := g.InstalledOwner(); ok {
		t.Errorf("271590 should have no installing owner, got %+v", g.Owners)
	}
	if !g.Installed {
		t.Error("271590 should still be listed as installed")
	}
}

func TestParseAppManifest_RejectsGarbage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	path := filepath.Join(dir, "appmanifest_1.acf")
	writeFixture(t, path, "not a vdf file at all {{{")
	if _, ok := parseAppManifest(path, dir); ok {
		t.Error("garbage manifest parsed as valid")
	}

	noID := filepath.Join(dir, "appmanifest_2.acf")
	writeFixture(t, noID, `"AppState"
{
	"name"		"No AppID"
}`)
	if _, ok := parseAppManifest(noID, dir); ok {
		t.Error("manifest without an appid parsed as valid")
	}
}

func TestNormalizeSteamID64(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		testOwnerID64:        testOwnerID64,
		" " + testOwnerID64:  testOwnerID64,
		"0":                  "",
		"":                   "",
		"1234":               "",
		"7656119800000000x":  "",
		"notasteamid at all": "",
	}
	for in, want := range cases {
		if got := normalizeSteamID64(in); got != want {
			t.Errorf("normalizeSteamID64(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseUnixSeconds_TreatsZeroAsNever(t *testing.T) {
	t.Parallel()
	if got := parseUnixSeconds("0"); !got.IsZero() {
		t.Errorf("0 became %v, want the zero time", got)
	}
	if got := parseUnixSeconds(""); !got.IsZero() {
		t.Errorf("empty became %v, want the zero time", got)
	}
	if got := parseUnixSeconds("1700000000"); got.IsZero() {
		t.Error("a real timestamp became the zero time")
	}
}
