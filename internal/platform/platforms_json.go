package platform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"account-switcher/internal/updatecheck"
)

var (
	platformsJSONCache struct {
		mu     sync.RWMutex
		exeDir string
		data   []byte
		loaded bool
	}
)

func invalidatePlatformsJSONCache() {
	platformsJSONCache.mu.Lock()
	platformsJSONCache.loaded = false
	platformsJSONCache.data = nil
	platformsJSONCache.mu.Unlock()
}

// LoadPlatformsJSON returns the effective platforms configuration: the base
// Platforms.json (see [ResolvePlatformsJSONPath]) after optional merge with
// {UserDataDir}/Platforms.custom.json. Matching platform keys in the custom file
// replace the base entry; new keys are added. Custom file must be valid JSON with
// a top-level "Platforms" object.
//
// When the default base Platforms.json is missing under the user data folder,
// it is created from the embedded catalog (first run). An existing file is left
// unchanged so a copy applied via the UI persists until you use Restore default.
//
// The returned slice is cached internally and shared across callers; it must be
// treated as read-only. Callers that need to mutate the bytes should make a copy.
func LoadPlatformsJSON(exeDir string) ([]byte, error) {
	exeDir = filepath.Clean(exeDir)

	platformsJSONCache.mu.RLock()
	if platformsJSONCache.loaded && platformsJSONCache.exeDir == exeDir {
		data := platformsJSONCache.data
		platformsJSONCache.mu.RUnlock()
		return data, nil
	}
	platformsJSONCache.mu.RUnlock()

	if err := seedEmbeddedPlatforms(exeDir); err != nil {
		return nil, err
	}
	s, err := loadSettings(exeDir)
	if err != nil {
		return nil, err
	}
	basePath := resolvePlatformsPath(exeDir, s)
	base, err := os.ReadFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", basePath, err)
	}
	if s.PlatformsJSONPath == "" {
		if merged, changed, err := addEmbeddedSteamPlatform(base, embeddedPlatformsJSON); err != nil {
			return nil, fmt.Errorf("restore embedded Steam platform: %w", err)
		} else if changed {
			if err := atomicWriteBytes(basePath, merged, 0o644); err != nil {
				return nil, fmt.Errorf("write %s: %w", basePath, err)
			}
			base = merged
		}
	}
	customPath := filepath.Join(UserDataDir(exeDir), "Platforms.custom.json")
	custom, err := os.ReadFile(customPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			platformsJSONCache.mu.Lock()
			platformsJSONCache.exeDir = exeDir
			platformsJSONCache.data = bytes.Clone(base)
			platformsJSONCache.loaded = true
			platformsJSONCache.mu.Unlock()
			return platformsJSONCache.data, nil
		}
		return nil, fmt.Errorf("read %s: %w", customPath, err)
	}
	out, err := mergePlatformsJSON(base, custom)
	if err != nil {
		return nil, fmt.Errorf("merge Platforms.custom.json: %w", err)
	}
	platformsJSONCache.mu.Lock()
	platformsJSONCache.exeDir = exeDir
	platformsJSONCache.data = bytes.Clone(out)
	platformsJSONCache.loaded = true
	platformsJSONCache.mu.Unlock()
	return platformsJSONCache.data, nil
}

func seedEmbeddedPlatforms(exeDir string) error {
	if len(embeddedPlatformsJSON) == 0 {
		return nil
	}
	ud := UserDataDir(exeDir)
	if err := os.MkdirAll(ud, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(ud, "Platforms.json")
	if st, err := os.Stat(dest); err == nil && !st.IsDir() {
		return refreshSeededPlatforms(dest)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return atomicWriteBytes(dest, bytes.Clone(embeddedPlatformsJSON), 0o644)
}

// refreshSeededPlatforms brings an already-seeded Platforms.json up to the
// descriptors this build ships.
//
// Without it the seed is written once and then frozen: the only other refresh
// path is the remote check, which this fork has switched off, so a descriptor fix
// shipped in a new build could never reach a machine that had already run an
// older one.
//
// Merged rather than replaced, because the two sides are not the same set: a
// platform the local file has and this build does not would otherwise vanish,
// taking the accounts saved under it out of the list. The shipped descriptors win
// where both define a platform, and everything else is left alone. A custom
// PlatformsJSONPath is unaffected, since the file written here is not the one
// that gets read in that case.
func refreshSeededPlatforms(dest string) error {
	localRaw, err := os.ReadFile(dest)
	if err != nil {
		return err
	}
	embeddedVer, err := updatecheck.ParsePlatformsJSONVersion(embeddedPlatformsJSON)
	if err != nil {
		// Nothing to compare against; leave the local file as it is.
		return nil
	}
	localVer, err := updatecheck.ParsePlatformsJSONVersion(localRaw)
	if err != nil {
		localVer = ""
	}
	if !updatecheck.IsVersionNewer(embeddedVer, localVer) {
		return nil
	}

	merged, err := mergePlatformsJSON(localRaw, embeddedPlatformsJSON)
	if err != nil {
		return err
	}
	// Carry the shipped version across, or the same refresh runs on every launch.
	stamped, err := setPlatformsJSONVersion(merged, embeddedVer)
	if err != nil {
		return err
	}
	return atomicWriteBytes(dest, stamped, 0o644)
}

func setPlatformsJSONVersion(raw []byte, version string) ([]byte, error) {
	var f platformsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	f.Version = version
	return json.Marshal(f)
}

func mergePlatformsJSON(base, overlay []byte) ([]byte, error) {
	var main, over platformsFile
	if err := json.Unmarshal(base, &main); err != nil {
		return nil, err
	}
	if main.Platforms == nil {
		main.Platforms = map[string]json.RawMessage{}
	}
	if err := json.Unmarshal(overlay, &over); err != nil {
		return nil, err
	}
	if over.Platforms == nil {
		return base, nil
	}
	for k, v := range over.Platforms {
		main.Platforms[k] = v
	}
	return json.Marshal(main)
}
