package gameart

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// SteamGridDB is the community artwork archive, and the only source here that
// covers every platform rather than one.
//
// A launcher can only ever offer art for its own games, and most of them offer
// none: Epic, Ubisoft, Rockstar and Battle.net leave nothing but an executable
// to take an icon from. SteamGridDB has grids, heroes, logos and icons for all
// of them, contributed by people rather than publishers, which is exactly the
// gap the rest of the chain cannot fill.
//
// It needs a free API key, so it is off until the user supplies one. Nothing
// here is reachable anonymously and there is no way around that.

// sgdbMaxBytes caps how much of a response is read. A grid list for a popular
// game runs to hundreds of entries and only the first of each shape is used.
const sgdbMaxBytes = 1 << 20

// sgdbBase is a variable so a test can point it at a stub server. Nothing in
// the app changes it.
var sgdbBase = "https://www.steamgriddb.com/api/v2"

// sgdbBaseForTest redirects the archive at a stub and returns the undo.
func sgdbBaseForTest(base string) func() {
	prev := sgdbBase
	sgdbBase = base
	return func() { sgdbBase = prev }
}

var (
	sgdbKeyMu sync.RWMutex
	sgdbKey   string
)

// SetSteamGridDBKey installs the user's API key. An empty key switches the
// source off, which is the default.
func SetSteamGridDBKey(key string) {
	sgdbKeyMu.Lock()
	sgdbKey = strings.TrimSpace(key)
	sgdbKeyMu.Unlock()
}

// SteamGridDBEnabled reports whether a key has been supplied.
func SteamGridDBEnabled() bool {
	sgdbKeyMu.RLock()
	defer sgdbKeyMu.RUnlock()
	return sgdbKey != ""
}

func steamGridDBKey() string {
	sgdbKeyMu.RLock()
	defer sgdbKeyMu.RUnlock()
	return sgdbKey
}

// sgdbAsset is one artwork entry. Only the URL and the shape matter here.
type sgdbAsset struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type sgdbListResponse struct {
	Success bool        `json:"success"`
	Data    []sgdbAsset `json:"data"`
	Errors  []string    `json:"errors"`
}

type sgdbSearchResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

// SteamGridDBCandidates returns artwork candidates for a Steam app id.
//
// Steam app ids are addressable directly, so this is one request rather than a
// search followed by a fetch.
func SteamGridDBCandidates(ctx context.Context, client *http.Client, steamAppID string) []Candidate {
	steamAppID = strings.TrimSpace(steamAppID)
	if steamAppID == "" || !SteamGridDBEnabled() {
		return nil
	}
	return sgdbArtwork(ctx, client, "steam/"+url.PathEscape(steamAppID))
}

// SteamGridDBCandidatesByName looks a game up by title, then returns its
// artwork.
//
// This is the path for every platform that is not Steam, and it costs two
// requests. That is worth paying once: the result is published to the art cache
// and never fetched again, and for these platforms the alternative is an
// executable icon or a blank tile.
func SteamGridDBCandidatesByName(ctx context.Context, client *http.Client, name string) []Candidate {
	name = strings.TrimSpace(name)
	if name == "" || !SteamGridDBEnabled() {
		return nil
	}
	id, ok := sgdbSearch(ctx, client, name)
	if !ok {
		return nil
	}
	return sgdbArtwork(ctx, client, fmt.Sprintf("game/%d", id))
}

// sgdbSearch resolves a title to a SteamGridDB game id.
//
// The first result is taken. The endpoint is an autocomplete, so it is ordered
// by relevance, and a game the user actually has installed is not an obscure
// match for its own name.
func sgdbSearch(ctx context.Context, client *http.Client, name string) (int, bool) {
	var out sgdbSearchResponse
	if !sgdbGet(ctx, client, "/search/autocomplete/"+url.PathEscape(name), &out) {
		return 0, false
	}
	if !out.Success || len(out.Data) == 0 {
		return 0, false
	}
	return out.Data[0].ID, out.Data[0].ID != 0
}

// sgdbArtwork fetches the three shapes worth having for one game reference.
//
// Grids are requested at 600x900 because that is the tile the view draws and
// the archive holds several shapes under the same name. Heroes and logos are
// taken unfiltered, as fallbacks for a game with no grid contributed yet.
func sgdbArtwork(ctx context.Context, client *http.Client, ref string) []Candidate {
	var out []Candidate

	var grids sgdbListResponse
	if sgdbGet(ctx, client, "/grids/"+ref+"?dimensions=600x900&types=static", &grids) && grids.Success {
		for _, a := range grids.Data {
			out = append(out, RemoteURL(TierPortrait, a.URL))
		}
	}
	var heroes sgdbListResponse
	if sgdbGet(ctx, client, "/heroes/"+ref+"?types=static", &heroes) && heroes.Success {
		for _, a := range heroes.Data {
			out = append(out, RemoteURL(TierWide, a.URL))
		}
	}
	var logos sgdbListResponse
	if sgdbGet(ctx, client, "/logos/"+ref+"?types=static", &logos) && logos.Success {
		for _, a := range logos.Data {
			out = append(out, RemoteURL(TierLogo, a.URL))
		}
	}

	// One of each is enough. The chain tries candidates in order and stops at
	// the first that publishes, so carrying a hundred grids for one game would
	// only bloat the request.
	return firstOfEachTier(out)
}

// firstOfEachTier keeps the best entry per shape and drops the rest.
func firstOfEachTier(in []Candidate) []Candidate {
	seen := map[Tier]bool{}
	var out []Candidate
	for _, c := range in {
		if strings.TrimSpace(c.URL) == "" || seen[c.Tier] {
			continue
		}
		seen[c.Tier] = true
		out = append(out, c)
	}
	return out
}

// sgdbGet performs one authenticated request and decodes it.
//
// Every failure is a debug line and a false: a missing key, a rate limit, and a
// game the archive has never heard of all mean the same thing to the chain,
// which is that this source has nothing and the next one should be tried.
func sgdbGet(ctx context.Context, client *http.Client, path string, out any) bool {
	key := steamGridDBKey()
	if key == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, remoteTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sgdbBase+path, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		artLog.Debug("steamgriddb request failed", slog.String("path", path), slog.Any("err", err))
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized {
		artLog.Warn("steamgriddb rejected the API key; artwork from it is unavailable")
		return false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		artLog.Debug("steamgriddb returned no artwork",
			slog.String("path", path), slog.Int("status", resp.StatusCode))
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, sgdbMaxBytes))
	if err != nil {
		return false
	}
	return json.Unmarshal(body, out) == nil
}
