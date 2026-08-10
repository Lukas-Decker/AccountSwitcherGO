//go:build !windows

package winutil

import "time"

// LocateExe has nothing to search outside Windows: there is no registry, and the
// platforms this looks for ship Windows executables.
func LocateExe(string, []string, time.Duration) (string, bool) { return "", false }
