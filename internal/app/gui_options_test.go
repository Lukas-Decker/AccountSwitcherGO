package app

import (
	"testing"

	"account-switcher/internal/cli"
	"account-switcher/internal/platform"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestMainWindowOptionsExposeBrowserTools(t *testing.T) {
	opts := mainWindowOptions(platform.AppSettings{}, cli.Parsed{})

	if !opts.DevToolsEnabled {
		t.Fatal("DevToolsEnabled = false, want true")
	}
	if opts.DefaultContextMenuDisabled {
		t.Fatal("DefaultContextMenuDisabled = true, want false")
	}
	for _, accelerator := range []string{"Ctrl+Shift+I", "F11"} {
		if opts.KeyBindings[accelerator] == nil {
			t.Fatalf("KeyBindings missing %q", accelerator)
		}
	}
}

func TestMainWindowOptionsPreserveStartupPlacement(t *testing.T) {
	centered := mainWindowOptions(platform.AppSettings{StartProgramCentered: true}, cli.Parsed{})
	if centered.InitialPosition != application.WindowCentered {
		t.Fatalf("centered InitialPosition = %v", centered.InitialPosition)
	}

	hidden := mainWindowOptions(platform.AppSettings{}, cli.Parsed{StartInTray: true})
	if !hidden.Hidden {
		t.Fatal("Hidden = false, want true for tray startup")
	}
}

func TestGitHubUpdaterConfigTracksStableOnly(t *testing.T) {
	// The pre-release preference was a switch with nothing behind it. The
	// updater now says so in one place rather than reading a setting that
	// never changed what it did.
	cfg := githubUpdaterConfig(platform.AppSettings{})
	if cfg.Prerelease {
		t.Fatal("Prerelease = true, want stable releases only")
	}
}
