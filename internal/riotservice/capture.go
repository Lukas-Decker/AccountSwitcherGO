package riotservice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"account-switcher/internal/basic"
	"account-switcher/internal/riot"
)

// captureTimeout bounds a capture. The client is on loopback, so this only has
// to cover a busy client, not a network.
const captureTimeout = 8 * time.Second

// riotClientInstallsPath is where Riot records every product's install
// directory. Read rather than guessed: the install can be on any drive, and on
// a machine with League and the Riot Client on different drives no amount of
// probing well-known paths finds it.
func riotClientInstallsPath() string {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "Riot Games", "RiotClientInstalls.json")
}

// CaptureFromClient records the signed-in account's profile against uniqueID.
//
// Called when an account is saved or switched to, which is exactly when the
// running client is signed in as that account. It is the only keyless source of
// this data, and it answers for one account at a time, so the moment of the
// switch is the one chance to record it.
//
// Reports whether anything was captured. A closed client is not an error: it is
// the ordinary case, and the caller carries on with whatever was recorded before.
func CaptureFromClient(uniqueID string) (bool, error) {
	uniqueID = strings.TrimSpace(uniqueID)
	if uniqueID == "" {
		return false, nil
	}

	client, err := riot.ConnectLCU(riotClientInstallsPath())
	if err != nil {
		if errors.Is(err, riot.ErrLCUNotRunning) {
			return false, nil
		}
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), captureTimeout)
	defer cancel()

	summoner, err := client.CurrentSummoner(ctx)
	if err != nil {
		// A lockfile left behind by a client that has since exited looks like this.
		if errors.Is(err, riot.ErrLCUNotRunning) {
			return false, nil
		}
		return false, err
	}

	captured := basic.RiotAccountLink{
		Level:      summoner.SummonerLevel,
		IconID:     summoner.ProfileIconID,
		CapturedAt: time.Now().UTC(),
	}
	if id, ok := summoner.ID(); ok {
		captured.RiotID = id.String()
	}

	// Ranked standings are a second call and a lesser prize: an account with no
	// ranked history has none, so a failure here must not discard the identity and
	// icon already in hand.
	if entries, rerr := client.CurrentRankedStats(ctx); rerr == nil {
		for _, e := range entries {
			captured.Ranks = append(captured.Ranks, basic.RiotRankSnapshot{
				Queue:        e.QueueType,
				Tier:         e.Tier,
				Rank:         e.Rank,
				LeaguePoints: e.LeaguePoints,
				Wins:         e.Wins,
				Losses:       e.Losses,
			})
		}
	} else {
		logRiot().Debug("ranked stats unavailable from the client", "err", rerr)
	}

	if err := basic.MergeRiotAccountSnapshot(PlatformKey, uniqueID, captured); err != nil {
		return false, err
	}
	logRiot().Info("captured Riot profile from the running client",
		"uniqueID", uniqueID, "riotId", captured.RiotID, "level", captured.Level, "ranks", len(captured.Ranks))
	return true, nil
}

// ProfileIconURLFor returns the icon URL for a saved account, or "" when none is
// known.
//
// Community Dragon's "latest" path, so no patch version has to be resolved first
// and an icon added in the newest patch resolves immediately.
func ProfileIconURLFor(uniqueID string) string {
	link, ok, err := basic.ReadRiotAccountLink(PlatformKey, uniqueID)
	if err != nil || !ok || link.IconID <= 0 {
		return ""
	}
	return riot.ProfileIconURLLatest(link.IconID)
}

// fillFromRunningClient replaces the card's figures with live ones when the
// client is signed in as this account.
//
// The identity check is the point: the client answers for whoever is signed in,
// which is usually not the account being asked about, and showing one account's
// rank under another's name would be worse than showing nothing.
func (s *Service) fillFromRunningClient(card *CardDTO, want riot.ID) bool {
	client, err := riot.ConnectLCU(riotClientInstallsPath())
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), captureTimeout)
	defer cancel()

	summoner, err := client.CurrentSummoner(ctx)
	if err != nil {
		return false
	}
	got, ok := summoner.ID()
	if !ok || !strings.EqualFold(got.String(), want.String()) {
		return false
	}

	card.Ranks = nil
	card.Source = "client"
	card.CapturedAt = ""
	card.Level = summoner.SummonerLevel
	card.IconID = summoner.ProfileIconID
	card.IconURL = s.cachedImage(ctx, riot.ProfileIconURLLatest(summoner.ProfileIconID))

	if entries, rerr := client.CurrentRankedStats(ctx); rerr == nil {
		s.appendRanks(ctx, card, entries, riot.QueueSoloDuo, riot.QueueFlex, riot.QueueTFT, riot.QueueTFTDoubleUp)
	} else {
		// Logged rather than swallowed: an account with a level but no ranks looks
		// the same whether it is genuinely unranked or the call quietly failed.
		logRiot().Info("ranked stats unavailable from the running client", "err", rerr)
	}
	return true
}

// storeSnapshot records a freshly fetched card so it outlives the session.
//
// Everything fetched is kept, whatever the source. A key that may only be used
// sparingly is exactly the one whose answers are worth holding on to.
func (s *Service) storeSnapshot(uniqueID string, card CardDTO) {
	captured := basic.RiotAccountLink{
		RiotID:     card.RiotID,
		Region:     card.Region,
		Level:      card.Level,
		IconID:     card.IconID,
		CapturedAt: time.Now().UTC(),
	}
	for _, r := range card.Ranks {
		captured.Ranks = append(captured.Ranks, basic.RiotRankSnapshot{
			Queue:        r.Queue,
			Tier:         r.Tier,
			Rank:         r.Rank,
			LeaguePoints: r.LeaguePoints,
			Wins:         r.Wins,
			Losses:       r.Losses,
		})
	}
	if err := basic.MergeRiotAccountSnapshot(PlatformKey, uniqueID, captured); err != nil {
		logRiot().Debug("could not store snapshot", "uniqueID", uniqueID, "err", err)
	}
}
