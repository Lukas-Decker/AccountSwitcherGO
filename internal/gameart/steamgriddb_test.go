package gameart

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// withKey installs a key for the duration of one test.
func withKey(t *testing.T, key string) {
	t.Helper()
	SetSteamGridDBKey(key)
	t.Cleanup(func() { SetSteamGridDBKey("") })
}

// The archive has no anonymous access, so with no key it must not make a
// request at all rather than making one and being refused.
func TestSteamGridDB_SilentWithoutAKey(t *testing.T) {
	SetSteamGridDBKey("")

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	defer srv.Close()

	if SteamGridDBEnabled() {
		t.Fatal("enabled with no key")
	}
	if got := SteamGridDBCandidates(context.Background(), srv.Client(), "730"); got != nil {
		t.Errorf("returned %v with no key", got)
	}
	if got := SteamGridDBCandidatesByName(context.Background(), srv.Client(), "Portal"); got != nil {
		t.Errorf("returned %v with no key", got)
	}
	if hits != 0 {
		t.Errorf("made %d requests with no key", hits)
	}
}

// A Steam app id is addressable directly, so it must not go through search.
func TestSteamGridDB_SteamAppIDIsOneLookup(t *testing.T) {
	withKey(t, "test-key")

	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing bearer token on %s", r.URL.Path)
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/grids/"):
			_, _ = w.Write([]byte(`{"success":true,"data":[{"url":"https://cdn.example/grid.png"}]}`))
		case strings.HasPrefix(r.URL.Path, "/heroes/"):
			_, _ = w.Write([]byte(`{"success":true,"data":[{"url":"https://cdn.example/hero.png"}]}`))
		default:
			_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
		}
	}))
	defer srv.Close()
	restore := sgdbBaseForTest(srv.URL)
	defer restore()

	got := SteamGridDBCandidates(context.Background(), srv.Client(), "730")
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want a grid and a hero: %+v", len(got), got)
	}
	if got[0].Tier != TierPortrait || got[1].Tier != TierWide {
		t.Errorf("tiers = %v,%v want portrait,wide", got[0].Tier, got[1].Tier)
	}
	for _, p := range paths {
		if strings.Contains(p, "/search/") {
			t.Errorf("searched by name for a steam appid: %s", p)
		}
		if !strings.Contains(p, "/steam/730") {
			t.Errorf("path %q does not address steam appid 730", p)
		}
	}
}

// Every other platform has to be found by title first.
func TestSteamGridDB_ByNameSearchesThenFetches(t *testing.T) {
	withKey(t, "k")

	var searched, fetched bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/autocomplete/"):
			searched = true
			_, _ = w.Write([]byte(`{"success":true,"data":[{"id":42,"name":"Fortnite"}]}`))
		case strings.Contains(r.URL.Path, "/game/42"):
			fetched = true
			if strings.HasPrefix(r.URL.Path, "/grids/") {
				_, _ = w.Write([]byte(`{"success":true,"data":[{"url":"https://cdn.example/g.png"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	restore := sgdbBaseForTest(srv.URL)
	defer restore()

	got := SteamGridDBCandidatesByName(context.Background(), srv.Client(), "Fortnite")
	if !searched || !fetched {
		t.Fatalf("searched=%v fetched=%v, want both", searched, fetched)
	}
	if len(got) != 1 || got[0].Tier != TierPortrait {
		t.Fatalf("got %+v, want one portrait", got)
	}
}

// A title the archive has never heard of must return nothing rather than
// fetching artwork for whatever the search happened to return.
func TestSteamGridDB_UnknownTitleYieldsNothing(t *testing.T) {
	withKey(t, "k")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
	}))
	defer srv.Close()
	restore := sgdbBaseForTest(srv.URL)
	defer restore()

	if got := SteamGridDBCandidatesByName(context.Background(), srv.Client(), "Nonexistent"); got != nil {
		t.Errorf("got %+v, want nothing", got)
	}
}

// A rejected key must degrade to no artwork, not to an error that breaks the
// pass for every other game.
func TestSteamGridDB_RejectedKeyDegradesQuietly(t *testing.T) {
	withKey(t, "wrong")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	restore := sgdbBaseForTest(srv.URL)
	defer restore()

	if got := SteamGridDBCandidates(context.Background(), srv.Client(), "730"); got != nil {
		t.Errorf("got %+v, want nothing", got)
	}
}

// Only the best entry per shape is carried: a popular game has hundreds of
// grids and the chain stops at the first that publishes.
func TestFirstOfEachTier(t *testing.T) {
	t.Parallel()

	in := []Candidate{
		RemoteURL(TierPortrait, "a"), RemoteURL(TierPortrait, "b"),
		RemoteURL(TierWide, "c"), RemoteURL(TierWide, "d"),
		RemoteURL(TierLogo, ""),
	}
	got := firstOfEachTier(in)
	if len(got) != 2 {
		t.Fatalf("got %d, want one per non-empty tier: %+v", len(got), got)
	}
	if got[0].URL != "a" || got[1].URL != "c" {
		t.Errorf("kept %q and %q, want the first of each tier", got[0].URL, got[1].URL)
	}
}

// The archive is only consulted once the cheap sources have all missed.
func TestResolve_ArchiveIsAReallyLastResort(t *testing.T) {
	setupWwwroot(t)
	dir := t.TempDir()

	var asked bool
	req := Request{
		PlatformKey: "Steam",
		GameID:      "arch-1",
		Candidates: []Candidate{
			LocalFile(TierPortrait, writeFile(t, filepath.Join(dir, "capsule.jpg"), jpegOf(t, 600, 900))),
		},
		Archive: func(ctx context.Context) []Candidate {
			asked = true
			return nil
		},
	}
	if res := Resolve(context.Background(), http.DefaultClient, req, true); res.Tier != TierPortrait {
		t.Fatalf("tier = %v, want portrait", res.Tier)
	}
	if asked {
		t.Error("archive was consulted even though a local portrait resolved")
	}
}

// And it is never consulted when the network is not allowed.
func TestResolve_ArchiveSkippedOffline(t *testing.T) {
	setupWwwroot(t)

	var asked bool
	req := Request{
		PlatformKey: "Steam",
		GameID:      "arch-2",
		Archive: func(ctx context.Context) []Candidate {
			asked = true
			return nil
		},
	}
	Resolve(context.Background(), http.DefaultClient, req, false)
	if asked {
		t.Error("archive was consulted with the network disallowed")
	}
}

// When everything else misses, the archive is what puts a real cover on the
// tile instead of an executable icon.
func TestResolve_ArchiveSuppliesArtWhenNothingElseDoes(t *testing.T) {
	setupWwwroot(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(pngOf(t, 600, 900))
	}))
	defer srv.Close()

	req := Request{
		PlatformKey: "Epic Games",
		GameID:      "Fortnite",
		Archive: func(ctx context.Context) []Candidate {
			return []Candidate{RemoteURL(TierPortrait, srv.URL+"/grid.png")}
		},
	}
	res := Resolve(context.Background(), srv.Client(), req, true)
	if res.Tier != TierPortrait || res.PublicURL == "" {
		t.Fatalf("res = %+v, want a portrait from the archive", res)
	}
}
