package app

import (
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"account-switcher/internal/platform"
)

// embeddedDistRoot finds where the built frontend sits inside the embedded FS.
// The embed directive keeps the source path, so files arrive as
// frontend/dist/... rather than at the root.
func embeddedDistRoot(embedded fs.FS) string {
	if _, err := fs.Stat(embedded, "frontend/dist"); err == nil {
		return "frontend/dist"
	}
	return "."
}

// pruneStaleWwwrootAssets deletes files from the on-disk wwwroot that the app
// also ships.
//
// The asset handler serves wwwroot before the embedded copy, so a file left
// there by an older version wins forever: an install that carried its data
// folder across kept showing the previous logo no matter how many times the
// real one was rebuilt. Anything the app ships is owned by the app, so the disk
// copy is removed and the embedded one takes over.
//
// User content is untouched: only paths that exist in the build are considered,
// so backgrounds, profile images, shortcut icons and game art all stay put.
func pruneStaleWwwrootAssets(embedded fs.FS) {
	wwwroot, err := platform.WwwrootDir()
	if err != nil {
		return
	}
	if st, err := os.Stat(wwwroot); err != nil || !st.IsDir() {
		return
	}

	root := embeddedDistRoot(embedded)
	var removed int
	err = fs.WalkDir(embedded, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, root)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		// index.html and the hashed asset bundles are new on every build and were
		// never copied to wwwroot; skip the walk's bulk cheaply.
		if !isAppOwnedWwwrootAsset(rel) {
			return nil
		}
		diskPath := filepath.Join(wwwroot, filepath.FromSlash(rel))
		st, err := os.Stat(diskPath)
		if err != nil || st.IsDir() {
			return nil
		}
		if err := os.Remove(diskPath); err != nil {
			slog.Debug("wwwroot prune: could not remove stale asset", "path", diskPath, "err", err)
			return nil
		}
		removed++
		return nil
	})
	if err != nil {
		slog.Debug("wwwroot prune: walk failed", "err", err)
	}
	if removed > 0 {
		slog.Info("wwwroot prune: removed stale copies of shipped assets", "count", removed)
	}
}

// isAppOwnedWwwrootAsset reports whether a shipped path is one that older
// versions copied into wwwroot, and so may be sitting there stale.
func isAppOwnedWwwrootAsset(rel string) bool {
	rel = path.Clean(rel)
	switch {
	case strings.HasPrefix(rel, "img/"):
		// img/profiles, img/shortcuts, img/games and img/gs hold downloaded or
		// user-supplied pictures. The app ships nothing into them, so they never
		// match a shipped path anyway, but skipping keeps the intent clear.
		for _, userDir := range []string{"img/profiles/", "img/shortcuts/", "img/games/", "img/gs/"} {
			if strings.HasPrefix(rel, userDir) {
				return false
			}
		}
		return true
	case rel == "wails.png", rel == "svelte.svg":
		return true
	default:
		return false
	}
}
