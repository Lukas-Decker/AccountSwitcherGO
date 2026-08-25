package gamelib

import (
	"context"
	"strings"
	"sync"

	"account-switcher/internal/crashlog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// GameArtUpdatedEvent carries artwork that finished resolving after the games
// view was already drawn.
const GameArtUpdatedEvent = "gamelib-game-art-updated"

// GameArtPatch updates one tile.
type GameArtPatch struct {
	PlatformKey string `json:"platformKey"`
	GameID      string `json:"gameId"`
	ArtURL      string `json:"artUrl"`
}

// GameArtDonePatch says a platform's artwork pass has finished, so the view can
// stop showing that it is still filling in.
type GameArtDonePatch struct {
	PlatformKey string `json:"platformKey"`
	// Resolved counts the tiles that gained art in this pass.
	Resolved int `json:"resolved"`
}

// GameArtDoneEvent is emitted once per background pass.
const GameArtDoneEvent = "gamelib-game-art-done"

// artRefreshMu keeps one background pass per platform.
//
// The view asks for its games on every open and on every tab switch, and a
// second pass would duplicate every request the first one is already making.
var (
	artRefreshMu  sync.Mutex
	artRefreshing = map[string]bool{}
)

// RefreshArtInBackground resolves artwork for a platform and reports each tile
// that gains it.
//
// The view is drawn from whatever art is already published, which is instant
// and, after the first pass, complete. This fills in the rest: a game seen for
// the first time, or one whose only source is a CDN that has not been asked
// yet. Tiles appear as they resolve rather than the whole grid waiting for the
// slowest download.
func RefreshArtInBackground(platformKey string, allowNetwork bool) {
	platformKey = strings.TrimSpace(platformKey)
	if platformKey == "" {
		return
	}
	artRefreshMu.Lock()
	if artRefreshing[platformKey] {
		artRefreshMu.Unlock()
		return
	}
	artRefreshing[platformKey] = true
	artRefreshMu.Unlock()

	go func() {
		defer crashlog.Capture()
		defer func() {
			artRefreshMu.Lock()
			delete(artRefreshing, platformKey)
			artRefreshMu.Unlock()
		}()

		before := artByGame(platformKey)
		res, err := ResolvePlatform(context.Background(), platformKey, allowNetwork, artworkAllowed(), false)
		if err != nil {
			return
		}

		var changed int
		for _, g := range res.Games {
			art := strings.TrimSpace(g.ArtURL)
			if art == "" || art == before[g.GameID] {
				continue
			}
			changed++
			emitEvent(GameArtUpdatedEvent, GameArtPatch{
				PlatformKey: platformKey,
				GameID:      g.GameID,
				ArtURL:      art,
			})
		}
		emitEvent(GameArtDoneEvent, GameArtDonePatch{PlatformKey: platformKey, Resolved: changed})
	}()
}

// artByGame is what the view already has, so the pass only reports differences
// rather than re-sending every tile it was already showing.
func artByGame(platformKey string) map[string]string {
	out := map[string]string{}
	res, err := ResolvePlatform(context.Background(), platformKey, false, false, true)
	if err != nil {
		return out
	}
	for _, g := range res.Games {
		if art := strings.TrimSpace(g.ArtURL); art != "" {
			out[g.GameID] = art
		}
	}
	return out
}

func emitEvent(name string, payload any) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit(name, payload)
}
