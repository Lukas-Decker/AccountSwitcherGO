package gamelib

import (
	"context"
	"sort"
	"sync"
	"time"

	"account-switcher/internal/security"
)

// cacheTTL is how long a resolve pass is reused.
//
// A full pass walks every library folder of every launcher and can read a few
// thousand small files, which is too slow to redo every time the view is
// opened. Libraries change when someone installs something, so a few minutes
// of staleness costs nothing, and an explicit refresh bypasses it anyway.
const cacheTTL = 5 * time.Minute

// LibraryDTO is the resolved library as the frontend consumes it.
type LibraryDTO struct {
	Games []Game `json:"games"`
	// Platforms carries the per-platform outcome so the view can explain an
	// empty section: not installed, no per-account records, or an error.
	Platforms []Result `json:"platforms"`
	// ResolvedAt is when the pass ran, so the view can show its age.
	ResolvedAt string `json:"resolvedAt"`
	// UsedNetwork records whether online enrichment actually ran, since a
	// local-only result is legitimately less complete.
	UsedNetwork bool `json:"usedNetwork"`
}

// Service exposes game resolution to the frontend.
type Service struct {
	mu       sync.Mutex
	cache    *LibraryDTO
	cachedAt time.Time
	// cachedNetwork records whether the cached pass used the network, so a
	// request that wants online enrichment is not served a local-only result.
	cachedNetwork bool
}

// NewService returns a game library service.
func NewService() *Service {
	return &Service{}
}

// ServiceName identifies the service to Wails.
func (s *Service) ServiceName() string {
	return "GameLibraryService"
}

// GetGames returns every resolved game across every platform, from cache when
// one is fresh. It resolves from local data only, so it never blocks on the
// network.
func (s *Service) GetGames() (LibraryDTO, error) {
	return s.resolve(context.Background(), false, false)
}

// GetGamesOnline resolves with online enrichment allowed, which for Steam adds
// the games an account owns but has never installed on this machine.
func (s *Service) GetGamesOnline() (LibraryDTO, error) {
	return s.resolve(context.Background(), true, false)
}

// RefreshGames forces a fresh pass, ignoring the cache.
func (s *Service) RefreshGames(allowNetwork bool) (LibraryDTO, error) {
	return s.resolve(context.Background(), allowNetwork, true)
}

// GetPlatformGames resolves one platform, bypassing the aggregate cache. It
// suits a single platform's page, which should not pay for every launcher.
func (s *Service) GetPlatformGames(platformKey string, allowNetwork bool) (Result, error) {
	if err := security.RequireUnlocked(); err != nil {
		return Result{}, err
	}
	res, err := ResolvePlatform(context.Background(), platformKey, allowNetwork)
	if err != nil {
		res.Warnings = append(res.Warnings, err.Error())
	}
	return res, nil
}

// SupportedPlatforms lists the platforms that have a resolver.
func (s *Service) SupportedPlatforms() []string {
	return RegisteredPlatforms()
}

func (s *Service) resolve(ctx context.Context, allowNetwork, force bool) (LibraryDTO, error) {
	// The library is derived from the accounts the switcher manages, so it is
	// behind the same lock as the account lists.
	if err := security.RequireUnlocked(); err != nil {
		return LibraryDTO{}, err
	}

	s.mu.Lock()
	cached := s.cache
	fresh := cached != nil && time.Since(s.cachedAt) < cacheTTL
	// A local-only cache cannot answer a request that wants online enrichment,
	// but an enriched cache is a superset and answers either.
	usable := fresh && (s.cachedNetwork || !allowNetwork)
	s.mu.Unlock()

	if usable && !force {
		return *cached, nil
	}

	results := ResolveAll(ctx, allowNetwork)
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].PlatformKey < results[j].PlatformKey
	})

	lists := make([][]Game, 0, len(results))
	for _, r := range results {
		lists = append(lists, r.Games)
	}
	dto := LibraryDTO{
		Games:       Merge(lists...),
		Platforms:   results,
		ResolvedAt:  time.Now().UTC().Format(time.RFC3339),
		UsedNetwork: allowNetwork,
	}

	s.mu.Lock()
	s.cache = &dto
	s.cachedAt = time.Now()
	s.cachedNetwork = allowNetwork
	s.mu.Unlock()

	return dto, nil
}

// InvalidateCache drops the cached pass, for use after a switch or an install
// that would have changed what is on disk.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	s.cache = nil
	s.cachedAt = time.Time{}
	s.mu.Unlock()
}
