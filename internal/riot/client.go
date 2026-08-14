package riot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrNoAPIKey reports that a key-gated call was made without a key. Data
	// Dragon and link building do not raise this.
	ErrNoAPIKey = errors.New("riot: no API key configured")
	// ErrUnauthorized means the key was rejected. A development key that has
	// passed its 24 hour life is the usual cause.
	ErrUnauthorized = errors.New("riot: API key rejected")
	// ErrAccountNotFound means Riot has no such account in that region.
	ErrAccountNotFound = errors.New("riot: account not found")
	// ErrRateLimited means the key's quota is spent. RetryAfter carries Riot's
	// own advice on when to try again.
	ErrRateLimited = errors.New("riot: rate limited")
)

// RateLimitError carries the wait Riot asked for.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("riot: rate limited, retry after %s", e.RetryAfter)
}
func (e *RateLimitError) Is(target error) bool { return target == ErrRateLimited }

// KeyFunc supplies the API key at call time.
//
// A function rather than a string so the key can live in an OS credential store
// and be read only when actually needed, instead of being held in memory for the
// life of the process.
type KeyFunc func() (string, error)

// Client talks to the Riot API and to Data Dragon.
//
// Everything it depends on is passed in: the HTTP client carries the host
// application's offline mode, proxy and timeout policy, and the key function
// decides where a key comes from. The package therefore has no opinion on any of
// that, which is what lets it move to another application unchanged.
type Client struct {
	HTTP *http.Client
	Key  KeyFunc
	// UserAgent is sent on every request. Riot asks callers to identify
	// themselves and a blank agent gets throttled harder.
	UserAgent string
}

// NewClient returns a Client using http.DefaultClient when none is given.
func NewClient(httpClient *http.Client, key KeyFunc) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{HTTP: httpClient, Key: key, UserAgent: "account-switcher"}
}

// HasKey reports whether a key is available, so callers can offer the keyed
// features without provoking an error to find out.
func (c *Client) HasKey() bool {
	if c == nil || c.Key == nil {
		return false
	}
	k, err := c.Key()
	return err == nil && strings.TrimSpace(k) != ""
}

// Account is the identity half of a Riot account, from account-v1.
type Account struct {
	PUUID    string `json:"puuid"`
	GameName string `json:"gameName"`
	TagLine  string `json:"tagLine"`
}

// ID returns the account's Riot ID.
func (a Account) ID() ID { return ID{GameName: a.GameName, TagLine: a.TagLine} }

// Summoner is the League profile attached to a PUUID, from summoner-v4.
type Summoner struct {
	PUUID         string `json:"puuid"`
	ProfileIconID int    `json:"profileIconId"`
	SummonerLevel int    `json:"summonerLevel"`
	RevisionDate  int64  `json:"revisionDate"`
}

// LeagueEntry is one ranked queue's standing, from league-v4.
type LeagueEntry struct {
	QueueType    string `json:"queueType"`
	Tier         string `json:"tier"`
	Rank         string `json:"rank"`
	LeaguePoints int    `json:"leaguePoints"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	HotStreak    bool   `json:"hotStreak"`
}

// Ranked queue identifiers as league-v4 spells them.
const (
	QueueSoloDuo  = "RANKED_SOLO_5x5"
	QueueFlex     = "RANKED_FLEX_SR"
	QueueTFT      = "RANKED_TFT"
	QueueTFTDoubl = "RANKED_TFT_DOUBLE_UP"
)

// Display renders a standing the way the games do, e.g. "Gold IV, 34 LP".
//
// Apex, Master and Challenger have no divisions, so the numeral is dropped
// rather than printed as a meaningless "I".
func (e LeagueEntry) Display() string {
	tier := strings.TrimSpace(e.Tier)
	if tier == "" {
		return ""
	}
	titled := strings.ToUpper(tier[:1]) + strings.ToLower(tier[1:])
	switch strings.ToUpper(tier) {
	case "MASTER", "GRANDMASTER", "CHALLENGER":
		return fmt.Sprintf("%s, %d LP", titled, e.LeaguePoints)
	}
	if r := strings.TrimSpace(e.Rank); r != "" {
		return fmt.Sprintf("%s %s, %d LP", titled, r, e.LeaguePoints)
	}
	return fmt.Sprintf("%s, %d LP", titled, e.LeaguePoints)
}

// WinRate returns wins as a percentage of games played, and whether there were
// any games to divide by.
func (e LeagueEntry) WinRate() (float64, bool) {
	total := e.Wins + e.Losses
	if total <= 0 {
		return 0, false
	}
	return float64(e.Wins) * 100 / float64(total), true
}

func (c *Client) do(ctx context.Context, url string, out any) error {
	if c == nil || c.HTTP == nil {
		return errors.New("riot: client not configured")
	}
	if c.Key == nil {
		return ErrNoAPIKey
	}
	key, err := c.Key()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNoAPIKey, err)
	}
	if strings.TrimSpace(key) == "" {
		return ErrNoAPIKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// The key travels in a header, never a query parameter: query strings end up
	// in proxy logs and browser history in a way headers do not.
	req.Header.Set("X-Riot-Token", key)
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrAccountNotFound
	case http.StatusTooManyRequests:
		return &RateLimitError{RetryAfter: retryAfter(resp.Header)}
	default:
		// Bounded read: an error body is for a message, not a payload.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("riot: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func retryAfter(h http.Header) time.Duration {
	if v := strings.TrimSpace(h.Get("Retry-After")); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 10 * time.Second
}

// AccountByRiotID resolves a Riot ID to a PUUID.
//
// Served from the regional cluster, not the platform host: sending this to
// euw1.api.riotgames.com returns 404 with nothing to say why.
func (c *Client) AccountByRiotID(ctx context.Context, id ID, region Region) (Account, error) {
	if !id.Valid() {
		return Account{}, ErrMissingTag
	}
	name, tag := id.APIPathSegments()
	url := "https://" + region.RouteHost() + "/riot/account/v1/accounts/by-riot-id/" + name + "/" + tag
	var acc Account
	if err := c.do(ctx, url, &acc); err != nil {
		return Account{}, err
	}
	return acc, nil
}

// SummonerByPUUID returns the League profile: level and profile icon id.
//
// Served from the platform host, unlike AccountByRiotID.
func (c *Client) SummonerByPUUID(ctx context.Context, puuid string, region Region) (Summoner, error) {
	if strings.TrimSpace(puuid) == "" {
		return Summoner{}, errors.New("riot: empty PUUID")
	}
	url := "https://" + region.Host() + "/lol/summoner/v4/summoners/by-puuid/" + puuid
	var s Summoner
	if err := c.do(ctx, url, &s); err != nil {
		return Summoner{}, err
	}
	return s, nil
}

// LeagueEntriesByPUUID returns every ranked standing for the account.
//
// An unranked account is an empty list rather than an error, so the caller shows
// "Unranked" instead of a failure.
func (c *Client) LeagueEntriesByPUUID(ctx context.Context, puuid string, region Region) ([]LeagueEntry, error) {
	if strings.TrimSpace(puuid) == "" {
		return nil, errors.New("riot: empty PUUID")
	}
	url := "https://" + region.Host() + "/lol/league/v4/entries/by-puuid/" + puuid
	var entries []LeagueEntry
	if err := c.do(ctx, url, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// EntryForQueue picks one queue's standing out of a set of entries.
func EntryForQueue(entries []LeagueEntry, queueType string) (LeagueEntry, bool) {
	for _, e := range entries {
		if strings.EqualFold(e.QueueType, queueType) {
			return e, true
		}
	}
	return LeagueEntry{}, false
}
