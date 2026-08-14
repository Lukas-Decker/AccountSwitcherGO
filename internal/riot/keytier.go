package riot

import (
	"context"
	"net/http"
	"strings"
)

// Riot does not say what kind of key you hold. There is no endpoint for it and
// no field in any response, so the only available signal is how hard it lets you
// push: every successful call carries an X-App-Rate-Limit header describing the
// quota attached to the key.
//
// A development key is issued with a fixed, well-known allowance. Anything
// noticeably larger came from an application Riot approved. That makes this an
// inference rather than a fact, which is why the tier is reported to the user
// alongside the limit it was inferred from instead of being asserted silently.

// KeyTier is how much the key is trusted to do.
type KeyTier string

const (
	// TierUnknown means no successful call has been made yet.
	TierUnknown KeyTier = "unknown"
	// TierDevelopment is the free key that expires daily, on the starter quota.
	TierDevelopment KeyTier = "development"
	// TierElevated is a personal or production key: a larger allowance that Riot
	// granted deliberately.
	TierElevated KeyTier = "elevated"
)

// developmentAppRateLimit is the allowance every development key is issued with:
// 20 requests per second, 100 per two minutes.
const developmentAppRateLimit = "20:1,100:120"

// KeyInfo is what a probe call learned about the key.
type KeyInfo struct {
	Tier KeyTier `json:"tier"`
	// AppRateLimit is Riot's own description of the quota, shown to the user so
	// the inference can be checked rather than trusted.
	AppRateLimit string `json:"appRateLimit"`
	// Valid is false when the key was rejected outright.
	Valid bool `json:"valid"`
}

// classifyAppRateLimit maps the header to a tier.
func classifyAppRateLimit(header string) KeyTier {
	norm := strings.ReplaceAll(strings.TrimSpace(header), " ", "")
	if norm == "" {
		return TierUnknown
	}
	if norm == developmentAppRateLimit {
		return TierDevelopment
	}
	return TierElevated
}

// ProbeKey makes one cheap call and reports what the key turns out to be.
//
// Uses the platform status endpoint: it is the least expensive thing the API
// serves, needs no account, and still returns the rate-limit headers, so nothing
// about a user's data is touched just to identify a key.
func (c *Client) ProbeKey(ctx context.Context, region Region) (KeyInfo, error) {
	if !c.HasKey() {
		return KeyInfo{Tier: TierUnknown}, ErrNoAPIKey
	}
	key, err := c.Key()
	if err != nil {
		return KeyInfo{Tier: TierUnknown}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://"+region.Host()+"/lol/status/v4/platform-data", nil)
	if err != nil {
		return KeyInfo{Tier: TierUnknown}, err
	}
	req.Header.Set("X-Riot-Token", key)
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return KeyInfo{Tier: TierUnknown}, err
	}
	defer resp.Body.Close()

	limit := resp.Header.Get("X-App-Rate-Limit")
	switch resp.StatusCode {
	case http.StatusOK:
		return KeyInfo{Tier: classifyAppRateLimit(limit), AppRateLimit: limit, Valid: true}, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		// The usual cause is a development key that has passed its daily expiry.
		return KeyInfo{Tier: TierUnknown, AppRateLimit: limit}, ErrUnauthorized
	case http.StatusTooManyRequests:
		// Still informative: only a real key gets throttled rather than refused.
		return KeyInfo{Tier: classifyAppRateLimit(limit), AppRateLimit: limit, Valid: true},
			&RateLimitError{RetryAfter: retryAfter(resp.Header)}
	default:
		return KeyInfo{Tier: TierUnknown, AppRateLimit: limit}, nil
	}
}

// AllowsLiveRefresh reports whether the tier may be polled for fresh data.
//
// A development key may not: its quota is small and it dies every day, so it is
// only good for filling in a snapshot on request.
func (t KeyTier) AllowsLiveRefresh() bool { return t == TierElevated }
