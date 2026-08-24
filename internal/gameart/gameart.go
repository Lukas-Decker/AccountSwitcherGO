// Package gameart resolves cover art for a game and publishes it where the
// webview can load it.
//
// No single source covers a library. Steam caches capsules for games it has
// shown in the client but not for the rest; GOG keeps image URLs in its
// database and nothing on disk; Epic, Ubisoft and Rockstar keep no artwork at
// all and leave only an executable to take an icon from. So a caller hands over
// every candidate it knows about and this picks the best one that actually
// resolves, remembering which tier it came from so a later pass can upgrade a
// placeholder once a better source becomes reachable.
package gameart

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"account-switcher/internal/crashlog"
	"account-switcher/internal/platform"
	"account-switcher/internal/winutil"
)

// Tier ranks how good a source is. Art is only replaced by a strictly better
// tier, so an exe icon fetched while offline is upgraded to a real capsule on
// the next pass that can reach the network, and never the other way round.
type Tier int

const (
	// TierNone means nothing was found.
	TierNone Tier = iota
	// TierExeIcon is an application icon pulled out of the game's executable.
	// Square, low resolution, and the last resort before showing no art.
	TierExeIcon
	// TierRemote is publisher artwork from the platform's own public CDN.
	TierRemote
	// TierLocal is artwork the launcher already cached on this machine, which
	// is the same image as the CDN copy but free and available offline.
	TierLocal
	// TierUserPicked is artwork the user set themselves, such as a Steam grid
	// override. Their choice outranks anything a publisher shipped.
	TierUserPicked
)

// maxArtBytes caps a download. A portrait capsule is a few hundred kilobytes;
// anything much larger is not the image that was asked for.
const maxArtBytes = 12 << 20

// artLog is the shared logger for this package.
var artLog = slog.Default().With("component", "gameart")

// Request is everything a caller knows that might yield art for one game.
// Every field is optional, and empty candidates are skipped rather than
// treated as failures.
type Request struct {
	PlatformKey string
	GameID      string

	// UserFiles are local paths the user chose, tried before anything else.
	UserFiles []string
	// LocalFiles are local paths the launcher wrote, tried in order.
	LocalFiles []string
	// RemoteURLs are public image URLs, tried in order, only when the caller
	// allows network access.
	RemoteURLs []string
	// IconExe is an executable to extract an icon from when nothing else
	// resolves.
	IconExe string
}

// Result is the outcome for one game.
type Result struct {
	// PublicURL is a wwwroot path the webview can load, or "" when nothing
	// resolved.
	PublicURL string
	Tier      Tier
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

// cacheFileName encodes the tier into the name so the published file describes
// its own provenance, with no index to keep in step with the directory.
func cacheFileName(gameID string, tier Tier, ext string) string {
	return fmt.Sprintf("%s@%d.%s", safeSegment(gameID), int(tier), ext)
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
	n, err := strconv.Atoi(rest[:dot])
	if err != nil || n < 0 {
		return TierNone
	}
	return Tier(n)
}

// bestAvailableTier reports the best tier this request could possibly produce,
// so a cached file that already matches it can be returned without touching the
// disk or the network again.
func (r Request) bestAvailableTier(allowNetwork bool) Tier {
	if firstExistingFile(r.UserFiles) != "" {
		return TierUserPicked
	}
	if firstExistingFile(r.LocalFiles) != "" {
		return TierLocal
	}
	if allowNetwork && len(r.RemoteURLs) > 0 {
		return TierRemote
	}
	if fileHasContent(r.IconExe) {
		return TierExeIcon
	}
	return TierNone
}

func firstExistingFile(paths []string) string {
	for _, p := range paths {
		if fileHasContent(p) {
			return p
		}
	}
	return ""
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
// with no art anywhere is ordinary, and the view already falls back to showing
// its name.
func Resolve(ctx context.Context, client *http.Client, req Request, allowNetwork bool) Result {
	if strings.TrimSpace(req.PlatformKey) == "" || strings.TrimSpace(req.GameID) == "" {
		return Result{}
	}

	cachedName, cachedTier, hasCached := findCached(req.PlatformKey, req.GameID)
	want := req.bestAvailableTier(allowNetwork)
	if hasCached && cachedTier >= want {
		return Result{PublicURL: publicURL(req.PlatformKey, cachedName), Tier: cachedTier}
	}

	// Tried best first. A remote fetch can 404 for a delisted app even though
	// the URL was well formed, so a failure at one tier falls through to the
	// next rather than giving up.
	if src := firstExistingFile(req.UserFiles); src != "" {
		if res, ok := publishLocal(req, src, TierUserPicked); ok {
			return res
		}
	}
	if src := firstExistingFile(req.LocalFiles); src != "" {
		if res, ok := publishLocal(req, src, TierLocal); ok {
			return res
		}
	}
	if allowNetwork {
		for _, url := range req.RemoteURLs {
			if res, ok := publishRemote(ctx, client, req, url); ok {
				return res
			}
		}
	}
	if fileHasContent(req.IconExe) {
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

// publishLocal copies a local image into wwwroot.
func publishLocal(req Request, src string, tier Tier) (Result, bool) {
	raw, err := os.ReadFile(src)
	if err != nil {
		return Result{}, false
	}
	ext, ok := imageExt(raw, "")
	if !ok {
		return Result{}, false
	}
	return writeArt(req, raw, tier, ext)
}

// publishRemote downloads an image and publishes it.
func publishRemote(ctx context.Context, client *http.Client, req Request, url string) (Result, bool) {
	url = strings.TrimSpace(url)
	if url == "" {
		return Result{}, false
	}
	// GOG stores some image references protocol-relative, which is not a URL a
	// Go client will accept.
	if strings.HasPrefix(url, "//") {
		url = "https:" + url
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return Result{}, false
	}

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
		// A 404 here is the normal answer for a game the CDN has no capsule
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
		// Some CDNs answer a missing asset with an HTML error page and a 200.
		artLog.Debug("game art response was not an image", slog.String("url", url))
		return Result{}, false
	}
	return writeArt(req, raw, TierRemote, ext)
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
	name := cacheFileName(req.GameID, TierExeIcon, "png")
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
	return Result{PublicURL: publicURL(req.PlatformKey, name), Tier: TierExeIcon}, true
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
	name := cacheFileName(req.GameID, tier, ext)
	// Written directly rather than atomically: this is a rebuildable cache, and
	// a torn write is caught by the magic-byte check on the next read.
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		return Result{}, false
	}
	pruneOtherTiers(dir, req.GameID, name)
	return Result{PublicURL: publicURL(req.PlatformKey, name), Tier: tier}, true
}

// pruneOtherTiers deletes the game's art from every tier except the one just
// written, so an upgraded capsule does not leave the old icon behind.
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

// imageExt decides the file extension, trusting the bytes over the label.
//
// Content-Type and the URL suffix are both routinely wrong: Steam serves some
// capsules as octet-stream, and an expired CDN path answers with an HTML error
// page that would otherwise be saved as a .jpg and render as a broken tile.
func imageExt(raw []byte, contentType string) (string, bool) {
	if ext, ok := extFromMagic(raw); ok {
		return ext, true
	}
	// SVG has no magic number, so it is the one case where the label decides.
	if strings.Contains(strings.ToLower(contentType), "svg") && bytes.Contains(raw[:min(len(raw), 512)], []byte("<svg")) {
		return "svg", true
	}
	return "", false
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
