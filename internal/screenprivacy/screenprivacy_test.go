package screenprivacy

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestApplyFollowsThePreferenceWithoutStrippingIt(t *testing.T) {
	t.Cleanup(func() { enabled.Store(false) })
	enabled.Store(false)

	var off application.WebviewWindowOptions
	Apply(&off)
	if off.ContentProtectionEnabled {
		t.Fatal("a window was protected with the preference off")
	}

	enabled.Store(true)
	var on application.WebviewWindowOptions
	Apply(&on)
	if !on.ContentProtectionEnabled {
		t.Fatal("a window was left capturable with the preference on")
	}

	// A window that sets the flag itself must keep it, so routing its options
	// through this package by mistake cannot silently strip the protection.
	enabled.Store(false)
	asked := application.WebviewWindowOptions{ContentProtectionEnabled: true}
	Apply(&asked)
	if !asked.ContentProtectionEnabled {
		t.Fatal("Apply stripped protection the window had asked for")
	}
}

func TestSetEnabledWithoutARunningAppIsHarmless(t *testing.T) {
	t.Cleanup(func() { enabled.Store(false) })

	// Called at startup before any window exists, and in headless runs where
	// application.Get() is nil.
	SetEnabled(true)
	if !Enabled() {
		t.Fatal("the preference was not recorded")
	}
	SetEnabled(false)
	if Enabled() {
		t.Fatal("the preference was not cleared")
	}
}
