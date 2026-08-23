package steam

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"account-switcher/internal/appclient"
	"account-switcher/internal/fsutil"
	"account-switcher/internal/paths"
)

// communityGamesCacheTTL keeps the profile game list off the network for a day.
// A library changes when someone buys something, which is rare next to how
// often the games view is opened.
const communityGamesCacheTTL = 24 * time.Hour

// communityGamesMaxBytes caps the response. A large library is a few hundred
// kilobytes of XML; anything past this is not a game list.
const communityGamesMaxBytes = 8 << 20

// communityGame is one entry of a public profile's game list.
type communityGame struct {
	AppID           string
	Name            string
	PlaytimeMinutes int64
}

type communityGamesDoc struct {
	XMLName xml.Name `xml:"gamesList"`
	Error   string   `xml:"error"`
	Games   []struct {
		AppID         string `xml:"appID"`
		Name          string `xml:"name"`
		HoursOnRecord string `xml:"hoursOnRecord"`
	} `xml:"games>game"`
}

// errCommunityProfilePrivate marks the expected failure: Steam only serves this
// list for profiles whose game details are public, and most are not.
var errCommunityProfilePrivate = errors.New("steam profile game list is not public")

func communityGamesCachePath(steamID64 string) (string, error) {
	r, err := paths.LoginCacheDir("Steam")
	if err != nil {
		return "", err
	}
	return filepath.Join(r, "GamesCache", steamID64+".xml"), nil
}

// fetchCommunityGames returns the account's full owned library from its public
// community profile.
//
// This is the only keyless way to see a game the account owns but has never
// installed on this machine. It needs no credentials and no API key, but it
// only answers for a profile whose game details are public, so it tops the
// local sources up rather than replacing them.
func fetchCommunityGames(ctx context.Context, steamID64 string) ([]communityGame, error) {
	steamID64 = normalizeSteamID64(steamID64)
	if steamID64 == "" {
		return nil, fmt.Errorf("invalid steamID64")
	}
	cache, err := communityGamesCachePath(steamID64)
	if err != nil {
		return nil, err
	}

	if raw, ok := readFreshCommunityGamesCache(cache); ok {
		return parseCommunityGames(raw)
	}
	if appclient.IsOfflineMode() {
		// A stale cache still beats nothing when there is no way to refresh it.
		if raw, err := os.ReadFile(cache); err == nil {
			return parseCommunityGames(raw)
		}
		return nil, appclient.ErrOfflineMode
	}

	raw, err := downloadCommunityGames(ctx, steamID64)
	if err != nil {
		if raw, rerr := os.ReadFile(cache); rerr == nil {
			return parseCommunityGames(raw)
		}
		return nil, err
	}
	games, err := parseCommunityGames(raw)
	if err != nil {
		return nil, err
	}
	if err := writeCommunityGamesCache(cache, raw); err != nil {
		steamLog.Debug("steam community game list cache write failed",
			slog.String("steamId64", steamID64), slog.Any("err", err))
	}
	return games, nil
}

func readFreshCommunityGamesCache(path string) ([]byte, bool) {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() == 0 {
		return nil, false
	}
	if time.Since(st.ModTime()) >= communityGamesCacheTTL {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return raw, true
}

func writeCommunityGamesCache(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, raw, 0o644)
}

func downloadCommunityGames(ctx context.Context, steamID64 string) ([]byte, error) {
	url := fmt.Sprintf("https://steamcommunity.com/profiles/%s/games?tab=all&xml=1", steamID64)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/xml")
	req.Header.Set("User-Agent", "account-switcher/3 (game library; +https://github.com/Account-Switcher)")

	resp, err := appclient.Shared.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &profileXMLHTTPError{StatusCode: resp.StatusCode}
	}
	return io.ReadAll(io.LimitReader(resp.Body, communityGamesMaxBytes))
}

func parseCommunityGames(raw []byte) ([]communityGame, error) {
	var doc communityGamesDoc
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if strings.TrimSpace(doc.Error) != "" {
		return nil, errCommunityProfilePrivate
	}
	// A private profile answers with a well-formed but empty list rather than an
	// error, so an empty result is reported as private instead of as "owns
	// nothing", which would otherwise look like a successful, wrong answer.
	if len(doc.Games) == 0 {
		return nil, errCommunityProfilePrivate
	}

	out := make([]communityGame, 0, len(doc.Games))
	for _, g := range doc.Games {
		appID := strings.TrimSpace(g.AppID)
		if appID == "" || !isAllDigitRunes(appID) {
			continue
		}
		if _, skip := steamInfraAppIDs[appID]; skip {
			continue
		}
		out = append(out, communityGame{
			AppID:           appID,
			Name:            strings.TrimSpace(g.Name),
			PlaytimeMinutes: parseHoursOnRecord(g.HoursOnRecord),
		})
	}
	return out, nil
}

// parseHoursOnRecord converts Steam's display string, which carries thousand
// separators once an account passes a thousand hours, into minutes.
func parseHoursOnRecord(s string) int64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return 0
	}
	hours, err := strconv.ParseFloat(s, 64)
	if err != nil || hours <= 0 {
		return 0
	}
	return int64(hours * 60)
}
