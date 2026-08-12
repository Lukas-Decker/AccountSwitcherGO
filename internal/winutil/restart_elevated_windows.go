//go:build windows

package winutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

var singletonReleaser func()

// RegisterSingletonReleaser registers a callback run before spawning an elevated copy
// (e.g. release the app singleton mutex so the new instance can start).
func RegisterSingletonReleaser(f func()) {
	singletonReleaser = f
}

var modshell32 = windows.NewLazySystemDLL("shell32.dll")

// ErrElevationDeclined reports that the UAC prompt was dismissed. Not a failure:
// the user was asked and said no, and the caller should carry on unelevated
// rather than showing an error.
var ErrElevationDeclined = errors.New("elevation declined")

// RestartElevated re-launches the current executable with verb "runas" (UAC), forwards extraArgs,
// then exits this process. Call RegisterSingletonReleaser first so the mutex is released.
func RestartElevated(extraArgs []string) error {
	return restartVia("runas", extraArgs)
}

// restartVia relaunches this executable through the shell and exits once the
// replacement is running.
//
// It waits on the process the shell actually started. The previous approach
// polled for the singleton mutex for three seconds, but that clock began when
// ShellExecuteW returned, which for runas is before the UAC prompt has been
// answered: anyone who took more than three seconds to click Yes was told the
// restart had failed while the elevated copy was starting normally behind the
// message.
func restartVia(verb string, extraArgs []string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self = filepath.Clean(self)

	// The replacement cannot take the singleton while this process still holds it.
	if singletonReleaser != nil {
		singletonReleaser()
	}

	child, err := shellExecuteAndWaitForStart(verb, self, joinArgsUTF16(extraArgs), filepath.Dir(self))
	if err != nil {
		if errors.Is(err, windows.ERROR_CANCELLED) {
			return ErrElevationDeclined
		}
		return err
	}
	defer windows.CloseHandle(child)

	// A process that is still alive after this has got past its own startup; one
	// that is already gone never came up, and exiting on it would leave the user
	// with no window at all.
	switch s, werr := windows.WaitForSingleObject(child, uint32(childStartupGrace.Milliseconds())); {
	case werr != nil:
		return fmt.Errorf("wait for restarted process: %w", werr)
	case s == uint32(windows.WAIT_TIMEOUT):
		os.Exit(0)
	default:
		var code uint32
		if gerr := windows.GetExitCodeProcess(child, &code); gerr == nil {
			return fmt.Errorf("restarted process exited immediately with code %d", code)
		}
		return fmt.Errorf("restarted process exited immediately")
	}
	return nil
}

func joinArgsUTF16(args []string) string {
	if len(args) == 0 {
		return ""
	}
	var b strings.Builder
	for i, a := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(syscall.EscapeArg(strings.TrimSpace(a)))
	}
	return b.String()
}
