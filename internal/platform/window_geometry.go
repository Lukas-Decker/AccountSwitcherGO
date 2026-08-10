package platform



// WindowGeometry is the main window's remembered size, position and maximised
// state.
type WindowGeometry struct {
	Width     int
	Height    int
	X         int
	Y         int
	Maximised bool
}

// minRememberedWindowSize guards against restoring a window too small to use.
// It matches the window's own MinWidth/MinHeight; a stored value below this came
// from a bad read rather than from the user.
const (
	minRememberedWindowWidth  = 760
	minRememberedWindowHeight = 520
)


// Valid reports whether a remembered size is usable. Position is allowed to be
// anything, including negative, because a second monitor left of the primary one
// has negative coordinates.
func (g WindowGeometry) Valid() bool {
	return g.Width >= minRememberedWindowWidth && g.Height >= minRememberedWindowHeight
}

// WindowGeometryFromSettings reads the remembered geometry out of settings.
func WindowGeometryFromSettings(s AppSettings) WindowGeometry {
	return WindowGeometry{
		Width:     s.WindowWidth,
		Height:    s.WindowHeight,
		X:         s.WindowX,
		Y:         s.WindowY,
		Maximised: s.WindowMaximised,
	}
}

// SaveWindowGeometry persists the main window's geometry.
//
// Callers hand this the window's last known bounds when it settles, not on every
// frame of a drag: each call rewrites the settings file.
func SaveWindowGeometry(g WindowGeometry) error {
	// Goes through the shared settings mutation so a resize cannot write its
	// copy of the settings back over a change the user made in the meantime.
	// This runs on a timer whenever the window moves, so it would win that race
	// often.
	return mutateSettings(func(s *AppSettings) error {
		// A maximised window reports the maximised size; keeping it would lose
		// the size to restore to, so only the flag is updated in that case.
		if !g.Maximised {
			if !g.Valid() {
				return nil
			}
			s.WindowWidth = g.Width
			s.WindowHeight = g.Height
			s.WindowX = g.X
			s.WindowY = g.Y
		}
		s.WindowMaximised = g.Maximised
		return nil
	})
}
