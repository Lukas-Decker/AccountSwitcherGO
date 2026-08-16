package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A platform's own card layout has to outlive its customisation toggle. Turning
// customisation off and on again should hand back the layout the user built,
// not a blank one, so the toggle can be used to compare against the global
// shape without losing work.
func TestPlatformCardLayoutSurvivesCustomizationBeingTurnedOff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SomePlatformSettings.json")

	// A file written by a user who built a layout and then turned the toggle off.
	seed := map[string]any{
		"AccountCardCustomizationEnabled": false,
		"AccountCard": map[string]any{
			"version": 1,
			"preset":  "custom",
			"custom": map[string]any{
				"minWidth":  150,
				"maxWidth":  180,
				"minHeight": 200,
				"avatarEm":  8,
				"fontScale": 1.2,
				"rows": []any{
					map[string]any{"blocks": []any{map[string]any{"kind": "avatar", "enabled": true}}},
					map[string]any{"blocks": []any{map[string]any{"kind": "displayName", "enabled": true}}},
				},
			},
		},
		"ShowShortNotes": true,
	}
	data, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	var loaded PlatformSettings
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.AccountCard == nil {
		t.Fatal("card config was not read back off disk")
	}
	if loaded.AccountCardCustomizationEnabled {
		t.Fatal("customisation should be off in this fixture")
	}
	if loaded.AccountCard.Custom == nil || loaded.AccountCard.Custom.MinWidth != 150 {
		t.Fatalf("custom layout did not survive the round trip: %+v", loaded.AccountCard)
	}
}

func TestNormalizeAccountCardConfigKeepsWhatItUnderstands(t *testing.T) {
	// A config written by a newer build: one known block, one invented one, and
	// a preset that does not exist.
	in := AccountCardConfig{
		Version:          99,
		Preset:           "enormous",
		Blocks:           map[string]bool{"note": false, "hologram": true},
		Displays:         map[string]string{"lastUsed": "icon", "note": "interpretive-dance"},
		StatusBadgeStyle: "corner",
	}

	out := NormalizeAccountCardConfig(in)

	if out.Preset != "small" {
		t.Fatalf("unknown preset should fall back to small, got %q", out.Preset)
	}
	if _, ok := out.Blocks["hologram"]; ok {
		t.Fatal("unknown block kind should be dropped")
	}
	if v, ok := out.Blocks["note"]; !ok || v {
		t.Fatal("a known block decision should be kept")
	}
	if out.Displays["lastUsed"] != "icon" {
		t.Fatal("a valid display mode should be kept")
	}
	if _, ok := out.Displays["note"]; ok {
		t.Fatal("an invalid display mode should be dropped")
	}
	if out.StatusBadgeStyle != "corner" {
		t.Fatal("a valid badge style should be kept")
	}
	if out.Version != CardConfigVersion {
		t.Fatalf("version should be stamped to the current one, got %d", out.Version)
	}
}

func TestNormalizeCardLayoutWidensAnInvertedWidthRange(t *testing.T) {
	// A max below the min would make the grid track invalid. The user's intent
	// is still readable from the two numbers, so widen rather than reject.
	out := normalizeCardLayout(CardLayout{MinWidth: 200, MaxWidth: 120, MinHeight: 150, AvatarEm: 6, FontScale: 1})
	if out.MinWidth > out.MaxWidth {
		t.Fatalf("expected a usable range, got %d..%d", out.MinWidth, out.MaxWidth)
	}
}
