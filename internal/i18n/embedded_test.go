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
