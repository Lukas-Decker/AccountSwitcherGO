package basic

import (
	"os"
	"path/filepath"
	"testing"
)

// A missed id file is worse than an error: ensureUniqueIDOnSave mints a fresh id
// when it cannot find one, which detaches a saved account from its session.
func TestResolveIDFilePathAdoptsTheLegacyName(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, idFileName)
	legacy := filepath.Join(dir, legacyIDFileName)

	// Only the legacy name present: adopted, and the value is preserved.
	if err := os.WriteFile(legacy, []byte("abc123"), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	if got := resolveIDFilePath(current); got != current {
		t.Fatalf("resolve = %q, want the current name", got)
	}
	body, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read adopted: %v", err)
	}
	if string(body) != "abc123" {
		t.Errorf("adopted file holds %q, want abc123", body)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("legacy file survived the rename")
	}

	// Both present: the current name wins and the legacy one is left alone, so a
	// stale duplicate cannot displace the real identity.
	if err := os.WriteFile(legacy, []byte("stale"), 0o644); err != nil {
		t.Fatalf("rewrite legacy: %v", err)
	}
	if got := resolveIDFilePath(current); got != current {
		t.Errorf("resolve = %q, want the current name", got)
	}
	if body, _ := os.ReadFile(current); string(body) != "abc123" {
		t.Errorf("current file was overwritten: %q", body)
	}

	// Neither present: the current path comes back, so the caller creates it under
	// the new name rather than resurrecting the old one.
	empty := t.TempDir()
	want := filepath.Join(empty, idFileName)
	if got := resolveIDFilePath(want); got != want {
		t.Errorf("resolve on empty dir = %q, want %q", got, want)
	}

	// A path that is not an id file is returned untouched.
	other := filepath.Join(dir, "settings.yaml")
	if got := resolveIDFilePath(other); got != other {
		t.Errorf("resolve on a non-id file = %q", got)
	}
}
