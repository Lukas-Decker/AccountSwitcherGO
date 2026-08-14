package platform

import (
	"testing"
	"time"
)

// Every configuration written before this option existed expects to be forced,
// so absent has to mean yes. Reading it as "off" would silently change what a
// switch does on every machine that already had settings.
func TestForceCloseDefaultsToOnWhenUnset(t *testing.T) {
	if !(PlatformSettings{}).ForceCloseEnabled() {
		t.Error("an unset ForceCloseAfterTimeout must behave as before, which is forced")
	}
	off := false
	if (PlatformSettings{ForceCloseAfterTimeout: &off}).ForceCloseEnabled() {
		t.Error("an explicit false must disable forcing")
	}
	on := true
	if !(PlatformSettings{ForceCloseAfterTimeout: &on}).ForceCloseEnabled() {
		t.Error("an explicit true must enable forcing")
	}
}

func TestCloseGraceIsBounded(t *testing.T) {
	if got := (PlatformSettings{}).CloseGrace(); got != 0 {
		t.Errorf("unset grace = %v, want 0 so the method default applies", got)
	}
	if got := (PlatformSettings{CloseTimeoutSeconds: 12}).CloseGrace(); got != 12*time.Second {
		t.Errorf("grace = %v", got)
	}
	// A mistyped value must not hang a switch for an hour.
	if got := (PlatformSettings{CloseTimeoutSeconds: 99999}).CloseGrace(); got != 120*time.Second {
		t.Errorf("grace was not clamped: %v", got)
	}
	if got := (PlatformSettings{CloseTimeoutSeconds: -5}).CloseGrace(); got != 0 {
		t.Errorf("negative grace = %v, want 0", got)
	}
}
