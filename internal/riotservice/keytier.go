package riotservice

import (
	"context"
	"strings"
	"sync"
	"time"

	"account-switcher/internal/riot"
)

// liveRefreshInterval is the shortest gap between web API refreshes of the same
// account.
//
// Ranks move on the timescale of a game, not a second, so a minute costs the
// user nothing and keeps a list of accounts from turning one window focus into a
// burst of calls. Riot's own limits are the ceiling; this is about not going
// near them.
const liveRefreshInterval = time.Minute

// keyProbeTTL bounds how long a key classification is trusted. A development key
// expires daily and a new one may be a different tier, so this is short enough
// to notice within a session.
const keyProbeTTL = 30 * time.Minute

var keyState struct {
	mu sync.Mutex

	info      riot.KeyInfo
	probedKey string
	probedAt  time.Time

	lastRefresh map[string]time.Time
}

// KeyInfoDTO is what the settings page shows about the stored key.
type KeyInfoDTO struct {
	Present bool   `json:"present"`
	Valid   bool   `json:"valid"`
	Tier    string `json:"tier"`
	// AppRateLimit is Riot's own quota string, shown so the inferred tier can be
	// checked rather than taken on trust.
	AppRateLimit string `json:"appRateLimit"`
	LiveAllowed  bool   `json:"liveAllowed"`
	Error        string `json:"error"`
}

// KeyInfo probes the stored key and reports what it is.
//
// Riot publishes no key type, so the tier is inferred from the quota it hands
// back. The quota is returned alongside it for exactly that reason.
func (s *Service) KeyInfo() (KeyInfoDTO, error) {
	key, err := apiKey()
	if err != nil {
		return KeyInfoDTO{}, err
	}
	if strings.TrimSpace(key) == "" {
		return KeyInfoDTO{Tier: string(riot.TierUnknown)}, nil
	}

	info, perr := s.probeKey(key)
	out := KeyInfoDTO{
		Present:      true,
		Valid:        info.Valid,
		Tier:         string(info.Tier),
		AppRateLimit: info.AppRateLimit,
		LiveAllowed:  info.Tier.AllowsLiveRefresh(),
	}
	if perr != nil {
		out.Error = perr.Error()
	}
	return out, nil
}

// probeKey classifies the key, reusing a recent answer for the same key.
func (s *Service) probeKey(key string) (riot.KeyInfo, error) {
	keyState.mu.Lock()
	if keyState.probedKey == key && time.Since(keyState.probedAt) < keyProbeTTL {
		info := keyState.info
		keyState.mu.Unlock()
		return info, nil
	}
	keyState.mu.Unlock()

	// Any region answers the status endpoint; the key's quota is not per-region.
	region, err := riot.LookupRegion("euw1")
	if err != nil {
		return riot.KeyInfo{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	info, perr := s.client().ProbeKey(ctx, region)

	keyState.mu.Lock()
	keyState.info, keyState.probedKey, keyState.probedAt = info, key, time.Now()
	keyState.mu.Unlock()

	// At Info, because "my key does not work" is a question the log has to be able
	// to answer. The quota is included since the tier is inferred from it, and a
	// misread quota is the one way this can be wrong.
	logRiot().Info("classified Riot API key",
		"tier", info.Tier, "appRateLimit", info.AppRateLimit,
		"valid", info.Valid, "liveAllowed", info.Tier.AllowsLiveRefresh(), "err", perr)
	return info, perr
}

// liveRefreshAllowed reports whether the web API may be called for uniqueID now.
//
// Two gates. A development key is never polled: its quota is small, it expires
// daily, and the snapshot it can fill in on request is the point of it. An
// elevated key is polled at most once a minute per account.
func (s *Service) liveRefreshAllowed(uniqueID string, force bool) bool {
	key, err := apiKey()
	if err != nil || strings.TrimSpace(key) == "" {
		return false
	}
	info, _ := s.probeKey(key)
	if !info.Valid {
		return false
	}
	if !info.Tier.AllowsLiveRefresh() {
		// A development key still fills a snapshot when the user asks for one.
		return force
	}

	keyState.mu.Lock()
	defer keyState.mu.Unlock()
	if keyState.lastRefresh == nil {
		keyState.lastRefresh = map[string]time.Time{}
	}
	if !force && time.Since(keyState.lastRefresh[uniqueID]) < liveRefreshInterval {
		return false
	}
	keyState.lastRefresh[uniqueID] = time.Now()
	return true
}

// resetKeyProbeForTest clears the cached classification.
func resetKeyProbeForTest() {
	keyState.mu.Lock()
	keyState.info, keyState.probedKey, keyState.probedAt = riot.KeyInfo{}, "", time.Time{}
	keyState.lastRefresh = nil
	keyState.mu.Unlock()
}
