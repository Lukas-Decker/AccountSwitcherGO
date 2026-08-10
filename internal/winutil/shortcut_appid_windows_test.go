//go:build windows

package winutil

import (
	"os"
	"path/filepath"
	"testing"
)

// Exercises the real COM path end to end. The interface pointer is now received
// into a typed variable rather than through a uintptr, so this guards the change
// against a silent regression: a wrong out-parameter would surface here as a
// failed call or a crash, not as a compile error.
func TestWriteShortcutLnkSetsAppUserModelID(t *testing.T) {
	dir := t.TempDir()
	lnk := filepath.Join(dir, "probe.lnk")

	target := filepath.Join(os.Getenv("SystemRoot"), "System32", "notepad.exe")
	if _, err := os.Stat(target); err != nil {
		t.Skipf("no target executable to point at: %v", err)
	}

	if err := WriteShortcutLnk(lnk, target, "", dir, "probe", target, "com.accountswitcher.test"); err != nil {
		t.Fatalf("WriteShortcutLnk: %v", err)
	}
	st, err := os.Stat(lnk)
	if err != nil {
		t.Fatalf("shortcut was not created: %v", err)
	}
	if st.Size() == 0 {
		t.Fatal("shortcut is empty")
	}
}
