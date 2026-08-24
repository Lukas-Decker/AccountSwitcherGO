// Package gameart resolves cover art for a game and publishes it where the
// webview can load it.
//
// No single source covers a library, and no single source is reliably the best
// one. A launcher's own cache is free to read but often holds only a wide
// banner or a bare wordmark, while the publisher's public CDN has the portrait
// capsule the grid actually wants. So a caller hands over every candidate it
// knows about, each tagged with the shape it will produce, and the chain works
// down from the best shape to the worst rather than from the nearest source to
// the furthest.
//
// Every candidate is tried until one publishes. A file that exists but does not
// decode, or a CDN path that 404s, moves to the next candidate rather than
// ending the tier: those are the ordinary cases, not the exceptional ones.
package gameart

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"account-switcher/internal/crashlog"
	"account-switcher/internal/platform"
	"account-switcher/internal/winutil"

	// Registered so a candidate can be checked by decoding its header rather
	// than by trusting its first few bytes.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// Tier ranks how good a candidate is for the 2:3 tile the games grid draws.
//
// It is deliberately about the artwork's shape rather than where it came from.
// Ranking by source instead would hand a game a wordmark off the local disk
// when the publisher's CDN has its actual cover, which is the wrong answer
// however cheap it was to reach.
type Tier int

const (
	// TierNone means nothing was found.
	TierNone Tier = iota
	// TierIcon is an application icon out of an executable. Square, small, and
	// the last thing to fall back on.
	TierIcon
	// TierLogo is a transparent wordmark. It has no background of its own, so
	// it fills a tile poorly, but it does identify the game.
	TierLogo
	// TierWide is a header, hero, or banner. Real artwork, but it has to be
	// cropped hard to fit a portrait tile.
	TierWide
	// TierPortrait is the library capsule, drawn for exactly this shape.
	TierPortrait
	// TierUserPicked is artwork the user chose themselves, such as a Steam
	// grid override. Their choice outranks anything a publisher shipped.
	TierUserPicked
)

// String renders a tier for logs.
func (t Tier) String() string {
	switch t {
	case TierIcon:
		return "icon"
	case TierLogo:
		return "logo"
	case TierWide:
		return "wide"
	case TierPortrait:
		return "portrait"
	case TierUserPicked:
		return "user"
	default:
		return "none"
	}
}

// Candidate is one place a game's art might be, and the shape it would yield.
//
// Exactly one of Path and URL is set. A candidate with neither is skipped.
type Candidate struct {
	Tier Tier
	// Path is a local file.
	Path string
	// URL is a public image address, only used when the caller allows network.
	URL string
}

// Local reports whether this candidate can be satisfied without the network.
func (c Candidate) Local() bool { return strings.TrimSpace(c.Path) != "" }

const (
	// maxArtBytes caps a download. A portrait capsule is a few hundred
	// kilobytes; anything much larger is not the image that was asked for.
	maxArtBytes = 12 << 20

	// minArtPixels rejects spacers and tracking pixels, which pass a
	// magic-byte check and then render as an empty tile. It sits below the
	// smallest executable icon so those still qualify.
	minArtPixels = 16

	// remoteTimeout bounds one candidate fetch, so a CDN that accepts the
	// connection and then stalls costs one candidate rather than the pass.
	remoteTimeout = 15 * time.Second

	// cacheSchema versions the published filename. Bumping it makes every
	// previously published file unreadable to findCached, so art re-resolves
	// under the current rules instead of a stale tier number being trusted.
	cacheSchema = 2
)

var artLog = slog.Default().With("component", "gameart")

// Request is everything a caller knows that might yield art for one game.
type Request struct {
	PlatformKey string
	GameID      string

	// Candidates may be in any order; the chain sorts them.
	Candidates []Candidate

	// IconExe is an executable whose icon is extracted only when no candidate
	// resolves. It is kept separate because extracting is a shell call rather
	// than a file read, and it is never the answer when real art exists.
	IconExe string

	// Archive supplies further candidates, and is called only once nothing
	// above has resolved.
	//
	// It exists for sources that cost a lookup before they can even offer a
	// URL. Asking one of those for every game in a library would be several
	// thousand requests to answer a question the local cache already answers
	// for most of them, so it is reached for only when the cheap sources have
	// all missed.
	Archive func(ctx context.Context) []Candidate
}

// Result is the outcome for one game.
type Result struct {
	// PublicURL is a wwwroot path the webview can load, or "" when nothing
	// resolved.
	PublicURL string
	Tier      Tier
}

// LocalFile returns a local candidate at the given tier, skipping empty paths.
func LocalFile(tier Tier, path string) Candidate {
	return Candidate{Tier: tier, Path: strings.TrimSpace(path)}
}

// RemoteURL returns a remote candidate at the given tier.
func RemoteURL(tier Tier, url string) Candidate {
	return Candidate{Tier: tier, URL: strings.TrimSpace(url)}
}

// LocalFiles returns candidates for several paths that share a tier.
func LocalFiles(tier Tier, paths ...string) []Candidate {
	out := make([]Candidate, 0, len(paths))
	for _, p := range paths {
		if c := LocalFile(tier, p); c.Path != "" {
			out = append(out, c)
		}
	}
	return out
}

// RemoteURLs returns candidates for several URLs that share a tier.
func RemoteURLs(tier Tier, urls ...string) []Candidate {
	out := make([]Candidate, 0, len(urls))
	for _, u := range urls {
		if c := RemoteURL(tier, u); c.URL != "" {
			out = append(out, c)
		}
	}
	return out
}

// ordered returns the candidates worth trying, best first.
//
// Sorted by tier, then local before remote within a tier: two candidates that
// produce the same shape are interchangeable, so the free one goes first.
// Sorting is stable, so a caller's own order survives among equals, which is
// how Steam's per-app folder is preferred over its flat legacy layout.
func (r Request) ordered(allowNetwork bool) []Candidate {
	out := make([]Candidate, 0, len(r.Candidates))
	for _, c := range r.Candidates {
		if c.Tier <= TierNone {
			continue
		}
		if c.Local() {
			out = append(out, c)
			continue
		}
		if allowNetwork && strings.TrimSpace(c.URL) != "" {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tier != out[j].Tier {
			return out[i].Tier > out[j].Tier
		}
		return out[i].Local() && !out[j].Local()
	})
	return out
}

// bestPossibleTier is the best outcome this request could produce, used to
// decide whether an already published file is worth replacing.
//
// Local candidates are checked against the disk because a path that is not
// there cannot deliver its tier. Remote candidates are taken at face value: the
// only way to know whether a CDN has an image is to ask it, and asking on every
// pass is exactly what the cache exists to avoid.
func (r Request) bestPossibleTier(allowNetwork bool) Tier {
	best := TierNone
	for _, c := range r.ordered(allowNetwork) {
		if c.Local() && !fileHasContent(c.Path) {
			continue
		}
		if c.Tier > best {
			best = c.Tier
		}
	}
	if best == TierNone && fileHasContent(r.IconExe) {
		return TierIcon
	}
	return best
}

func fileHasContent(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}

// Resolve publishes the best art it can find for one game.
//
// It returns an empty result rather than an error when nothing resolves: a game
// with no art anywhere is ordinary, and the view falls back to showing its name.
func Resolve(ctx context.Context, client *http.Client, req Request, allowNetwork bool) Result {
	if strings.TrimSpace(req.PlatformKey) == "" || strings.TrimSpace(req.GameID) == "" {
		return Result{}
	}

	cachedName, cachedTier, hasCached := findCached(req.PlatformKey, req.GameID)
	if hasCached && cachedTier >= req.bestPossibleTier(allowNetwork) {
		return Result{PublicURL: publicURL(req.PlatformKey, cachedName), Tier: cachedTier}
	}

	for _, c := range req.ordered(allowNetwork) {
		// Nothing already published can be improved on by a candidate that
		// would only match it, so stop rather than re-fetching for a draw.
		if hasCached && c.Tier <= cachedTier {
			break
		}
		if c.Local() {
			if res, ok := publishLocal(req, c); ok {
				return res
			}
			continue
		}
		if res, ok := publishRemote(ctx, client, req, c); ok {
			return res
		}
	}

	// The archive is real artwork, so it is worth asking before falling back to
	// an executable's icon. Only when the network is allowed, and only when
	// what is already published is poor enough to be worth improving on.
	if allowNetwork && req.Archive != nil && (!hasCached || cachedTier < TierWide) {
		for _, c := range archiveCandidates(ctx, req, cachedTier, hasCached) {
			if res, ok := publishRemote(ctx, client, req, c); ok {
				return res
			}
		}
	}

	if fileHasContent(req.IconExe) && (!hasCached || cachedTier < TierIcon) {
		if res, ok := publishExeIcon(req); ok {
			return res
		}
	}

	// Nothing better was reachable, so a cached lower tier still beats nothing.
	if hasCached {
		return Result{PublicURL: publicURL(req.PlatformKey, cachedName), Tier: cachedTier}
	}
	return Result{}
}

// archiveCandidates asks the archive and returns what is worth trying, best
// shape first and nothing that would only match what is already published.
func archiveCandidates(ctx context.Context, req Request, cachedTier Tier, hasCached bool) []Candidate {
	found := req.Archive(ctx)
	if len(found) == 0 {
		return nil
	}
	probe := Request{Candidates: found}
	ordered := probe.ordered(true)
	out := ordered[:0]
	for _, c := range ordered {
		if hasCached && c.Tier <= cachedTier {
			break
		}
		out = append(out, c)
	}
	return out
}

// publishLocal copies a local image into wwwroot.
func publishLocal(req Request, c Candidate) (Result, bool) {
	if !fileHasContent(c.Path) {
		return Result{}, false
	}
	raw, err := os.ReadFile(c.Path)
	if err != nil {
		return Result{}, false
	}
	ext, ok := imageExt(raw, "")
	if !ok {
		artLog.Debug("skipping unusable local art", slog.String("path", c.Path))
		return Result{}, false
	}
	return writeArt(req, raw, c.Tier, ext)
}

// publishRemote downloads an image and publishes it.
func publishRemote(ctx context.Context, client *http.Client, req Request, c Candidate) (Result, bool) {
	url := strings.TrimSpace(c.URL)
	// GOG stores some image references protocol-relative, which is not a URL a
	// Go client will accept.
	if strings.HasPrefix(url, "//") {
		url = "https:" + url
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return Result{}, false
	}

	ctx, cancel := context.WithTimeout(ctx, remoteTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, false
	}
	httpReq.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,*/*")
	httpReq.Header.Set("User-Agent", "account-switcher/3 (game art; +https://github.com/Account-Switcher)")

	resp, err := client.Do(httpReq)
	if err != nil {
		artLog.Debug("game art fetch failed", slog.String("url", url), slog.Any("err", err))
		return Result{}, false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// A 404 here is the normal answer for a game the CDN has no artwork
		// for, so this is not worth a warning.
		artLog.Debug("game art not on CDN", slog.String("url", url), slog.Int("status", resp.StatusCode))
		return Result{}, false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxArtBytes))
	if err != nil || len(raw) == 0 {
		return Result{}, false
	}
	ext, ok := imageExt(raw, resp.Header.Get("Content-Type"))
	if !ok {
		artLog.Debug("game art response was not a usable image", slog.String("url", url))
		return Result{}, false
	}
	return writeArt(req, raw, c.Tier, ext)
}

// publishExeIcon extracts the executable's icon as the last resort.
func publishExeIcon(req Request) (Result, bool) {
	dir, err := artDir(req.PlatformKey)
	if err != nil {
		return Result{}, false
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, false
	}
	name := cacheFileName(req.GameID, TierIcon, "png")
	out := filepath.Join(dir, name)
	if err := winutil.ExtractExeIcon(req.IconExe, out); err != nil {
		artLog.Debug("game art exe icon failed",
			slog.String("exe", req.IconExe), slog.Any("err", err))
		return Result{}, false
	}
	if !fileHasContent(out) {
		return Result{}, false
	}
	pruneOtherTiers(dir, req.GameID, name)
	return Result{PublicURL: publicURL(req.PlatformKey, name), Tier: TierIcon}, true
}

// writeArt publishes image bytes and removes the copies from other tiers.
func writeArt(req Request, raw []byte, tier Tier, ext string) (Result, bool) {
	dir, err := artDir(req.PlatformKey)
	if err != nil {
		return Result{}, false
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, false
	}
	// Done once, here, so every later read is of the smaller file rather than
	// the format the source happened to serve.
	raw, ext = transcode(raw, ext)

	name := cacheFileName(req.GameID, tier, ext)
	// Written directly rather than atomically: this is a rebuildable cache, and
	// a torn write is caught by the decode check on the next read.
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		return Result{}, false
	}
	pruneOtherTiers(dir, req.GameID, name)
	return Result{PublicURL: publicURL(req.PlatformKey, name), Tier: tier}, true
}

// imageExt decides the file extension, and rejects anything that is not a
// usable image.
//
// Content-Type and the URL suffix are both routinely wrong: Steam serves some
// capsules as octet-stream, and an expired CDN path answers with an HTML error
// page that would otherwise be saved as a .jpg and render as a broken tile.
// So the bytes decide, and where the format is one Go can read, the header is
// decoded so a truncated file is caught rather than published.
func imageExt(raw []byte, contentType string) (string, bool) {
	ext, ok := extFromMagic(raw)
	if !ok {
		// SVG has no magic number, so it is the one case where the label
		// decides.
		if strings.Contains(strings.ToLower(contentType), "svg") &&
			bytes.Contains(raw[:min(len(raw), 512)], []byte("<svg")) {
			return "svg", true
		}
		return "", false
	}
	if !decodableImage(raw) {
		return "", false
	}
	return ext, true
}

// decodableImage checks the header of formats Go can read.
//
// An unregistered format is accepted on its magic bytes alone: ico and avif
// have no decoder here, and refusing them would throw away a perfectly good
// executable icon.
func decodableImage(raw []byte) bool {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return err == image.ErrFormat
	}
	return cfg.Width >= minArtPixels && cfg.Height >= minArtPixels
}

// extFromMagic identifies the format from its leading bytes.
func extFromMagic(raw []byte) (string, bool) {
	switch {
	case len(raw) >= 3 && raw[0] == 0xFF && raw[1] == 0xD8 && raw[2] == 0xFF:
		return "jpg", true
	case len(raw) >= 8 && bytes.Equal(raw[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "png", true
	case len(raw) >= 6 && (bytes.Equal(raw[:6], []byte("GIF87a")) || bytes.Equal(raw[:6], []byte("GIF89a"))):
		return "gif", true
	case len(raw) >= 12 && bytes.Equal(raw[:4], []byte("RIFF")) && bytes.Equal(raw[8:12], []byte("WEBP")):
		return "webp", true
	case len(raw) >= 12 && bytes.Equal(raw[4:8], []byte("ftyp")) && bytes.Contains(raw[8:12], []byte("avi")):
		return "avif", true
	case len(raw) >= 4 && bytes.Equal(raw[:4], []byte{0x00, 0x00, 0x01, 0x00}):
		return "ico", true
	case len(raw) >= 2 && bytes.Equal(raw[:2], []byte("BM")):
		return "bmp", true
	}
	return "", false
}

// safeSegment turns an arbitrary platform key or game id into a filename.
//
// Game ids range from bare numbers to Epic's opaque names to a platform title
// with spaces and punctuation. Sanitising alone would let two different ids
// collapse onto one file, so anything that had to be changed also carries a
// hash of the original.
func safeSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	var b strings.Builder
	changed := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) && r < 128, unicode.IsDigit(r), r == '-', r == '_':
			b.WriteRune(r)
		default:
			changed = true
		}
	}
	out := b.String()
	if len(out) > 48 {
		out = out[:48]
		changed = true
	}
	if !changed {
		return out
	}
	sum := sha1.Sum([]byte(s))
	suffix := hex.EncodeToString(sum[:])[:8]
	if out == "" {
		return suffix
	}
	return out + "-" + suffix
}

// artDir is where a platform's published art lives inside wwwroot.
func artDir(platformKey string) (string, error) {
	wwwroot, err := platform.WwwrootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(wwwroot, "img", "games", safeSegment(platformKey)), nil
}

// publicURL is the path the webview loads. Absolute so it resolves the same
// from any route the hash router happens to be on.
func publicURL(platformKey, fileName string) string {
	return "/img/games/" + safeSegment(platformKey) + "/" + fileName
}

// cacheFileName encodes the schema and tier into the name so the published file
// describes its own provenance, with no index to keep in step with the
// directory.
func cacheFileName(gameID string, tier Tier, ext string) string {
	return fmt.Sprintf("%s@v%dt%d.%s", safeSegment(gameID), cacheSchema, int(tier), ext)
}

// findCached returns the published art for a game and the tier it came from.
func findCached(platformKey, gameID string) (fileName string, tier Tier, ok bool) {
	dir, err := artDir(platformKey)
	if err != nil {
		return "", TierNone, false
	}
	matches, err := filepath.Glob(filepath.Join(dir, safeSegment(gameID)+"@*.*"))
	if err != nil || len(matches) == 0 {
		return "", TierNone, false
	}
	// More than one tier can be on disk if a write was interrupted; the best
	// one wins and the rest are cleaned up on the next successful publish.
	best := TierNone
	bestName := ""
	for _, m := range matches {
		st, err := os.Stat(m)
		if err != nil || st.IsDir() || st.Size() == 0 {
			continue
		}
		t := tierFromFileName(filepath.Base(m))
		if t > best {
			best, bestName = t, filepath.Base(m)
		}
	}
	if bestName == "" {
		return "", TierNone, false
	}
	return bestName, best, true
}

// tierFromFileName reads the tier back out of a published filename, and
// reports TierNone for a name written under an older schema so it is treated
// as no cache at all and replaced.
func tierFromFileName(name string) Tier {
	at := strings.LastIndex(name, "@")
	if at < 0 {
		return TierNone
	}
	rest := name[at+1:]
	dot := strings.Index(rest, ".")
	if dot <= 0 {
		return TierNone
	}
	marker := rest[:dot]
	prefix := fmt.Sprintf("v%dt", cacheSchema)
	if !strings.HasPrefix(marker, prefix) {
		return TierNone
	}
	n, err := strconv.Atoi(marker[len(prefix):])
	if err != nil || n < 0 || Tier(n) > TierUserPicked {
		return TierNone
	}
	return Tier(n)
}

// pruneOtherTiers deletes the game's art from every tier except the one just
// written, so an upgraded capsule does not leave the old icon behind. It also
// clears names written under an older schema.
func pruneOtherTiers(dir, gameID, keep string) {
	matches, err := filepath.Glob(filepath.Join(dir, safeSegment(gameID)+"@*.*"))
	if err != nil {
		return
	}
	for _, m := range matches {
		if filepath.Base(m) == keep {
			continue
		}
		// Absolute path from Glob, which only ever returns entries inside dir.
		_ = os.Remove(m)
	}
}

// artConcurrency bounds how many games are resolved at once.
//
// The work is a stat and a copy for local art but a round trip for remote, and
// a library of several hundred games would otherwise open that many sockets at
// once. Eight keeps a cold first run brisk without flooding a CDN.
const artConcurrency = 8

// ResolveMany resolves art for a batch of games concurrently and returns the
// results keyed by game id.
func ResolveMany(ctx context.Context, client *http.Client, reqs []Request, allowNetwork bool) map[string]Result {
	out := make(map[string]Result, len(reqs))
	if len(reqs) == 0 {
		return out
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, artConcurrency)

	for _, req := range reqs {
		select {
		case <-ctx.Done():
			return out
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(req Request) {
			defer crashlog.Capture()
			defer wg.Done()
			defer func() { <-sem }()

			res := Resolve(ctx, client, req, allowNetwork)
			if res.PublicURL == "" {
				return
			}
			mu.Lock()
			out[req.GameID] = res
			mu.Unlock()
		}(req)
	}
	wg.Wait()
	return out
}
