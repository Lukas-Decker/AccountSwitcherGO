package riot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Data Dragon and Community Dragon are asset mirrors, open to anyone with no
// registration. They will serve profile icon 4568 to anybody who asks, but they
// cannot say whose icon it is: that answer only comes from a keyed call. Keeping
// the two apart is why none of the functions here take a key.

const (
	ddragonBase = "https://ddragon.leagueoflegends.com"
	cdragonBase = "https://raw.communitydragon.org"
)

// ProfileIconURL returns the Data Dragon URL for a profile icon.
//
// Needs the patch version the icon was published under, which is what
// LatestVersion is for.
func ProfileIconURL(version string, iconID int) string {
	return ddragonBase + "/cdn/" + version + "/img/profileicon/" + strconv.Itoa(iconID) + ".png"
}

// ProfileIconURLLatest returns the Community Dragon URL for a profile icon.
//
// Preferred when no version is at hand: Community Dragon serves a "latest" path,
// so an icon added in the newest patch resolves without a version lookup first.
// A brand new icon can also be missing from Data Dragon for a few days after a
// patch, which this avoids.
func ProfileIconURLLatest(iconID int) string {
	return cdragonBase + "/latest/plugins/rcp-be-lol-game-data/global/default/v1/profile-icons/" +
		strconv.Itoa(iconID) + ".jpg"
}

// RankedEmblemURL returns the emblem for a tier, e.g. "GOLD".
//
// Community Dragon only: Data Dragon has never shipped ranked emblems.
func RankedEmblemURL(tier string) string {
	t := normalizeTier(tier)
	if t == "" {
		return ""
	}
	return cdragonBase + "/latest/plugins/rcp-fe-lol-static-assets/global/default/images/ranked-mini-crests/" +
		t + ".svg"
}

func normalizeTier(tier string) string {
	switch t := lower(tier); t {
	case "iron", "bronze", "silver", "gold", "platinum", "emerald",
		"diamond", "master", "grandmaster", "challenger":
		return t
	default:
		return ""
	}
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

// VersionCache holds the current patch version so a list of accounts does not
// fetch it once per account.
//
// Patches land every two weeks, so a day is a generous refresh interval and
// still bounds how long a stale version can linger.
type VersionCache struct {
	TTL time.Duration

	mu      sync.Mutex
	version string
	fetched time.Time
}

const defaultVersionTTL = 24 * time.Hour

// LatestVersion returns the current Data Dragon patch version, e.g. "15.14.1".
//
// Keyless. The HTTP client is passed in for the same reason as everywhere else
// in this package: offline mode and proxy settings belong to the caller.
func (c *VersionCache) LatestVersion(ctx context.Context, httpClient *http.Client) (string, error) {
	ttl := c.TTL
	if ttl <= 0 {
		ttl = defaultVersionTTL
	}

	c.mu.Lock()
	if c.version != "" && time.Since(c.fetched) < ttl {
		v := c.version
		c.mu.Unlock()
		return v, nil
	}
	c.mu.Unlock()

	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ddragonBase+"/api/versions.json", nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("riot: versions.json: %s", resp.Status)
	}

	var versions []string
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "", errors.New("riot: versions.json was empty")
	}

	// Newest first, which is the order Riot publishes.
	c.mu.Lock()
	c.version, c.fetched = versions[0], time.Now()
	c.mu.Unlock()
	return versions[0], nil
}
