package gameart

import (
	"context"
	"net/http"
	"strings"
)

// ArchiveRef identifies a game to the artwork archives.
//
// SteamAppID is optional and only Steam can supply one, but where it exists it
// is worth carrying: both archives can address a Steam app exactly, which beats
// matching on a title that several games share.
type ArchiveRef struct {
	SteamAppID string
	Name       string
}

// ArchiveCandidates asks every configured archive and returns what they offer,
// best shape first.
//
// Both are asked rather than stopping at the first that answers, because they
// fail in different places: SteamGridDB is people uploading grids for games
// they play, so it is rich for popular titles and empty for obscure ones, and
// IGDB is a catalogue with a cover for almost everything but no community
// artwork at all. Taking the best shape across both covers more of a library
// than either does alone, and the cost is bounded because this is only reached
// for games nothing cheaper resolved.
func ArchiveCandidates(ctx context.Context, client *http.Client, ref ArchiveRef) []Candidate {
	ref.SteamAppID = strings.TrimSpace(ref.SteamAppID)
	ref.Name = strings.TrimSpace(ref.Name)
	if ref.SteamAppID == "" && ref.Name == "" {
		return nil
	}

	var out []Candidate
	if ref.SteamAppID != "" {
		out = append(out, SteamGridDBCandidates(ctx, client, ref.SteamAppID)...)
	} else {
		out = append(out, SteamGridDBCandidatesByName(ctx, client, ref.Name)...)
	}
	out = append(out, IGDBCandidates(ctx, client, ref.SteamAppID, ref.Name)...)

	// Order across both, then keep one per shape: the chain stops at the first
	// candidate that publishes, so a second cover only matters if the first
	// fails to download, and carrying every one would multiply the requests a
	// stubborn game costs.
	probe := Request{Candidates: out}
	return firstOfEachTier(probe.ordered(true))
}

// AnyArchiveEnabled reports whether at least one archive has credentials.
func AnyArchiveEnabled() bool {
	return SteamGridDBEnabled() || IGDBEnabled()
}
