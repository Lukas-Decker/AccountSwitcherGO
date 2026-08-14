//go:build windows

package winutil

// RestartSelf re-launches the current executable with extraArgs, then exits this process.
func RestartSelf(extraArgs []string) error {
	return restartVia("open", extraArgs)
}
