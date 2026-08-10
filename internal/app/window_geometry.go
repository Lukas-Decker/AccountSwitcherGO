package app

import (
	"log"
	"sync"
	"time"

	"account-switcher/internal/platform"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// windowGeometrySettleDelay is how long the window must sit still before its
// geometry is written. Resize and move events arrive continuously while a drag
// is in progress, and each write is a settings file rewrite.
const windowGeometrySettleDelay = 600 * time.Millisecond

type windowGeometryRecorder struct {
	win application.Window

	mu    sync.Mutex
	timer *time.Timer
}

// rememberWindowGeometry persists the window's size and position whenever the
// user finishes resizing or moving it, and restores the maximised state now.
func rememberWindowGeometry(win application.Window, guiSettings platform.AppSettings) {
	if win == nil {
		return
	}
	if guiSettings.WindowMaximised {
		win.Maximise()
	}

	r := &windowGeometryRecorder{win: win}
	win.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) { r.schedule() })
	win.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) { r.schedule() })

	// Closing is the one moment the current geometry is certainly final, and a
	// pending settle timer would never fire after shutdown.
	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) { r.saveNow() })
}

func (r *windowGeometryRecorder) schedule() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.timer != nil {
		r.timer.Stop()
	}
	r.timer = time.AfterFunc(windowGeometrySettleDelay, r.saveNow)
}

func (r *windowGeometryRecorder) saveNow() {
	r.mu.Lock()
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	win := r.win
	r.mu.Unlock()
	if win == nil {
		return
	}

	// Reading the window has to happen on the UI thread; the settle timer fires
	// on its own goroutine.
	application.InvokeSync(func() {
		width, height := win.Size()
		x, y := win.Position()
		geometry := platform.WindowGeometry{
			Width:     width,
			Height:    height,
			X:         x,
			Y:         y,
			Maximised: win.IsMaximised(),
		}
		if err := platform.SaveWindowGeometry(geometry); err != nil {
			log.Printf("save window geometry: %v", err)
		}
	})
}
