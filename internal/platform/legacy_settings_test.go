//go:build envtests

// Off by default. These tests redirect %APPDATA% and %USERPROFILE% or read
// the running League Client, so they reach outside the package and are only
// as isolated as their own setup. Run them with: go test -tags envtests ./...

package platform

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLegacyWindowSettingsMigratesPlatforms(t *testing.T) {
	setTestAppData(t)
	exeDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	userDataDir, err := DefaultUserDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(userDataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	legacySettings := []byte(`{
  "DisabledPlatforms": [],
  "EnabledBasicPlatforms": ["e"]
}`)
	if err := os.WriteFile(filepath.Join(userDataDir, legacyWindowSettingsFileName), legacySettings, 0o644); err != nil {
		t.Fatal(err)
	}
	legacyCatalog := []byte(`{"Version":"2025-11-09_00","Platforms":{
  "Epic Games":{"Identifiers":["e","epic"]},
  "GOG Galaxy":{"Identifiers":["g","gog"]}
}}`)
	if err := os.WriteFile(filepath.Join(userDataDir, "Platforms.json"), legacyCatalog, 0o644); err != nil {
		t.Fatal(err)
	}
	previousEmbedded := append([]byte(nil), embeddedPlatformsJSON...)
	t.Cleanup(func() { SetEmbeddedPlatformsJSON(previousEmbedded) })
	SetEmbeddedPlatformsJSON([]byte(`{"Version":"4.0.2","Platforms":{"Steam":{"Identifiers":["s","steam","valve"]}}}`))

	ResetPathSingletonsForTest(exeDir)
	if err := InitDataPaths(exeDir); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadAppSettings(exeDir)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(settings.DisabledPlatforms, "Steam") {
		t.Fatalf("legacy enabled Steam was disabled: %v", settings.DisabledPlatforms)
	}
	if slices.Contains(settings.DisabledPlatforms, "Epic Games") {
		t.Fatalf("legacy enabled basic platform was disabled: %v", settings.DisabledPlatforms)
	}
	if !slices.Contains(settings.DisabledPlatforms, "GOG Galaxy") {
		t.Fatalf("legacy inactive basic platform was enabled: %v", settings.DisabledPlatforms)
	}

	loadedCatalog, err := LoadPlatformsJSON(exeDir)
	if err != nil {
		t.Fatal(err)
	}
	names, err := parsePlatformNames(loadedCatalog)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(names, "Steam") {
		t.Fatalf("Steam descriptor was not restored: %v", names)
	}
	if _, err := os.Stat(filepath.Join(userDataDir, legacyWindowSettingsFileName)); err != nil {
		t.Fatalf("legacy settings should remain available for rollback: %v", err)
	}
}

func TestExistingSettingsTakePrecedenceOverLegacyWindowSettings(t *testing.T) {
	setTestAppData(t)
	exeDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	userDataDir, err := DefaultUserDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(userDataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDataDir, legacyWindowSettingsFileName), []byte(`{"DisabledPlatforms":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDataDir, settingsFileName), []byte(`{
  "version": 1,
  "language": "en-US",
  "disabledPlatforms": ["Steam"]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	ResetPathSingletonsForTest(exeDir)
	settings, err := LoadAppSettings(exeDir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(settings.DisabledPlatforms, "Steam") {
		t.Fatalf("legacy settings overwrote current settings: %v", settings.DisabledPlatforms)
	}
}
