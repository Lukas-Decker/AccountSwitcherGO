//go:build envtests

// Off by default. These tests redirect %APPDATA% and %USERPROFILE% or read
// the running League Client, so they reach outside the package and are only
// as isolated as their own setup. Run them with: go test -tags envtests ./...

package platform

import (
	"sync"
	"testing"
)

// Batch updates and window-geometry saves are two independent writers of the same
// file. A geometry save that loads before a batch update and writes after it puts
// its stale copy back, and the user's change vanishes with no error anywhere.
func TestConcurrentWritersDoNotLoseEachOthersChanges(t *testing.T) {
	// APPDATA is redirected first. Without it the settings path can resolve to the
	// real user data directory however temporary the exe dir is, and the test then
	// writes over the user's own settings.
	setTestAppData(t)
	dir := t.TempDir()
	ResetPathSingletonsForTest(dir)

	svc := &PlatformService{}
	on := true
	lost := 0

	for round := 0; round < 10; round++ {
		// Start from a clean slate each round: the flag must survive exactly one
		// write while geometry saves are hammering the same file.
		off := false
		if err := svc.UpdateSettings(SettingsBatchUpdate{ExitToTray: &off}); err != nil {
			t.Fatalf("reset: %v", err)
		}

		var wg sync.WaitGroup
		for i := 0; i < 40; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				_ = SaveWindowGeometry(WindowGeometry{X: 10, Y: 20, Width: 900 + n, Height: 700})
			}(i)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.UpdateSettings(SettingsBatchUpdate{ExitToTray: &on})
		}()
		wg.Wait()

		s, err := loadSettings(dir)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if !s.ExitToTray {
			lost++
		}
	}

	t.Logf("rounds where the batch update was lost: %d/10", lost)
	if lost > 0 {
		t.Errorf("a geometry save overwrote the batch update in %d of 10 rounds", lost)
	}
}
