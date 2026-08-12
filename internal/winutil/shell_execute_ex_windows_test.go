//go:build windows

package winutil

import (
	"os"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ShellExecuteExW validates cbSize and fails without saying why, so a struct that
// drifts from the Windows layout breaks the restart paths silently.
func TestShellExecuteExStartsAProcessAndReportsIt(t *testing.T) {
	if got := unsafe.Sizeof(shellExecuteInfoW{}); got != 112 {
		t.Fatalf("SHELLEXECUTEINFOW is %d bytes, Windows x64 expects 112", got)
	}

	comspec := os.Getenv("COMSPEC")
	if comspec == "" {
		t.Skip("no COMSPEC on this machine")
	}
	h, err := shellExecuteAndWaitForStart("open", comspec, "/c exit 7", os.TempDir())
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	defer windows.CloseHandle(h)

	// The handle has to be the child's, not a placeholder: the restart paths read
	// the process state through it to tell a started copy from one that died.
	if s, werr := windows.WaitForSingleObject(h, 5000); werr != nil || s != windows.WAIT_OBJECT_0 {
		t.Fatalf("wait on child: state=%d err=%v", s, werr)
	}
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		t.Fatalf("exit code: %v", err)
	}
	if code != 7 {
		t.Errorf("child exit code = %d, want 7", code)
	}
}
