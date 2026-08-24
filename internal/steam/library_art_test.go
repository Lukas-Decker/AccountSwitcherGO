package steam

import (
	"path/filepath"
	"strings"
	"testing"
)

func indexOfSuffix(list []string, suffix string) int {
	for i, v := range list {
		if strings.HasSuffix(filepath.ToSlash(v), suffix) {
			return i
		}
	}
	return -1
}

// The user's own grid image is the one they expect to see, so it has to outrank
// everything Steam cached on its own.
func TestSteamArtRequest_UserGridBeatsLauncherCache(t *testing.T) {
	t.Parallel()

	req := steamArtRequest(`C:\Steam`, "730", []string{"12345"})

	if len(req.UserFiles) == 0 {
		t.Fatal("no grid overrides offered")
	}
	if idx := indexOfSuffix(req.UserFiles, "config/grid/730p.png"); idx != 0 {
		t.Errorf("portrait grid override at %d, want first: %v", idx, req.UserFiles)
	}
	for _, p := range req.UserFiles {
		if !strings.Contains(filepath.ToSlash(p), "/userdata/12345/config/grid/") {
			t.Errorf("grid candidate outside the account's grid folder: %q", p)
		}
	}
}

// The grid tiles are 2:3, so a portrait capsule has to be tried before a wide
// header that would have to be cropped.
func TestSteamArtRequest_PortraitBeforeWide(t *testing.T) {
	t.Parallel()

	req := steamArtRequest(`C:\Steam`, "730", nil)

	portrait := indexOfSuffix(req.LocalFiles, "730/library_600x900.jpg")
	header := indexOfSuffix(req.LocalFiles, "730/header.jpg")
	if portrait < 0 || header < 0 {
		t.Fatalf("missing candidates: portrait=%d header=%d", portrait, header)
	}
	if portrait > header {
		t.Errorf("header (%d) is tried before the portrait capsule (%d)", header, portrait)
	}
}

// Steam changed its cache layout and left the old files in place, so an install
// that predates the change is only found by the flat naming.
func TestSteamArtRequest_CoversBothCacheLayouts(t *testing.T) {
	t.Parallel()

	req := steamArtRequest(`C:\Steam`, "730", nil)

	if indexOfSuffix(req.LocalFiles, "librarycache/730/library_600x900.jpg") < 0 {
		t.Error("per-app folder layout missing")
	}
	if indexOfSuffix(req.LocalFiles, "librarycache/730_library_600x900.jpg") < 0 {
		t.Error("flat legacy layout missing")
	}
}

func TestSteamRemoteArtURLs(t *testing.T) {
	t.Parallel()

	urls := steamRemoteArtURLs("730")
	if len(urls) == 0 {
		t.Fatal("no remote URLs")
	}
	for _, u := range urls {
		if !strings.HasPrefix(u, "https://") {
			t.Errorf("non-https CDN URL: %q", u)
		}
		if !strings.Contains(u, "/730/") {
			t.Errorf("URL is not for app 730: %q", u)
		}
		// A key or a session would make this useless on a fresh install, which
		// is exactly the case remote art exists to cover.
		if strings.Contains(u, "key=") || strings.Contains(u, "token") {
			t.Errorf("CDN URL wants credentials: %q", u)
		}
	}
	if !strings.Contains(urls[0], "library_600x900") {
		t.Errorf("first URL = %q, want a portrait capsule", urls[0])
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
