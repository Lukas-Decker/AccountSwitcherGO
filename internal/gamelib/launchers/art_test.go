package launchers

import (
	"os"
	"path/filepath"
	"testing"
)

// A folder with several executables is as likely to yield an installer or a
// crash reporter icon as the game's own, and a wrong icon is worse than none.
func TestExeForIcon_OnlyGuessesWhenThereIsOneExecutable(t *testing.T) {
	t.Parallel()

	single := t.TempDir()
	game := filepath.Join(single, "Game.exe")
	writeFile(t, game, "MZ")
	if got := exeForIcon(single, ""); got != game {
		t.Errorf("exeForIcon = %q, want %q", got, game)
	}

	many := t.TempDir()
	writeFile(t, filepath.Join(many, "Game.exe"), "MZ")
	writeFile(t, filepath.Join(many, "CrashReporter.exe"), "MZ")
	if got := exeForIcon(many, ""); got != "" {
		t.Errorf("exeForIcon guessed %q from a folder with two executables", got)
	}

	if got := exeForIcon("", ""); got != "" {
		t.Errorf("exeForIcon(\"\") = %q, want empty", got)
	}
}

// Epic records the launch executable relative to the install folder, which is
// the one case where the right binary is known rather than guessed.
func TestExeForIcon_ExplicitPathWinsAndResolvesRelative(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Launcher.exe"), "MZ")
	want := filepath.Join(dir, "Binaries", "Win64", "Game.exe")
	writeFile(t, want, "MZ")

	got := exeForIcon(dir, filepath.Join("Binaries", "Win64", "Game.exe"))
	if got != want {
		t.Errorf("exeForIcon = %q, want %q", got, want)
	}

	// An explicit path that no longer exists falls back to the folder scan
	// rather than handing the extractor a missing file.
	if got := exeForIcon(dir, "Gone.exe"); got != filepath.Join(dir, "Launcher.exe") {
		t.Errorf("stale explicit path gave %q", got)
	}
}

func TestInstallDirIcons(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "game.ico"), "icon")
	writeFile(t, filepath.Join(dir, "cover.png"), "png")
	writeFile(t, filepath.Join(dir, "readme.txt"), "text")
	writeFile(t, filepath.Join(dir, "game.exe"), "MZ")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := installDirIcons(dir)
	if len(got) != 2 {
		t.Fatalf("got %v, want just the two image files", got)
	}
	for _, p := range got {
		switch filepath.Ext(p) {
		case ".ico", ".png":
		default:
			t.Errorf("non-image candidate %q", p)
		}
	}

	if got := installDirIcons(""); got != nil {
		t.Errorf("empty path gave %v", got)
	}
	if got := installDirIcons(filepath.Join(dir, "nope")); got != nil {
		t.Errorf("missing folder gave %v", got)
	}
}

// GOG stores some image references protocol-relative and some without an
// extension, so the candidate list has to cover both without duplicating.
func TestAppendUniqueURL(t *testing.T) {
	t.Parallel()

	list := appendUniqueURL(nil, "https://images.gog.com/abc.jpg")
	if len(list) != 1 || list[0] != "https://images.gog.com/abc.jpg" {
		t.Fatalf("got %v", list)
	}

	// Adding the same URL twice must not grow the list.
	list = appendUniqueURL(list, "https://images.gog.com/abc.jpg")
	if len(list) != 1 {
		t.Errorf("duplicate URL added: %v", list)
	}

	// A bare image id is a template the client completes, so both common
	// completions are offered and the bare form kept as a last try.
	list = appendUniqueURL(nil, "https://images.gog.com/abc")
	if len(list) != 3 {
		t.Fatalf("got %v, want the two completions plus the bare form", list)
	}
	if filepath.Ext(list[0]) == "" {
		t.Errorf("first candidate %q has no extension", list[0])
	}
}
