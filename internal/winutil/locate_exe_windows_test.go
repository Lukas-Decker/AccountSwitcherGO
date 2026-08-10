//go:build windows

package winutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// An install on a drive other than the one in the default path is the case that
// used to send the user to the file picker.
func TestLocateExeFindsSameLayoutOnAnotherDrive(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "Riot Games", "Riot Client", "RiotClientServices.exe")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The sweep has to find it, and has to stop at its depth limit rather than
	// walking the whole drive.
	got, ok := scanDirForExe(root, "RiotClientServices.exe", 3, time.Now().Add(2*time.Second))
	if !ok || got != want {
		t.Fatalf("got (%q, %v), want %q", got, ok, want)
	}
	if _, ok := scanDirForExe(root, "RiotClientServices.exe", 0, time.Now().Add(time.Second)); ok {
		t.Fatal("depth limit should have stopped the walk short of the match")
	}
}
