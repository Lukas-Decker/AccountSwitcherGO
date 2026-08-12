package streamer

import "testing"

// withStubbedWatcher swaps the Win32 detector out and resets the package state, so
// each test starts from a known point and never installs a real hook.
func withStubbedWatcher(t *testing.T) {
	t.Helper()
	reset := func() {
		state.mu.Lock()
		state.manual, state.autoEnabled, state.autoActive, state.detectedExe, state.hook = false, false, false, "", nil
		state.mu.Unlock()
	}
	prev := watchFn
	watchFn = func(bool) {}
	t.Cleanup(func() {
		watchFn = prev
		reset()
	})
	reset()
}

func TestEffectiveFollowsOverrideAndDetection(t *testing.T) {
	withStubbedWatcher(t)

	SetAutoEnabled(true)
	if Current().Effective {
		t.Fatal("auto enabled with nothing detected should not censor")
	}

	setDetected(true, "obs64.exe")
	if got := Current(); !got.Effective || got.DetectedExe != "obs64.exe" {
		t.Fatalf("detected broadcaster should censor: %+v", got)
	}

	// Re-enabling must not resurrect a detection from before the watcher stopped:
	// the process it saw may have exited while nothing was listening.
	SetAutoEnabled(false)
	SetAutoEnabled(true)
	if got := Current(); got.AutoActive || got.Effective || got.DetectedExe != "" {
		t.Fatalf("stale detection survived a disable/enable cycle: %+v", got)
	}

	// The override has to win on its own, with detection idle.
	SetManual(true)
	SetAutoEnabled(false)
	if !Current().Effective {
		t.Fatal("manual override should censor regardless of detection")
	}
}

func TestVirtualAdaptersAreNotSaltMaterial(t *testing.T) {
	cases := []struct {
		name    string
		mac     string
		virtual bool
	}{
		{"Ethernet", "d8:5e:d3:11:22:33", false},
		{"VMware Network Adapter VMnet1", "00:50:56:c0:00:01", true},
		{"Ethernet", "08:00:27:1a:2b:3c", true}, // VirtualBox OUI on an innocent name
		{"Wi-Fi", "9e:b6:d0:aa:bb:cc", true},    // randomised: locally administered
	}
	for _, tc := range cases {
		if got := isVirtualAdapter(tc.name, tc.mac); got != tc.virtual {
			t.Errorf("isVirtualAdapter(%q, %q) = %v, want %v", tc.name, tc.mac, got, tc.virtual)
		}
	}
}
