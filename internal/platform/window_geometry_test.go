package platform

import "testing"

func TestWindowGeometryValidRejectsUnusablySmall(t *testing.T) {
	// Nothing recorded yet: the window should fall back to its default size
	// rather than open at 0x0.
	if (WindowGeometry{}).Valid() {
		t.Fatal("an empty geometry should not be treated as remembered")
	}
	if (WindowGeometry{Width: 400, Height: 300}).Valid() {
		t.Fatal("a size below the window minimum should be rejected")
	}
	if !(WindowGeometry{Width: 1100, Height: 700}).Valid() {
		t.Fatal("a normal size should be accepted")
	}
}

func TestWindowGeometryAllowsNegativePosition(t *testing.T) {
	// A monitor left of or above the primary one has negative coordinates, and
	// the window belongs there if that is where the user left it.
	g := WindowGeometry{Width: 900, Height: 600, X: -1400, Y: -200}
	if !g.Valid() {
		t.Fatalf("geometry on a secondary monitor should be valid: %#v", g)
	}
}

func TestWindowGeometryFromSettingsRoundTrips(t *testing.T) {
	s := AppSettings{WindowWidth: 1280, WindowHeight: 800, WindowX: 40, WindowY: 60, WindowMaximised: true}
	got := WindowGeometryFromSettings(s)
	want := WindowGeometry{Width: 1280, Height: 800, X: 40, Y: 60, Maximised: true}
	if got != want {
		t.Fatalf("geometry = %#v, want %#v", got, want)
	}
}
