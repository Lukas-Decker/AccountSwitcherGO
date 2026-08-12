package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The seed is written once and then never revisited by anything else: this fork
// has the remote refresh switched off, so a descriptor fix shipped in a new build
// reaches an existing install through this path or not at all.
func TestSeedRefreshesAnOlderPlatformsFile(t *testing.T) {
	previous := append([]byte(nil), embeddedPlatformsJSON...)
	defer SetEmbeddedPlatformsJSON(previous)
	SetEmbeddedPlatformsJSON([]byte(`{"Version":"4.0.3","Platforms":{"Steam":{"Identifiers":["s","steam"]}}}`))

	dir := t.TempDir()
	ResetPathSingletonsForTest(dir)
	dest := filepath.Join(UserDataDir(dir), "Platforms.json")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// The version style the older builds shipped, which no longer parses as semver
	// and so must count as older than anything the current build carries. Steam is
	// deliberately stale here and Ubisoft exists only on disk.
	stale := []byte(`{"Version":"2025-11-20_01","Platforms":` +
		`{"Steam":{"Identifiers":["old"]},"Ubisoft":{"Identifiers":["u"]}}}`)
	if err := os.WriteFile(dest, stale, 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}
	if err := seedEmbeddedPlatforms(dir); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var out struct {
		Version   string                     `json:"Version"`
		Platforms map[string]json.RawMessage `json:"Platforms"`
	}
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Version != "4.0.3" {
		t.Errorf("version was not stamped forward: %q", out.Version)
	}
	if !strings.Contains(string(out.Platforms["Steam"]), "steam") {
		t.Errorf("shipped descriptor did not win: %s", out.Platforms["Steam"])
	}
	// A platform this build does not carry must survive, or the accounts saved
	// under it drop out of the list.
	if _, ok := out.Platforms["Ubisoft"]; !ok {
		t.Error("a platform present only on disk was dropped")
	}

	// Now that it matches the build, seeding again must not rewrite it.
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if err := seedEmbeddedPlatforms(dir); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	after, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("restat: %v", err)
	}
	if !after.ModTime().Equal(info.ModTime()) {
		t.Error("an up-to-date Platforms.json was rewritten")
	}
}
