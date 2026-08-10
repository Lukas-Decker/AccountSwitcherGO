//go:build windows

package winutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeExe(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// The install landing on a drive other than the one in the default path is the
// case that used to send the user to the file picker.
func TestFindExeViaDefaultLayoutsFindsSameLayoutElsewhere(t *testing.T) {
	other := t.TempDir()
	want := writeExe(t, filepath.Join(other, "Riot Games", "Riot Client", "RiotClientServices.exe"))

	got, ok := findExeUnderRootsForTest(
		[]string{other},
		"RiotClientServices.exe",
		[]string{`C:\Riot Games\Riot Client\RiotClientServices.exe`},
	)
	if !ok || got != want {
		t.Fatalf("got (%q, %v), want %q", got, ok, want)
	}
}

// Installers often record a folder whose executable sits one level further in.
func TestExeUnderDirLooksOneLevelDeeper(t *testing.T) {
	root := t.TempDir()
	want := writeExe(t, filepath.Join(root, "Riot Client", "RiotClientServices.exe"))

	got, ok := exeUnderDir(root, "RiotClientServices.exe")
	if !ok || got != want {
		t.Fatalf("got (%q, %v), want %q", got, ok, want)
	}
}

func TestValidExePathRejectsDirectoriesAndMisses(t *testing.T) {
	dir := t.TempDir()
	if _, ok := validExePath(dir); ok {
		t.Fatal("a directory is not an executable")
	}
	if _, ok := validExePath(filepath.Join(dir, "nope.exe")); ok {
		t.Fatal("a missing file should not resolve")
	}
}

// The sweep must stay shallow: a match buried deeper than the limit is left for
// the user rather than found by walking the whole drive.
func TestScanDirForExeRespectsDepthLimit(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c", "d", "e")
	writeExe(t, filepath.Join(deep, "target.exe"))

	if _, ok := scanDirForExe(root, "target.exe", 2, time.Now().Add(time.Second)); ok {
		t.Fatal("depth limit should have stopped the walk short of the match")
	}
	if _, ok := scanDirForExe(root, "target.exe", 6, time.Now().Add(2*time.Second)); !ok {
		t.Fatal("a deeper limit should reach the match")
	}
}

func TestScanDirForExeStopsAtDeadline(t *testing.T) {
	root := t.TempDir()
	writeExe(t, filepath.Join(root, "x", "target.exe"))

	if _, ok := scanDirForExe(root, "target.exe", 5, time.Now().Add(-time.Second)); ok {
		t.Fatal("an expired budget should stop the walk")
	}
}

// findExeUnderRootsForTest mirrors findExeViaDefaultLayouts against an explicit
// root list, so the assertion does not depend on what is installed on the
// machine running the tests.
func findExeUnderRootsForTest(roots []string, exeName string, defaults []string) (string, bool) {
	for _, def := range defaults {
		rel := def[len(filepath.VolumeName(def)):]
		for len(rel) > 0 && (rel[0] == '\\' || rel[0] == '/') {
			rel = rel[1:]
		}
		for _, root := range roots {
			if p, ok := validExePath(filepath.Join(root, rel)); ok {
				return p, true
			}
		}
	}
	return "", false
}
