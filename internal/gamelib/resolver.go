package gamelib

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"account-switcher/internal/crashlog"
)

// Options tune a resolve pass.
type Options struct {
	// AllowNetwork lets a resolver reach for an online library listing: the
	// games an account owns but has never installed here. That is account data
	// rather than presentation, so it stays opt-in.
	AllowNetwork bool

	// AllowArtwork lets a resolver fetch cover art over the network.
	//
	// Deliberately separate from AllowNetwork. Artwork is images for tiles the
	// user is already looking at, it is cached permanently once fetched, and
	// without it every platform that ships no artwork of its own shows nothing
	// but executable icons forever. Gating it behind the same opt-in as account
	// data meant the whole remote half of the art chain never ran.
	AllowArtwork bool

	// ArtFromCacheOnly makes the art step read only what has already been
	// published, so a pass returns as fast as the library can be listed. The
	// caller is expected to follow it with a full pass and report what changed.
	ArtFromCacheOnly bool
	// KnownAccounts maps account id to display name for the platform being
	// resolved, so a resolver can name owners and can attribute an account-less
	// install when the platform has exactly one account.
	KnownAccounts map[string]string
	// ActiveAccountID is the account currently logged into the platform, used
	// as the inferred owner for launchers that record nothing per account.
	ActiveAccountID string
}

// SingleKnownAccount returns the only known account when there is exactly one,
// which is the only case where inferring an owner is defensible.
func (o Options) SingleKnownAccount() (id, name string, ok bool) {
	if len(o.KnownAccounts) != 1 {
		return "", "", false
	}
	for k, v := range o.KnownAccounts {
		return k, v, true
	}
	return "", "", false
}

// Resolver produces the game list for one platform.
//
// A resolver reports what it can and returns a partial list with a warning
// rather than failing outright: a missing GOG database says nothing about the
// Epic manifests sitting next to it, and one launcher being absent must not
// blank the whole view.
type Resolver interface {
	// PlatformKey matches the key used in Platforms.json and in the account
	// lists, so results line up with the accounts they belong to.
	PlatformKey() string
	// Resolve returns the games this platform knows about.
	Resolve(ctx context.Context, opts Options) (Result, error)
}

// Result is one platform's resolved games plus what went wrong on the way.
type Result struct {
	PlatformKey string   `json:"platformKey"`
	Games       []Game   `json:"games"`
	Warnings    []string `json:"warnings"`
	// Unsupported marks a platform that keeps no discoverable library on this
	// machine, so the UI can say so instead of showing a bare empty list.
	Unsupported bool `json:"unsupported"`
	// DurationMS records how long the pass took, since a slow launcher is the
	// usual reason a refresh feels stuck.
	DurationMS int64 `json:"durationMs"`
	// ActiveAccountID is the account currently signed in to this platform.
	//
	// The view needs it to decide what a click means: if the signed-in account
	// already has the game there is nothing to switch, so it just launches.
	ActiveAccountID string `json:"activeAccountId"`
	// Accounts lists every account on the platform, so a game can be started on
	// one that no source recorded as an owner. Free-to-play titles are the
	// common case: nothing marks them owned, but any account can play them.
	Accounts []AccountRef `json:"accounts"`
}

// AccountRef is one switchable account.
type AccountRef struct {
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
}

// ResolverFunc adapts a plain function to [Resolver].
type ResolverFunc struct {
	Key string
	Fn  func(ctx context.Context, opts Options) (Result, error)
}

// PlatformKey implements [Resolver].
func (r ResolverFunc) PlatformKey() string { return r.Key }

// Resolve implements [Resolver].
func (r ResolverFunc) Resolve(ctx context.Context, opts Options) (Result, error) {
	return r.Fn(ctx, opts)
}

var (
	registryMu sync.RWMutex
	registered = map[string]Resolver{}
)

// Register adds a platform resolver. Registering the same key twice replaces
// the earlier one, so a build can swap in a better resolver without unhooking
// the old registration first.
func Register(r Resolver) {
	if r == nil {
		return
	}
	key := strings.TrimSpace(r.PlatformKey())
	if key == "" {
		return
	}
	registryMu.Lock()
	registered[key] = r
	registryMu.Unlock()
}

// ResolverFor returns the resolver registered for a platform.
func ResolverFor(platformKey string) (Resolver, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	r, ok := registered[strings.TrimSpace(platformKey)]
	return r, ok
}

// RegisteredPlatforms lists the platforms that have a resolver, sorted.
func RegisteredPlatforms() []string {
	registryMu.RLock()
	keys := make([]string, 0, len(registered))
	for k := range registered {
		keys = append(keys, k)
	}
	registryMu.RUnlock()
	sort.Strings(keys)
	return keys
}

// OptionsForPlatform supplies the per-platform account context a resolver needs.
// It is set once at startup by the layer that can read account lists, which
// gamelib itself must not import.
var OptionsForPlatform func(platformKey string) Options

func optionsFor(platformKey string, allowNetwork, allowArtwork, cacheOnlyArt bool) Options {
	var opts Options
	if OptionsForPlatform != nil {
		opts = OptionsForPlatform(platformKey)
	}
	opts.AllowNetwork = allowNetwork
	opts.AllowArtwork = allowArtwork
	opts.ArtFromCacheOnly = cacheOnlyArt
	return opts
}

// ResolvePlatform runs one platform's resolver.
func ResolvePlatform(ctx context.Context, platformKey string, allowNetwork, allowArtwork, cacheOnlyArt bool) (Result, error) {
	r, ok := ResolverFor(platformKey)
	if !ok {
		return Result{PlatformKey: platformKey, Unsupported: true, Games: []Game{}}, nil
	}
	started := time.Now()
	opts := optionsFor(platformKey, allowNetwork, allowArtwork, cacheOnlyArt)
	res, err := r.Resolve(ctx, opts)
	res.PlatformKey = platformKey
	res.ActiveAccountID = opts.ActiveAccountID
	res.Accounts = accountRefs(opts.KnownAccounts)
	// The user's own choices are applied last, over whatever the sources said.
	res.Games = applyPrefs(platformKey, res.Games)
	res.DurationMS = time.Since(started).Milliseconds()
	if res.Games == nil {
		res.Games = []Game{}
	}
	return res, err
}

// accountRefs turns the account map into a stable, ordered list for the view.
func accountRefs(known map[string]string) []AccountRef {
	out := make([]AccountRef, 0, len(known))
	for id, name := range known {
		out = append(out, AccountRef{AccountID: id, AccountName: name})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !strings.EqualFold(out[i].AccountName, out[j].AccountName) {
			return strings.ToLower(out[i].AccountName) < strings.ToLower(out[j].AccountName)
		}
		return out[i].AccountID < out[j].AccountID
	})
	return out
}

// ResolveAll runs every registered resolver concurrently.
//
// One platform's failure is recorded as a warning on that platform's result and
// never aborts the pass, because the common case is several launchers installed
// and one of them broken or mid-update.
func ResolveAll(ctx context.Context, allowNetwork, allowArtwork, cacheOnlyArt bool) []Result {
	keys := RegisteredPlatforms()
	results := make([]Result, len(keys))

	var wg sync.WaitGroup
	for i, key := range keys {
		wg.Add(1)
		go func(i int, key string) {
			defer crashlog.Capture()
			defer wg.Done()
			res, err := ResolvePlatform(ctx, key, allowNetwork, allowArtwork, cacheOnlyArt)
			if err != nil {
				res.Warnings = append(res.Warnings, err.Error())
			}
			results[i] = res
		}(i, key)
	}
	wg.Wait()
	return results
}
