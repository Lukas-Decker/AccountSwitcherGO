package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"account-switcher/internal/api"
	"account-switcher/internal/appclient"
	"account-switcher/internal/appconfig"
)

// PlatformsJSONRawURL is the canonical remote Platforms.json used for background updates.
// TODO: switch refs/heads/go to refs/heads/main when the go branch is merged to main.
// PlatformsJSONRawURL points at this project's own copy of Platforms.json.
//
// A build with no repository configured returns "", and the caller skips the
// check rather than fetching platform definitions from somebody else's fork.
func PlatformsJSONRawURL() string {
	return appconfig.RawFileURL("Platforms.json")
}

const maxPlatformsJSONBytes = 4 << 20

type platformsVersion struct {
	Version string `json:"Version"`
}

// ErrNoPlatformsSource means this build has no repository configured to read
// platform definitions from.
var ErrNoPlatformsSource = errors.New("updatecheck: no Platforms.json source configured")

// FetchRemotePlatformsJSON downloads Platforms.json from GitHub.
func FetchRemotePlatformsJSON(ctx context.Context, appVersion string) ([]byte, error) {
	if appclient.IsOfflineMode() {
		return nil, appclient.ErrOfflineMode
	}
	if ctx == nil {
		ctx = context.Background()
	}
	url := PlatformsJSONRawURL()
	if strings.TrimSpace(url) == "" {
		return nil, ErrNoPlatformsSource
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", api.UserAgent(strings.TrimSpace(appVersion)))
	resp, err := appclient.Shared.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPlatformsJSONBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("updatecheck: platforms HTTP %d", resp.StatusCode)
	}
	return body, nil
}

// ParsePlatformsJSONVersion reads the top-level Version field from Platforms.json.
func ParsePlatformsJSONVersion(data []byte) (string, error) {
	var pv platformsVersion
	if err := json.Unmarshal(data, &pv); err != nil {
		return "", err
	}
	v := strings.TrimSpace(pv.Version)
	if v == "" {
		return "", fmt.Errorf("updatecheck: Platforms.json missing Version")
	}
	return v, nil
}

// IsVersionNewer reports whether latest is strictly newer than current using the
// project semver clock (MAJOR.MINOR.PATCH as year.month.day).
func IsVersionNewer(latest, current string) bool {
	latest = strings.TrimSpace(latest)
	current = strings.TrimSpace(current)
	if latest == "" {
		return false
	}
	if current == "" {
		return true
	}
	if _, err := ParseVersionClock(latest); err != nil {
		return false
	}
	if _, err := ParseVersionClock(current); err != nil {
		return true
	}
	return !IsUpToDate(current, latest)
}
