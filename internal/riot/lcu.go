package riot

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The League Client exposes the same data the client itself renders over a local
// REST API, with no registration and no key. It is the better source whenever it
// is available: a development key dies after a day and a personal key still has
// to be applied for, while this is simply there for as long as the user has the
// client open.
//
// The catch is whose data it is. The client knows exactly one account, the one
// signed in right now, so it can enrich the active account and nothing else. The
// web API remains the only way to look up an arbitrary saved account.

var (
	// ErrLCUNotRunning reports that no League Client lockfile was found. Expected,
	// not exceptional: the client is usually closed.
	ErrLCUNotRunning = errors.New("riot: League Client is not running")
	// ErrLCUUnreadable reports a lockfile that exists but makes no sense.
	ErrLCUUnreadable = errors.New("riot: League Client lockfile is unreadable")
)

// LCUCredentials is what the lockfile carries.
type LCUCredentials struct {
	Port     int
	Password string
	Protocol string
}

// BaseURL returns the address the client is listening on.
//
// Always loopback: the port in the lockfile is bound to 127.0.0.1 only, and
// pointing this at any other host would be sending the client's password to a
// stranger.
func (c LCUCredentials) BaseURL() string {
	scheme := c.Protocol
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://127.0.0.1:" + strconv.Itoa(c.Port)
}

// authHeader returns the Basic credential. The username is always "riot".
func (c LCUCredentials) authHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("riot:"+c.Password))
}

// ParseLockfile reads the five colon-separated fields the client writes:
// name:pid:port:password:protocol.
func ParseLockfile(data []byte) (LCUCredentials, error) {
	parts := strings.Split(strings.TrimSpace(string(data)), ":")
	if len(parts) < 5 {
		return LCUCredentials{}, fmt.Errorf("%w: got %d fields, want 5", ErrLCUUnreadable, len(parts))
	}
	port, err := strconv.Atoi(parts[2])
	if err != nil || port <= 0 {
		return LCUCredentials{}, fmt.Errorf("%w: bad port %q", ErrLCUUnreadable, parts[2])
	}
	if strings.TrimSpace(parts[3]) == "" {
		return LCUCredentials{}, fmt.Errorf("%w: empty password", ErrLCUUnreadable)
	}
	return LCUCredentials{Port: port, Password: parts[3], Protocol: parts[4]}, nil
}

// LeagueInstallDirs returns the product directories Riot records in
// RiotClientInstalls.json.
//
// Read rather than guessed. Riot lets the user install anywhere, and this
// machine has League on a different drive from the Riot Client, which no amount
// of probing well-known paths would have found.
func LeagueInstallDirs(manifestPath string) ([]string, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest struct {
		AssociatedClient map[string]string `json:"associated_client"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	var out []string
	for dir := range manifest.AssociatedClient {
		if strings.Contains(strings.ToLower(dir), "league of legends") {
			out = append(out, filepath.Clean(filepath.FromSlash(dir)))
		}
	}
	return out, nil
}

// FindLockfile locates a live League Client lockfile.
//
// The lockfile exists only while the client is running and is deleted on exit,
// so its presence is the availability check; there is no separate "is it up"
// question worth asking.
func FindLockfile(manifestPath string, extraDirs ...string) (string, error) {
	var dirs []string
	if installs, err := LeagueInstallDirs(manifestPath); err == nil {
		dirs = append(dirs, installs...)
	}
	dirs = append(dirs, extraDirs...)

	for _, dir := range dirs {
		p := filepath.Join(dir, "lockfile")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", ErrLCUNotRunning
}

// LCUClient talks to the local League Client.
type LCUClient struct {
	creds LCUCredentials
	http  *http.Client
}

// NewLCUClient builds a client for the given credentials.
//
// It carries its own http.Client rather than borrowing the caller's, because it
// needs a TLS setting no shared client should ever have. The League Client
// serves a self-signed certificate, so verification has to be off; that is only
// acceptable because the dialler below refuses to connect anywhere except
// loopback, which makes the usual risk, talking to an impostor across a network,
// impossible rather than merely unlikely.
func NewLCUClient(creds LCUCredentials) *LCUClient {
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	return &LCUClient{
		creds: creds,
		http: &http.Client{
			Timeout: 6 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					host, _, err := net.SplitHostPort(addr)
					if err != nil {
						return nil, err
					}
					if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
						return nil, fmt.Errorf("riot: refusing non-loopback LCU address %q", addr)
					}
					return dialer.DialContext(ctx, network, addr)
				},
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // loopback only, see above
			},
		},
	}
}

// ConnectLCU finds a running client and returns a client for it.
func ConnectLCU(manifestPath string, extraDirs ...string) (*LCUClient, error) {
	lockPath, err := FindLockfile(manifestPath, extraDirs...)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(lockPath)
	if err != nil {
		// Racing the client's own shutdown, which removes the file.
		return nil, ErrLCUNotRunning
	}
	creds, err := ParseLockfile(data)
	if err != nil {
		return nil, err
	}
	return NewLCUClient(creds), nil
}

func (c *LCUClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.creds.BaseURL()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.creds.authHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		// A stale lockfile from a client that exited without cleaning up looks
		// exactly like this, so it reports as "not running" rather than an error.
		return fmt.Errorf("%w: %v", ErrLCUNotRunning, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("riot: LCU %s: %s: %s", path, resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// LCUSummoner is the signed-in account, as the client sees it.
type LCUSummoner struct {
	PUUID         string `json:"puuid"`
	GameName      string `json:"gameName"`
	TagLine       string `json:"tagLine"`
	DisplayName   string `json:"displayName"`
	ProfileIconID int    `json:"profileIconId"`
	SummonerLevel int    `json:"summonerLevel"`
}

// ID returns the Riot ID, falling back to the legacy display name for a client
// old enough to predate Riot IDs.
func (s LCUSummoner) ID() (ID, bool) {
	if strings.TrimSpace(s.GameName) != "" && strings.TrimSpace(s.TagLine) != "" {
		return ID{GameName: s.GameName, TagLine: s.TagLine}, true
	}
	return ID{}, false
}

// CurrentSummoner returns the account currently signed in to the client.
func (c *LCUClient) CurrentSummoner(ctx context.Context) (LCUSummoner, error) {
	var s LCUSummoner
	if err := c.get(ctx, "/lol-summoner/v1/current-summoner", &s); err != nil {
		return LCUSummoner{}, err
	}
	return s, nil
}

// lcuRankedStats is the shape the client returns for ranked standings.
type lcuRankedStats struct {
	QueueMap map[string]struct {
		QueueType    string `json:"queueType"`
		Tier         string `json:"tier"`
		Division     string `json:"division"`
		LeaguePoints int    `json:"leaguePoints"`
		Wins         int    `json:"wins"`
		Losses       int    `json:"losses"`
	} `json:"queueMap"`
}

// CurrentRankedStats returns the signed-in account's standings in the same shape
// the web API uses, so callers do not care which source produced them.
//
// The client says "division" where the web API says "rank", and writes "NA" for
// the tiers that have no divisions; both are normalised away here.
func (c *LCUClient) CurrentRankedStats(ctx context.Context) ([]LeagueEntry, error) {
	var stats lcuRankedStats
	if err := c.get(ctx, "/lol-ranked/v1/current-ranked-stats", &stats); err != nil {
		return nil, err
	}
	var out []LeagueEntry
	for key, q := range stats.QueueMap {
		tier := strings.TrimSpace(q.Tier)
		if tier == "" || strings.EqualFold(tier, "NONE") {
			continue
		}
		division := strings.TrimSpace(q.Division)
		if strings.EqualFold(division, "NA") {
			division = ""
		}
		queueType := strings.TrimSpace(q.QueueType)
		if queueType == "" {
			queueType = key
		}
		out = append(out, LeagueEntry{
			QueueType:    queueType,
			Tier:         tier,
			Rank:         division,
			LeaguePoints: q.LeaguePoints,
			Wins:         q.Wins,
			Losses:       q.Losses,
		})
	}
	return out, nil
}
