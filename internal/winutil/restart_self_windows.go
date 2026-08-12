//go:build windows

package winutil

import "time"

// childStartupGrace is how long a relaunched copy is given to fall over before
// this process treats the restart as successful and exits.
//
// It is not a deadline for starting: that is settled by the handle the shell
// returns. This only distinguishes a process that came up from one that died on
// the spot, so it need only outlast an immediate crash.
const childStartupGrace = 1500 * time.Millisecond

// RestartSelf re-launches the current executable with extraArgs, then exits this process.
func RestartSelf(extraArgs []string) error {
	return restartVia("open", extraArgs)
}
