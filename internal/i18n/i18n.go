package i18n

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	cacheMu sync.Mutex
	cache   = map[string]map[string]string{}
)

// T returns a localized string from the existing frontend resource JSON files.
// It is intentionally small for native Go surfaces such as the tray menu.
func T(exeDir, language, key string, vars map[string]string) string {
	language = strings.TrimSpace(language)
	if language == "" {
		language = "en-US"
	}
	messages := loadMessages(exeDir, language)
	if language != "en-US" {
		en := loadMessages(exeDir, "en-US")
		merged := make(map[string]string, len(en)+len(messages))
		for k, v := range en {
			merged[k] = v
		}
		for k, v := range messages {
			merged[k] = v
		}
		messages = merged
	}
	template := messages[key]
	if template == "" {
		template = key
	}
	for k, v := range vars {
		template = strings.ReplaceAll(template, "{"+k+"}", v)
	}
	return template
}

func loadMessages(exeDir, language string) map[string]string {
	cacheKey := exeDir + "\x00" + language
	cacheMu.Lock()
	if v, ok := cache[cacheKey]; ok {
		cacheMu.Unlock()
		return v
	}
	cacheMu.Unlock()

	messages := readMessages(exeDir, language)
	cacheMu.Lock()
	cache[cacheKey] = messages
	cacheMu.Unlock()
	return messages
}

func readMessages(exeDir, language string) map[string]string {
	for _, base := range resourceSearchRoots(exeDir) {
		p := filepath.Join(base, "frontend", "src", "Resources", language+".json")
		if messages, ok := readResourceFile(p); ok {
			return messages
		}
		p = filepath.Join(base, "Resources", language+".json")
		if messages, ok := readResourceFile(p); ok {
			return messages
		}
	}
	// Nothing on disk: an installed copy has no source tree beside it, and
	// without this the caller would render raw keys such as "Tray_Exit".
	if messages, ok := readEmbeddedResource(language); ok {
		return messages
	}
	return map[string]string{}
}

var (
	embeddedMu        sync.RWMutex
	embeddedResources fs.FS
	embeddedRoot      string
)

// SetEmbeddedResources supplies the locale files compiled into the binary, used
// when no source tree is present next to the executable. Call once at startup.
func SetEmbeddedResources(files fs.FS, root string) {
	embeddedMu.Lock()
	embeddedResources = files
	embeddedRoot = strings.Trim(strings.ReplaceAll(root, "\\", "/"), "/")
	embeddedMu.Unlock()
}

func readEmbeddedResource(language string) (map[string]string, bool) {
	embeddedMu.RLock()
	files, root := embeddedResources, embeddedRoot
	embeddedMu.RUnlock()
	if files == nil {
		return nil, false
	}
	name := language + ".json"
	if root != "" {
		name = root + "/" + name
	}
	data, err := fs.ReadFile(files, name)
	if err != nil {
		return nil, false
	}
	return parseResourceBytes(data)
}

func resourceSearchRoots(exeDir string) []string {
	seen := map[string]struct{}{}
	var roots []string
	addAncestors := func(start string) {
		start = strings.TrimSpace(start)
		if start == "" {
			return
		}
		dir, err := filepath.Abs(start)
		if err != nil {
			dir = start
		}
		for {
			if _, ok := seen[dir]; !ok {
				seen[dir] = struct{}{}
				roots = append(roots, dir)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return
			}
			dir = parent
		}
	}
	addAncestors(exeDir)
	if wd, err := os.Getwd(); err == nil {
		addAncestors(wd)
	}
	return roots
}

func readResourceFile(path string) (map[string]string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return parseResourceBytes(data)
}

func parseResourceBytes(data []byte) (map[string]string, bool) {
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, false
	}
	return messages, true
}
