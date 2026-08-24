package gameart

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"account-switcher/internal/paths"
)

// pngBytes is a valid one-pixel PNG, used wherever a test needs bytes that pass
// the magic-number check.
var pngBytes = []byte{
	0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89,
}

var jpegBytes = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}

// setupWwwroot points the app's data root at a temp dir so published art lands
// somewhere disposable.
func setupWwwroot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	paths.InitDataRoot(root)
	return filepath.Join(root, "wwwroot")
}

func writeFile(t *testing.T, path string, body []byte) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// publishedFiles lists what ended up in a platform's art directory.
func publishedFiles(t *testing.T, wwwroot, platformKey string) []string {
	t.Helper()
	ents, err := os.ReadDir(filepath.Join(wwwroot, "img", "games", safeSegment(platformKey)))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	return out
}

// The whole point of the chain: the best available source wins, and the tier is
// recorded so a later pass can tell what it settled for.
func TestResolve_PrefersUserArtOverLauncherArt(t *testing.T) {
	wwwroot := setupWwwroot(t)
	dir := t.TempDir()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "730",
		UserFiles:   []string{writeFile(t, filepath.Join(dir, "user.png"), pngBytes)},
		LocalFiles:  []string{writeFile(t, filepath.Join(dir, "local.jpg"), jpegBytes)},
	}
	res := Resolve(context.Background(), http.DefaultClient, req, false)

	if res.Tier != TierUserPicked {
		t.Errorf("tier = %v, want TierUserPicked", res.Tier)
	}
	// The user's file was a PNG and the launcher's a JPEG, so the extension is
	// proof of which one was actually published.
	if !strings.HasSuffix(res.PublicURL, ".png") {
		t.Errorf("publicURL = %q, want the user's png", res.PublicURL)
	}
	if got := publishedFiles(t, wwwroot, "Steam"); len(got) != 1 {
		t.Errorf("published %v, want exactly one file", got)
	}
}

func TestResolve_FallsBackThroughTheChain(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	// A local candidate that does not exist must not stop the chain.
	req := Request{
		PlatformKey: "Steam",
		GameID:      "570",
		LocalFiles: []string{
			filepath.Join(dir, "missing.jpg"),
			writeFile(t, filepath.Join(dir, "present.jpg"), jpegBytes),
		},
	}
	res := Resolve(context.Background(), http.DefaultClient, req, false)
	if res.Tier != TierLocal {
		t.Fatalf("tier = %v, want TierLocal", res.Tier)
	}
	if !strings.HasSuffix(res.PublicURL, ".jpg") {
		t.Errorf("publicURL = %q", res.PublicURL)
	}
}

// Remote art is only reached when the caller allows it, so a local-only pass
// never blocks on a socket.
func TestResolve_SkipsRemoteWhenNetworkIsNotAllowed(t *testing.T) {
	setupWwwroot(t)

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	req := Request{PlatformKey: "Steam", GameID: "1", RemoteURLs: []string{srv.URL + "/a.png"}}
	if res := Resolve(context.Background(), srv.Client(), req, false); res.PublicURL != "" {
		t.Errorf("resolved %q with network disallowed", res.PublicURL)
	}
	if hits != 0 {
		t.Errorf("made %d requests with network disallowed", hits)
	}

	if res := Resolve(context.Background(), srv.Client(), req, true); res.Tier != TierRemote {
		t.Errorf("tier = %v, want TierRemote", res.Tier)
	}
}

// A CDN answers 404 for a game it has no capsule for, which is ordinary and
// must fall through to the next URL rather than ending the chain.
func TestResolve_RemoteFallsThroughOn404(t *testing.T) {
	setupWwwroot(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/good.png") {
			_, _ = w.Write(pngBytes)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "2",
		RemoteURLs:  []string{srv.URL + "/missing.jpg", srv.URL + "/good.png"},
	}
	res := Resolve(context.Background(), srv.Client(), req, true)
	if res.Tier != TierRemote || !strings.HasSuffix(res.PublicURL, ".png") {
		t.Errorf("res = %+v, want the second URL", res)
	}
}

// Some CDNs answer a missing asset with an HTML error page and a 200, which
// would otherwise be saved as a .jpg and render as a broken tile.
func TestResolve_RejectsNonImageResponses(t *testing.T) {
	setupWwwroot(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("<!doctype html><html><body>Not found</body></html>"))
	}))
	defer srv.Close()

	req := Request{PlatformKey: "Steam", GameID: "3", RemoteURLs: []string{srv.URL + "/x.jpg"}}
	if res := Resolve(context.Background(), srv.Client(), req, true); res.PublicURL != "" {
		t.Errorf("published an HTML error page as art: %q", res.PublicURL)
	}
}

// An exe icon fetched while offline must be replaced once a real capsule
// becomes reachable, and the stale file must not be left behind.
func TestResolve_UpgradesToABetterTierAndPrunesTheOld(t *testing.T) {
	wwwroot := setupWwwroot(t)
	dir := t.TempDir()

	req := Request{PlatformKey: "Steam", GameID: "440"}

	// Start with the weakest tier that can be produced without an executable.
	req.LocalFiles = []string{writeFile(t, filepath.Join(dir, "local.jpg"), jpegBytes)}
	first := Resolve(context.Background(), http.DefaultClient, req, false)
	if first.Tier != TierLocal {
		t.Fatalf("tier = %v, want TierLocal", first.Tier)
	}

	req.UserFiles = []string{writeFile(t, filepath.Join(dir, "user.png"), pngBytes)}
	second := Resolve(context.Background(), http.DefaultClient, req, false)
	if second.Tier != TierUserPicked {
		t.Fatalf("tier = %v, want TierUserPicked", second.Tier)
	}
	if got := publishedFiles(t, wwwroot, "Steam"); len(got) != 1 {
		t.Errorf("published %v, want the old tier pruned", got)
	}
}

// The reverse must not happen: losing the better source keeps what is cached
// rather than downgrading the tile.
func TestResolve_KeepsCachedArtWhenNothingBetterResolves(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	local := writeFile(t, filepath.Join(dir, "local.jpg"), jpegBytes)
	req := Request{PlatformKey: "Steam", GameID: "620", LocalFiles: []string{local}}
	first := Resolve(context.Background(), http.DefaultClient, req, false)
	if first.PublicURL == "" {
		t.Fatal("nothing published")
	}

	// Steam clearing its cache removes the source but not what was published.
	if err := os.Remove(local); err != nil {
		t.Fatal(err)
	}
	second := Resolve(context.Background(), http.DefaultClient, req, false)
	if second.PublicURL != first.PublicURL {
		t.Errorf("second = %q, want the cached %q", second.PublicURL, first.PublicURL)
	}
}

// A second pass over an unchanged library must not re-download anything.
func TestResolve_CachedArtSkipsTheNetwork(t *testing.T) {
	setupWwwroot(t)

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	req := Request{PlatformKey: "Steam", GameID: "4", RemoteURLs: []string{srv.URL + "/a.png"}}
	Resolve(context.Background(), srv.Client(), req, true)
	Resolve(context.Background(), srv.Client(), req, true)

	if hits != 1 {
		t.Errorf("made %d requests, want 1", hits)
	}
}

// Game ids are not filenames: Epic uses opaque names and the single-title
// platforms use their own titles, spaces and apostrophes included.
func TestSafeSegment(t *testing.T) {
	t.Parallel()

	if got := safeSegment("730"); got != "730" {
		t.Errorf("safeSegment(730) = %q, want it untouched", got)
	}
	// Two ids that sanitise the same must not collide on one file.
	a := safeSegment("Escape from Tarkov")
	b := safeSegment("Escape/from/Tarkov")
	if a == b {
		t.Errorf("distinct ids collapsed onto %q", a)
	}
	if strings.ContainsAny(a, ` /\:"'`) {
		t.Errorf("safeSegment left path characters in %q", a)
	}
	if safeSegment("") != "unknown" {
		t.Errorf("empty id = %q, want unknown", safeSegment(""))
	}
}

func TestTierFromFileName(t *testing.T) {
	t.Parallel()

	cases := map[string]Tier{
		"730@4.png":     TierUserPicked,
		"730@3.jpg":     TierLocal,
		"730@2.jpg":     TierRemote,
		"730@1.png":     TierExeIcon,
		"730.jpg":       TierNone,
		"730@x.jpg":     TierNone,
		"no-at-sign":    TierNone,
		"a-b1c2@3.webp": TierLocal,
	}
	for name, want := range cases {
		if got := tierFromFileName(name); got != want {
			t.Errorf("tierFromFileName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestExtFromMagic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  []byte
		want string
		ok   bool
	}{
		{"png", pngBytes, "png", true},
		{"jpeg", jpegBytes, "jpg", true},
		{"gif", []byte("GIF89a....."), "gif", true},
		{"webp", append([]byte("RIFF\x00\x00\x00\x00WEBP"), 0), "webp", true},
		{"ico", []byte{0x00, 0x00, 0x01, 0x00, 0x01}, "ico", true},
		{"html", []byte("<!doctype html>"), "", false},
		{"empty", nil, "", false},
	}
	for _, c := range cases {
		got, ok := extFromMagic(c.raw)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: got %q,%v want %q,%v", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestResolveMany_ResolvesEveryGame(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	var reqs []Request
	for _, id := range []string{"1", "2", "3", "4", "5"} {
		reqs = append(reqs, Request{
			PlatformKey: "Steam",
			GameID:      id,
			LocalFiles:  []string{writeFile(t, filepath.Join(dir, id+".png"), pngBytes)},
		})
	}
	out := ResolveMany(context.Background(), http.DefaultClient, reqs, false)
	if len(out) != len(reqs) {
		t.Fatalf("resolved %d of %d", len(out), len(reqs))
	}
	for _, r := range reqs {
		if out[r.GameID].PublicURL == "" {
			t.Errorf("game %q resolved to nothing", r.GameID)
		}
	}
}

// A game with nothing anywhere is ordinary, and the view falls back to its
// name, so this must be an empty result rather than a failure.
func TestResolve_NothingAvailable(t *testing.T) {
	setupWwwroot(t)

	res := Resolve(context.Background(), http.DefaultClient, Request{PlatformKey: "Steam", GameID: "9"}, true)
	if res.PublicURL != "" || res.Tier != TierNone {
		t.Errorf("res = %+v, want empty", res)
	}
}
