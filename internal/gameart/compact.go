package gameart

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"account-switcher/internal/crashlog"
	"account-switcher/internal/platform"
)

// CompactResult reports what a sweep reclaimed.
type CompactResult struct {
	// Orphans are files written under a schema nothing can read any more.
	OrphansRemoved int   `json:"orphansRemoved"`
	OrphanBytes    int64 `json:"orphanBytes"`
	// Reencoded are files rebuilt at the size and quality the grid draws.
	Reencoded   int   `json:"reencoded"`
	BytesSaved  int64 `json:"bytesSaved"`
	BytesBefore int64 `json:"bytesBefore"`
	BytesAfter  int64 `json:"bytesAfter"`
}

var compactMu sync.Mutex

// CompactCache reclaims space in the published art directory.
//
// Two things accumulate there. Files written under an older cache schema stay
// on disk forever, because the lookup that would find them no longer recognises
// their names and only a publish for that same game clears them. And art
// published before the size and quality rules existed is still whatever the
// source happened to ship, which for a store capsule is several times what the
// tile needs.
//
// This is disk only: nothing is downloaded, and a file that cannot be improved
// is left exactly as it is.
func CompactCache() (CompactResult, error) {
	compactMu.Lock()
	defer compactMu.Unlock()

	var res CompactResult
	wwwroot, err := platform.WwwrootDir()
	if err != nil {
		return res, err
	}
	root := filepath.Join(wwwroot, "img", "games")
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return res, nil
	}

	err = filepath.Walk(root, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil || fi == nil || fi.IsDir() {
			return nil
		}
		name := fi.Name()
		res.BytesBefore += fi.Size()

		// Reachable art lives in a platform folder under a name that parses to
		// a tier. Anything else is from an earlier layout: the first version
		// published straight to img/games/<appid>.jpg with no platform folder
		// and no tier, and those files are invisible to every lookup the app
		// now makes. They only ever grow.
		inPlatformDir := filepath.Dir(path) != root
		if !inPlatformDir || tierFromFileName(name) == TierNone {
			if os.Remove(path) == nil {
				res.OrphansRemoved++
				res.OrphanBytes += fi.Size()
			}
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil || len(raw) == 0 {
			return nil
		}
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
		out, newExt := normalize(raw, ext)
		if len(out) >= len(raw) {
			res.BytesAfter += fi.Size()
			return nil
		}

		target := path
		if newExt != ext {
			target = strings.TrimSuffix(path, filepath.Ext(path)) + "." + newExt
		}
		if err := os.WriteFile(target, out, 0o644); err != nil {
			res.BytesAfter += fi.Size()
			return nil
		}
		if target != path {
			_ = os.Remove(path)
		}
		res.Reencoded++
		res.BytesSaved += int64(len(raw) - len(out))
		res.BytesAfter += int64(len(out))
		return nil
	})
	if err != nil {
		return res, err
	}

	artLog.Info("compacted the game art cache",
		slog.Int("orphansRemoved", res.OrphansRemoved),
		slog.Int64("orphanBytes", res.OrphanBytes),
		slog.Int("reencoded", res.Reencoded),
		slog.Int64("bytesSaved", res.BytesSaved))
	return res, nil
}

// CompactCacheInBackground runs a sweep without blocking the caller.
//
// Re-encoding a few thousand images takes long enough that doing it on the way
// into the games view would be felt, and none of it changes what is on screen:
// every file keeps its URL unless its format changes, and the view reloads
// anyway on the next pass.
func CompactCacheInBackground() {
	go func() {
		defer crashlog.Capture()
		if _, err := CompactCache(); err != nil {
			artLog.Debug("art cache compaction failed", slog.Any("err", err))
		}
	}()
}
