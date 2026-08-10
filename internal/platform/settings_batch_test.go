package platform

import (
	"encoding/json"
	"testing"
)

func TestApplySettingsBatchUpdateOfflineDisablesDiscord(t *testing.T) {
	on := true
	settings := AppSettings{DiscordRpc: true}

	effects := applySettingsBatchUpdate(&settings, SettingsBatchUpdate{OfflineMode: &on})

	if !settings.OfflineMode || settings.DiscordRpc {
		t.Fatalf("offline update left settings inconsistent: %#v", settings)
	}
	if effects.offlineMode == nil || !*effects.offlineMode || !effects.discordPresenceRefresh || !effects.dirty {
		t.Fatalf("offline update effects = %#v", effects)
	}
}

func TestApplySettingsBatchUpdateSanitizesThemeAndAccent(t *testing.T) {
	theme := "New_Theme"
	badAccent := "#not-ok"
	settings := AppSettings{
		Theme:             "Old",
		ThemeAccentPreset: "windows",
		ThemeAccentCustom: "#112233",
	}

	applySettingsBatchUpdate(&settings, SettingsBatchUpdate{
		Theme:             &theme,
		ThemeAccentCustom: &badAccent,
	})

	if settings.Theme != "New_Theme" {
		t.Fatalf("Theme = %q, want New_Theme", settings.Theme)
	}
	if settings.ThemeAccentPreset != "" || settings.ThemeAccentCustom != "" {
		t.Fatalf("accent fields = (%q, %q), want both cleared", settings.ThemeAccentPreset, settings.ThemeAccentCustom)
	}
}

func TestControllerSupportDefaultsOnWhenMissing(t *testing.T) {
	settings := AppSettings{}

	normalizeAppSettingsDefaults(&settings, map[string]json.RawMessage{})

	if !settings.ControllerSupportEnabled {
		t.Fatal("ControllerSupportEnabled should default to true when the key is absent")
	}
}

func TestControllerSupportPreservesExplicitFalse(t *testing.T) {
	settings := AppSettings{ControllerSupportEnabled: false}

	normalizeAppSettingsDefaults(&settings, map[string]json.RawMessage{
		"controllerSupportEnabled": json.RawMessage("false"),
	})

	if settings.ControllerSupportEnabled {
		t.Fatal("ControllerSupportEnabled should preserve explicit false")
	}
}

func TestApplySettingsBatchUpdateControllerSupport(t *testing.T) {
	off := false
	settings := AppSettings{ControllerSupportEnabled: true}

	effects := applySettingsBatchUpdate(&settings, SettingsBatchUpdate{ControllerSupportEnabled: &off})

	if settings.ControllerSupportEnabled {
		t.Fatal("ControllerSupportEnabled should be false")
	}
	if !effects.dirty {
		t.Fatal("controller support update should mark settings dirty")
	}
	if effects.controllerSupport == nil || *effects.controllerSupport {
		t.Fatalf("controller support effect = %#v, want explicit false", effects.controllerSupport)
	}
}

func TestNormalizeCommandPaletteHotkey(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "default for empty", value: "", want: "Ctrl+K"},
		{name: "legacy control alias", value: "Control + p", want: "Ctrl+P"},
		{name: "multiple modifiers", value: "ctrl+shift+k", want: "Ctrl+Shift+K"},
		{name: "function key", value: "Alt+F4", want: "Alt+F4"},
		{name: "single key rejected", value: "K", want: "Ctrl+K"},
		{name: "escape rejected", value: "Ctrl+Escape", want: "Ctrl+K"},
		{name: "duplicate modifier rejected", value: "Ctrl+Control+K", want: "Ctrl+K"},
		{name: "extra key rejected", value: "Ctrl+K+P", want: "Ctrl+K"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCommandPaletteHotkey(tt.value); got != tt.want {
				t.Fatalf("normalizeCommandPaletteHotkey(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
