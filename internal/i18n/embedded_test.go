package i18n

import (
	"testing"
	"testing/fstest"
)

// An installed copy has no frontend/src/Resources beside the exe. Before the
// embedded fallback the tray rendered its keys, so "Exit" came out as
// "Tray_Exit".
func TestTFallsBackToEmbeddedWhenNoSourceTree(t *testing.T) {
	resetForTest(t)

	SetEmbeddedResources(fstest.MapFS{
		"res/en-US.json": &fstest.MapFile{Data: []byte(`{"Tray_Exit":"Exit"}`)},
	}, "res")

	// A directory with no resource files anywhere above it.
	if got := T(t.TempDir(), "en-US", "Tray_Exit", nil); got != "Exit" {
		t.Fatalf("T = %q, want %q (the key leaked through)", got, "Exit")
	}
}

// Uses a locale code that exists only in the embedded set, so the assertion
// cannot be answered by the repository's own Resources directory, which the
// disk search finds through the working directory when tests run in-tree.
func TestTPrefersTranslationAndKeepsVarSubstitution(t *testing.T) {
	resetForTest(t)

	SetEmbeddedResources(fstest.MapFS{
		"res/zz-ZZ.json": &fstest.MapFile{Data: []byte(`{"Tray_Exit":"Beenden","Tray_Switch":"Wechsel zu {account}"}`)},
	}, "res")

	dir := t.TempDir()
	if got := T(dir, "zz-ZZ", "Tray_Exit", nil); got != "Beenden" {
		t.Fatalf("translated key = %q, want %q", got, "Beenden")
	}
	if got := T(dir, "zz-ZZ", "Tray_Switch", map[string]string{"account": "Bob"}); got != "Wechsel zu Bob" {
		t.Fatalf("substituted key = %q, want %q", got, "Wechsel zu Bob")
	}
}

func resetForTest(t *testing.T) {
	t.Helper()
	cacheMu.Lock()
	cache = map[string]map[string]string{}
	cacheMu.Unlock()
	t.Cleanup(func() {
		SetEmbeddedResources(nil, "")
		cacheMu.Lock()
		cache = map[string]map[string]string{}
		cacheMu.Unlock()
	})
}
