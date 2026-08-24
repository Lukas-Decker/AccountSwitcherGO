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
	"time"
)

// IGDB is the second artwork archive, and the one with real cover art for
// games that never had a community contributor.
//
// It complements SteamGridDB rather than duplicating it: SteamGridDB is people
// uploading grids for games they play, so it is excellent for popular titles
// and empty for obscure ones, while IGDB is a catalogue with a cover for
// essentially every game that has ever shipped. Asking both and taking the best
// shape either offers covers more of a library than either alone.
//
// Authentication is Twitch's client credentials flow, so it needs a client id
// and a secret rather than a single key. Both come from the user; nothing here
// works anonymously.
const (
	igdbTokenURL = "https://id.twitch.tv/oauth2/token"
	igdbAPIBase  = "https://api.igdb.com/v4"

	// igdbImageBase serves the artwork itself, and needs no credentials at all:
	// only finding an image id costs a authenticated call.
	igdbImageBase = "https://images.igdb.com/igdb/image/upload/"

	// igdbSteamCategory is the external_games category for Steam, which is how
	// a Steam app id is turned into an IGDB game without guessing at its name.
	igdbSteamCategory = 1

	igdbMaxBytes = 1 << 20

	// igdbMinInterval throttles to IGDB's documented four requests a second.
	// The art resolver runs eight games at once, so without this a cold pass
	// over a large library would be refused rather than served.
	igdbMinInterval = 260 * time.Millisecond
)

var (
	igdbCredMu     sync.RWMutex
	igdbClientID   string
	igdbSecret     string
	igdbToken      string
	igdbTokenUntil time.Time

	igdbRateMu   sync.Mutex
	igdbLastCall time.Time
)

// igdbBase is a variable so a test can point the API at a stub.
var igdbBase = igdbAPIBase

// igdbTokenEndpoint likewise.
var igdbTokenEndpoint = igdbTokenURL

// igdbEndpointsForTest redirects both and returns the undo.
func igdbEndpointsForTest(api, token string) func() {
	prevAPI, prevToken := igdbBase, igdbTokenEndpoint
	igdbBase, igdbTokenEndpoint = api, token
	return func() { igdbBase, igdbTokenEndpoint = prevAPI, prevToken }
}

// SetIGDBCredentials installs the Twitch application credentials. Either one
// empty switches the source off, which is the default.
func SetIGDBCredentials(clientID, clientSecret string) {
	igdbCredMu.Lock()
	igdbClientID = strings.TrimSpace(clientID)
	igdbSecret = strings.TrimSpace(clientSecret)
	// Credentials changing invalidates any token minted from the old ones.
	igdbToken = ""
	igdbTokenUntil = time.Time{}
	igdbCredMu.Unlock()
}

// IGDBEnabled reports whether both credentials have been supplied.
func IGDBEnabled() bool {
	igdbCredMu.RLock()
	defer igdbCredMu.RUnlock()
	return igdbClientID != "" && igdbSecret != ""
}

// igdbCredentials returns the id and secret.
func igdbCredentials() (string, string) {
	igdbCredMu.RLock()
	defer igdbCredMu.RUnlock()
	return igdbClientID, igdbSecret
}

// igdbThrottle spaces requests out to stay inside IGDB's rate limit.
func igdbThrottle() {
	igdbRateMu.Lock()
	defer igdbRateMu.Unlock()
	if wait := igdbMinInterval - time.Since(igdbLastCall); wait > 0 {
		time.Sleep(wait)
	}
	igdbLastCall = time.Now()
}

// igdbAccessToken returns a valid bearer token, minting one when needed.
//
// Tokens last about two months, so this is a once-per-session cost in practice.
// The margin means a token is replaced before it expires rather than after a
// request has already been refused for it.
func igdbAccessToken(ctx context.Context, client *http.Client) (string, bool) {
	igdbCredMu.RLock()
	tok, until := igdbToken, igdbTokenUntil
	igdbCredMu.RUnlock()
	if tok != "" && time.Now().Before(until) {
		return tok, true
	}

	id, secret := igdbCredentials()
	if id == "" || secret == "" {
		return "", false
	}

	form := url.Values{}
	form.Set("client_id", id)
	form.Set("client_secret", secret)
	form.Set("grant_type", "client_credentials")

	ctx, cancel := context.WithTimeout(ctx, remoteTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		igdbTokenEndpoint+"?"+form.Encode(), nil)
	if err != nil {
		return "", false
	}
	resp, err := client.Do(req)
	if err != nil {
		artLog.Debug("igdb token request failed", slog.Any("err", err))
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		artLog.Warn("igdb rejected the credentials; artwork from it is unavailable",
			slog.Int("status", resp.StatusCode))
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, igdbMaxBytes))
	if err != nil {
		return "", false
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if json.Unmarshal(body, &out) != nil || strings.TrimSpace(out.AccessToken) == "" {
		return "", false
	}

	lifetime := time.Duration(out.ExpiresIn) * time.Second
	if lifetime > time.Hour {
		lifetime -= time.Hour
	}
	igdbCredMu.Lock()
	igdbToken = out.AccessToken
	igdbTokenUntil = time.Now().Add(lifetime)
	igdbCredMu.Unlock()
	return out.AccessToken, true
}

// IGDBCandidates returns artwork candidates for one game.
//
// A Steam app id is resolved through the external games table, which is an
// exact mapping rather than a name match. Everything else falls back to a
// search, and takes the first hit for the same reason SteamGridDB does: a game
// the user has installed is not an obscure match for its own title.
func IGDBCandidates(ctx context.Context, client *http.Client, steamAppID, name string) []Candidate {
	if !IGDBEnabled() {
		return nil
	}
	token, ok := igdbAccessToken(ctx, client)
	if !ok {
		return nil
	}

	id, ok := igdbGameID(ctx, client, token, steamAppID, name)
	if !ok {
		return nil
	}
	return igdbArtworkFor(ctx, client, token, id)
}

// igdbGameID resolves a game to its IGDB id.
func igdbGameID(ctx context.Context, client *http.Client, token, steamAppID, name string) (int64, bool) {
	if steamAppID = strings.TrimSpace(steamAppID); steamAppID != "" {
		var rows []struct {
			Game int64 `json:"game"`
		}
		query := fmt.Sprintf(`fields game; where uid = "%s" & category = %d; limit 1;`,
			igdbEscape(steamAppID), igdbSteamCategory)
		if igdbPost(ctx, client, token, "/external_games", query, &rows) && len(rows) > 0 && rows[0].Game != 0 {
			return rows[0].Game, true
		}
	}

	if name = strings.TrimSpace(name); name == "" {
		return 0, false
	}
	var rows []struct {
		ID int64 `json:"id"`
	}
	query := fmt.Sprintf(`search "%s"; fields id; limit 1;`, igdbEscape(name))
	if !igdbPost(ctx, client, token, "/games", query, &rows) || len(rows) == 0 {
		return 0, false
	}
	return rows[0].ID, rows[0].ID != 0
}

// igdbArtworkFor turns a game id into candidates.
//
// The cover is drawn portrait, which is the tile's shape, so it is the reason
// this source is worth asking. Artworks are landscape key art and stand in when
// a catalogue entry has no cover.
func igdbArtworkFor(ctx context.Context, client *http.Client, token string, id int64) []Candidate {
	var rows []struct {
		Cover struct {
			ImageID string `json:"image_id"`
		} `json:"cover"`
		Artworks []struct {
			ImageID string `json:"image_id"`
		} `json:"artworks"`
	}
	query := fmt.Sprintf(`fields cover.image_id, artworks.image_id; where id = %d; limit 1;`, id)
	if !igdbPost(ctx, client, token, "/games", query, &rows) || len(rows) == 0 {
		return nil
	}

	var out []Candidate
	if img := strings.TrimSpace(rows[0].Cover.ImageID); img != "" {
		out = append(out, RemoteURL(TierPortrait, igdbImageURL("t_cover_big", img)))
	}
	for _, a := range rows[0].Artworks {
		if img := strings.TrimSpace(a.ImageID); img != "" {
			out = append(out, RemoteURL(TierWide, igdbImageURL("t_1080p", img)))
			break
		}
	}
	return out
}

// igdbImageURL builds a CDN address. These need no credentials, so once an
// image id is known the download costs nothing but bandwidth.
func igdbImageURL(size, imageID string) string {
	return igdbImageBase + size + "/" + imageID + ".jpg"
}

// igdbEscape makes a value safe to embed in an Apicalypse string literal.
//
// The query language has no parameter binding, so a title containing a quote
// would otherwise end the literal and change the query.
func igdbEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ';' {
			return ' '
		}
		return r
	}, s)
}

// igdbPost performs one Apicalypse query and decodes the result.
//
// Like the other archive, every failure is a debug line and a false: a bad
// credential, a rate limit and a game the catalogue has never heard of all mean
// the same thing here, which is that this source has nothing to offer.
func igdbPost(ctx context.Context, client *http.Client, token, path, query string, out any) bool {
	id, _ := igdbCredentials()
	if id == "" {
		return false
	}
	igdbThrottle()

	ctx, cancel := context.WithTimeout(ctx, remoteTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, igdbBase+path, strings.NewReader(query))
	if err != nil {
		return false
	}
	req.Header.Set("Client-ID", id)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		artLog.Debug("igdb request failed", slog.String("path", path), slog.Any("err", err))
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		// The cached token is the likely culprit, so drop it and let the next
		// call mint a fresh one rather than failing for as long as it is held.
		igdbCredMu.Lock()
		igdbToken = ""
		igdbTokenUntil = time.Time{}
		igdbCredMu.Unlock()
		artLog.Warn("igdb refused the request", slog.Int("status", resp.StatusCode))
		return false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		artLog.Debug("igdb returned no artwork",
			slog.String("path", path), slog.Int("status", resp.StatusCode))
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, igdbMaxBytes))
	if err != nil {
		return false
	}
	return json.Unmarshal(body, out) == nil
}
