package gameart

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"account-switcher/internal/paths"
)

// pngOf and jpegOf build real images, because the chain now decodes a
// candidate's header rather than trusting its first few bytes.
func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func jpegOf(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

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

// The user's own choice outranks anything a publisher shipped.
func TestResolve_UserArtOutranksEverything(t *testing.T) {
	wwwroot := setupWwwroot(t)
	dir := t.TempDir()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "730",
		Candidates: []Candidate{
			LocalFile(TierUserPicked, writeFile(t, filepath.Join(dir, "user.png"), pngOf(t, 60, 90))),
			LocalFile(TierPortrait, writeFile(t, filepath.Join(dir, "capsule.jpg"), jpegOf(t, 60, 90))),
		},
	}
	res := Resolve(context.Background(), http.DefaultClient, req, false)

	if res.Tier != TierUserPicked {
		t.Errorf("tier = %v, want user", res.Tier)
	}
	// The user's file was a PNG and the capsule a JPEG, so the extension proves
	// which one was published.
	if !strings.HasSuffix(res.PublicURL, ".png") {
		t.Errorf("publicURL = %q, want the user's png", res.PublicURL)
	}
	if got := publishedFiles(t, wwwroot, "Steam"); len(got) != 1 {
		t.Errorf("published %v, want exactly one file", got)
	}
}

// The bug this rewrite fixes: a candidate that exists but does not decode used
// to abandon its whole tier, so the next local file was never tried.
func TestResolve_BrokenCandidateFallsThroughToTheNext(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	// A truncated JPEG: correct magic bytes, no decodable image behind them.
	truncated := jpegOf(t, 60, 90)[:20]

	req := Request{
		PlatformKey: "Steam",
		GameID:      "570",
		Candidates: []Candidate{
			LocalFile(TierPortrait, filepath.Join(dir, "missing.jpg")),
			LocalFile(TierPortrait, writeFile(t, filepath.Join(dir, "broken.jpg"), truncated)),
			LocalFile(TierPortrait, writeFile(t, filepath.Join(dir, "good.png"), pngOf(t, 60, 90))),
		},
	}
	res := Resolve(context.Background(), http.DefaultClient, req, false)

	if res.Tier != TierPortrait || !strings.HasSuffix(res.PublicURL, ".png") {
		t.Errorf("res = %+v, want the third candidate", res)
	}
}

// The point of ranking by shape: a cover from the network beats a bare wordmark
// off the disk, even though the disk was free to read.
func TestResolve_RemotePortraitBeatsLocalLogo(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(pngOf(t, 600, 900))
	}))
	defer srv.Close()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "570",
		Candidates: []Candidate{
			LocalFile(TierLogo, writeFile(t, filepath.Join(dir, "logo.png"), pngOf(t, 200, 80))),
			RemoteURL(TierPortrait, srv.URL+"/library_600x900.jpg"),
		},
	}
	if res := Resolve(context.Background(), http.DefaultClient, req, true); res.Tier != TierPortrait {
		t.Errorf("tier = %v, want portrait from the CDN", res.Tier)
	}
}

// Offline, that same game has to settle for the local wordmark rather than
// showing nothing.
func TestResolve_FallsBackToLocalWhenOffline(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "570",
		Candidates: []Candidate{
			LocalFile(TierLogo, writeFile(t, filepath.Join(dir, "logo.png"), pngOf(t, 200, 80))),
			RemoteURL(TierPortrait, "https://example.invalid/capsule.jpg"),
		},
	}
	if res := Resolve(context.Background(), http.DefaultClient, req, false); res.Tier != TierLogo {
		t.Errorf("tier = %v, want the local logo", res.Tier)
	}
}

// Local candidates must never cost a request, and a disallowed network must
// never be touched.
func TestResolve_SkipsRemoteWhenNetworkIsNotAllowed(t *testing.T) {
	setupWwwroot(t)

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write(pngOf(t, 60, 90))
	}))
	defer srv.Close()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "1",
		Candidates:  []Candidate{RemoteURL(TierPortrait, srv.URL+"/a.png")},
	}
	if res := Resolve(context.Background(), srv.Client(), req, false); res.PublicURL != "" {
		t.Errorf("resolved %q with network disallowed", res.PublicURL)
	}
	if hits != 0 {
		t.Errorf("made %d requests with network disallowed", hits)
	}
	if res := Resolve(context.Background(), srv.Client(), req, true); res.Tier != TierPortrait {
		t.Errorf("tier = %v, want portrait", res.Tier)
	}
}

// A CDN answers 404 for a game it has no artwork for, which is ordinary and
// must fall through rather than ending the chain.
func TestResolve_RemoteFallsThroughOn404(t *testing.T) {
	setupWwwroot(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/good.png") {
			_, _ = w.Write(pngOf(t, 60, 90))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "2",
		Candidates: []Candidate{
			RemoteURL(TierPortrait, srv.URL+"/missing.jpg"),
			RemoteURL(TierPortrait, srv.URL+"/good.png"),
		},
	}
	res := Resolve(context.Background(), srv.Client(), req, true)
	if res.Tier != TierPortrait || !strings.HasSuffix(res.PublicURL, ".png") {
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

	req := Request{
		PlatformKey: "Steam",
		GameID:      "3",
		Candidates:  []Candidate{RemoteURL(TierPortrait, srv.URL+"/x.jpg")},
	}
	if res := Resolve(context.Background(), srv.Client(), req, true); res.PublicURL != "" {
		t.Errorf("published an HTML error page as art: %q", res.PublicURL)
	}
}

// A spacer or tracking pixel is a real image and passes every byte check, but
// renders as an empty tile.
func TestResolve_RejectsTinyImages(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "4",
		Candidates: []Candidate{
			LocalFile(TierPortrait, writeFile(t, filepath.Join(dir, "pixel.png"), pngOf(t, 1, 1))),
			LocalFile(TierWide, writeFile(t, filepath.Join(dir, "real.png"), pngOf(t, 460, 215))),
		},
	}
	res := Resolve(context.Background(), http.DefaultClient, req, false)
	if res.Tier != TierWide {
		t.Errorf("tier = %v, want the 1x1 rejected and the real image used", res.Tier)
	}
}

// A better shape becoming reachable must replace what is published, and must
// not leave the old file behind.
func TestResolve_UpgradesAndPrunesTheOld(t *testing.T) {
	wwwroot := setupWwwroot(t)
	dir := t.TempDir()

	logo := LocalFile(TierLogo, writeFile(t, filepath.Join(dir, "logo.png"), pngOf(t, 200, 80)))
	req := Request{PlatformKey: "Steam", GameID: "440", Candidates: []Candidate{logo}}
	if first := Resolve(context.Background(), http.DefaultClient, req, false); first.Tier != TierLogo {
		t.Fatalf("tier = %v, want logo", first.Tier)
	}

	req.Candidates = append(req.Candidates,
		LocalFile(TierPortrait, writeFile(t, filepath.Join(dir, "capsule.jpg"), jpegOf(t, 600, 900))))
	if second := Resolve(context.Background(), http.DefaultClient, req, false); second.Tier != TierPortrait {
		t.Fatalf("tier = %v, want portrait", second.Tier)
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

	capsule := writeFile(t, filepath.Join(dir, "capsule.jpg"), jpegOf(t, 600, 900))
	req := Request{
		PlatformKey: "Steam",
		GameID:      "620",
		Candidates:  []Candidate{LocalFile(TierPortrait, capsule)},
	}
	first := Resolve(context.Background(), http.DefaultClient, req, false)
	if first.PublicURL == "" {
		t.Fatal("nothing published")
	}

	// Steam clearing its cache removes the source but not what was published.
	if err := os.Remove(capsule); err != nil {
		t.Fatal(err)
	}
	second := Resolve(context.Background(), http.DefaultClient, req, false)
	if second.PublicURL != first.PublicURL {
		t.Errorf("second = %q, want the cached %q", second.PublicURL, first.PublicURL)
	}
}

// A second pass over an unchanged library must not re-download anything, and
// must not even ask about a shape it already has.
func TestResolve_CachedArtSkipsTheNetwork(t *testing.T) {
	setupWwwroot(t)

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write(pngOf(t, 600, 900))
	}))
	defer srv.Close()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "5",
		Candidates:  []Candidate{RemoteURL(TierPortrait, srv.URL+"/a.png")},
	}
	Resolve(context.Background(), srv.Client(), req, true)
	Resolve(context.Background(), srv.Client(), req, true)

	if hits != 1 {
		t.Errorf("made %d requests, want 1", hits)
	}
}

// Having a portrait cached must stop the chain asking a CDN about wide art it
// would only reject as worse.
func TestResolve_DoesNotFetchAWorseShapeThanTheOneCached(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write(pngOf(t, 460, 215))
	}))
	defer srv.Close()

	req := Request{
		PlatformKey: "Steam",
		GameID:      "6",
		Candidates: []Candidate{
			LocalFile(TierPortrait, writeFile(t, filepath.Join(dir, "capsule.jpg"), jpegOf(t, 600, 900))),
			RemoteURL(TierWide, srv.URL+"/header.jpg"),
		},
	}
	if res := Resolve(context.Background(), srv.Client(), req, true); res.Tier != TierPortrait {
		t.Fatalf("tier = %v, want portrait", res.Tier)
	}
	if hits != 0 {
		t.Errorf("made %d requests for a shape worse than the local one", hits)
	}
}

// Ordering is the whole contract: best shape first, and the free copy first
// among equals.
func TestOrdered(t *testing.T) {
	t.Parallel()

	req := Request{Candidates: []Candidate{
		RemoteURL(TierLogo, "https://example.com/logo.png"),
		RemoteURL(TierPortrait, "https://example.com/capsule.jpg"),
		LocalFile(TierWide, `C:\header.jpg`),
		LocalFile(TierPortrait, `C:\capsule.jpg`),
		LocalFile(TierUserPicked, `C:\grid.png`),
	}}

	got := req.ordered(true)
	want := []Tier{TierUserPicked, TierPortrait, TierPortrait, TierWide, TierLogo}
	if len(got) != len(want) {
		t.Fatalf("got %d candidates, want %d", len(got), len(want))
	}
	for i, tier := range want {
		if got[i].Tier != tier {
			t.Fatalf("candidate %d = %v, want %v", i, got[i].Tier, tier)
		}
	}
	// The two portraits tie on shape, so the local one is tried first.
	if !got[1].Local() {
		t.Error("remote portrait ordered before the local one")
	}

	// With no network, remote candidates drop out entirely.
	offline := req.ordered(false)
	for _, c := range offline {
		if !c.Local() {
			t.Errorf("remote candidate %+v survived an offline ordering", c)
		}
	}
}

// Game ids are not filenames: Epic uses opaque names and the single-title
// platforms use their own titles, spaces and apostrophes included.
func TestSafeSegment(t *testing.T) {
	t.Parallel()

	if got := safeSegment("730"); got != "730" {
		t.Errorf("safeSegment(730) = %q, want it untouched", got)
	}
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

// A filename from an older schema must read as no cache at all, so art
// re-resolves under the current rules instead of a stale tier being trusted.
func TestTierFromFileName(t *testing.T) {
	t.Parallel()

	cases := map[string]Tier{
		"730@v2t5.png":  TierUserPicked,
		"730@v2t4.jpg":  TierPortrait,
		"730@v2t3.jpg":  TierWide,
		"730@v2t2.png":  TierLogo,
		"730@v2t1.png":  TierIcon,
		"730@v2t9.png":  TierNone,
		"730@4.png":     TierNone,
		"730@v1t4.png":  TierNone,
		"730.jpg":       TierNone,
		"730@v2tx.jpg":  TierNone,
		"no-at-sign":    TierNone,
		"a-b1c2@v2t3.w": TierWide,
	}
	for name, want := range cases {
		if got := tierFromFileName(name); got != want {
			t.Errorf("tierFromFileName(%q) = %v, want %v", name, got, want)
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
			Candidates: []Candidate{
				LocalFile(TierPortrait, writeFile(t, filepath.Join(dir, id+".png"), pngOf(t, 60, 90))),
			},
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
