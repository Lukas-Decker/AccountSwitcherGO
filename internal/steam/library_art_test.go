package steam

import (
	"path/filepath"
	"strings"
	"testing"

	"account-switcher/internal/gameart"
)

// indexOfLocal finds a local candidate whose path ends with suffix.
func indexOfLocal(cands []gameart.Candidate, suffix string) int {
	for i, c := range cands {
		if c.Local() && strings.HasSuffix(filepath.ToSlash(c.Path), suffix) {
			return i
		}
	}
	return -1
}

func indexOfRemote(cands []gameart.Candidate, needle string) int {
	for i, c := range cands {
		if !c.Local() && strings.Contains(c.URL, needle) {
			return i
		}
	}
	return -1
}

func tierOfLocal(t *testing.T, cands []gameart.Candidate, suffix string) gameart.Tier {
	t.Helper()
	i := indexOfLocal(cands, suffix)
	if i < 0 {
		t.Fatalf("no local candidate ending %q", suffix)
	}
	return cands[i].Tier
}

// The user's own grid image is the one they expect to see, so it has to outrank
// everything Steam cached on its own and everything the CDN serves.
func TestSteamArtRequest_UserGridIsTheTopTier(t *testing.T) {
	t.Parallel()

	req := steamArtRequest(`C:\Steam`, "730", "Counter-Strike 2", []string{"12345"})
	ordered := req.Candidates

	i := indexOfLocal(ordered, "config/grid/730p.png")
	if i < 0 {
		t.Fatal("no portrait grid override offered")
	}
	if ordered[i].Tier != gameart.TierUserPicked {
		t.Errorf("grid override tier = %v, want user", ordered[i].Tier)
	}
	for _, c := range ordered {
		if c.Tier != gameart.TierUserPicked {
			continue
		}
		if !strings.Contains(filepath.ToSlash(c.Path), "/userdata/12345/config/grid/") {
			t.Errorf("user-tier candidate outside the account's grid folder: %q", c.Path)
		}
	}
}

// The shapes have to be graded, because that grading is what lets a CDN capsule
// beat a local wordmark.
func TestSteamArtRequest_ShapesAreGraded(t *testing.T) {
	t.Parallel()

	c := steamArtRequest(`C:\Steam`, "730", "Counter-Strike 2", nil).Candidates

	if got := tierOfLocal(t, c, "730/library_600x900.jpg"); got != gameart.TierPortrait {
		t.Errorf("library_600x900 tier = %v, want portrait", got)
	}
	if got := tierOfLocal(t, c, "730/header.jpg"); got != gameart.TierWide {
		t.Errorf("header tier = %v, want wide", got)
	}
	if got := tierOfLocal(t, c, "730/library_hero.jpg"); got != gameart.TierWide {
		t.Errorf("library_hero tier = %v, want wide", got)
	}
	if got := tierOfLocal(t, c, "730/logo.png"); got != gameart.TierLogo {
		t.Errorf("logo tier = %v, want logo", got)
	}
}

// These are the names a current client actually writes, measured against a real
// cache. library_header.jpg in particular was missing before.
func TestSteamArtRequest_CoversTheNamesSteamWrites(t *testing.T) {
	t.Parallel()

	c := steamArtRequest(`C:\Steam`, "730", "Counter-Strike 2", nil).Candidates
	for _, name := range []string{
		"730/library_600x900.jpg",
		"730/library_header.jpg",
		"730/header.jpg",
		"730/library_hero.jpg",
		"730/logo.png",
	} {
		if indexOfLocal(c, name) < 0 {
			t.Errorf("missing candidate %q", name)
		}
	}
}

// A game whose capsule Steam never cached still has to reach the store CDN, and
// the portrait URL has to be tried before the wide one.
func TestSteamRemoteArt_PortraitBeforeWide(t *testing.T) {
	t.Parallel()

	c := steamArtRequest(`C:\Steam`, "730", "Counter-Strike 2", nil).Candidates
	portrait := indexOfRemote(c, "library_600x900.jpg")
	header := indexOfRemote(c, "header.jpg")
	if portrait < 0 || header < 0 {
		t.Fatalf("missing remote candidates: portrait=%d header=%d", portrait, header)
	}
	for _, c := range c {
		if c.Local() || !strings.Contains(c.URL, "library_600x900") {
			continue
		}
		if c.Tier != gameart.TierPortrait {
			t.Errorf("remote capsule tier = %v, want portrait", c.Tier)
		}
	}
}

// The CDN hosts are not interchangeable, and neither needs a key: a build that
// required one would be useless on the fresh install this tier exists for.
func TestSteamRemoteArt_HostsAreKeylessAndDirect(t *testing.T) {
	t.Parallel()

	c := steamArtRequest(`C:\Steam`, "730", "Counter-Strike 2", nil).Candidates
	var remotes int
	for _, cand := range c {
		if cand.Local() {
			continue
		}
		remotes++
		if !strings.HasPrefix(cand.URL, "https://") {
			t.Errorf("non-https CDN URL: %q", cand.URL)
		}
		if !strings.Contains(cand.URL, "/730/") {
			t.Errorf("URL is not for app 730: %q", cand.URL)
		}
		if strings.Contains(cand.URL, "key=") || strings.Contains(cand.URL, "token") {
			t.Errorf("CDN URL wants credentials: %q", cand.URL)
		}
		// shared.cloudflare.steamstatic.com only ever answers a redirect to
		// shared.steamstatic.com, so using it costs a round trip per request.
		if strings.Contains(cand.URL, "shared.cloudflare.steamstatic.com") {
			t.Errorf("URL uses the redirecting alias: %q", cand.URL)
		}
	}
	if remotes == 0 {
		t.Fatal("no remote candidates offered")
	}
}

// The grid folder is per account, so every account on the machine is a place a
// custom image could be, and a duplicate id must not be scanned twice.
func TestSteamUserID32s(t *testing.T) {
	t.Parallel()

	accounts := map[string]string{
		testOwnerID64: "Owner",
		testOtherID64: "Other",
		"garbage":     "Nope",
	}
	got := steamUserID32s(`C:\Steam`, accounts)
	if len(got) != 2 {
		t.Fatalf("got %v, want two ids", got)
	}
	seen := map[string]struct{}{}
	for _, id := range got {
		if _, dup := seen[id]; dup {
			t.Errorf("duplicate id32 %q", id)
		}
		seen[id] = struct{}{}
	}
}
