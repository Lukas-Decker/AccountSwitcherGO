//go:build !windows

package winutil

import (
	"errors"
	"fmt"
)

// ErrElevationDeclined exists so callers can test for it without a build tag.
// Never returned here: there is no UAC prompt to decline.
var ErrElevationDeclined = errors.New("elevation declined")

var singletonReleaser func()

// RegisterSingletonReleaser is a no-op on non-Windows.
func RegisterSingletonReleaser(f func()) {
	singletonReleaser = f
}

// RestartElevated is unsupported on non-Windows.
func RestartElevated(extraArgs []string) error {
	return fmt.Errorf("restart elevated: %w", ErrUnsupported)
}
