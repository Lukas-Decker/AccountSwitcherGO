package riotservice

import (
	"errors"
	"sort"
	"strings"
	"time"

	"account-switcher/internal/basic"
	"account-switcher/internal/riot"
	"account-switcher/internal/security"
)

// betweenAccounts is the pause between accounts during a bulk snapshot.
//
// A development key allows 20 requests a second and 100 every two minutes, and
// each account costs three. Ten accounts is thirty calls, which fits, but only
// if they are not fired all at once: the per-second limit is the easy one to
// trip and the two-minute one is the one that hurts.
const betweenAccounts = 400 * time.Millisecond

// SnapshotResultDTO reports what a bulk snapshot managed.
type SnapshotResultDTO struct {
	Total     int `json:"total"`
	Refreshed int `json:"refreshed"`
	Skipped   int `json:"skipped"`
	Failed    int `json:"failed"`
	// StoppedEarly is set when Riot asked for a pause, so the user knows the run
	// is incomplete rather than finished.
	StoppedEarly bool     `json:"stoppedEarly"`
	Errors       []string `json:"errors"`
}

// SnapshotAll refreshes every linked account and stores what comes back.
//
// This is the deliberate counterpart to never polling: a development key is not
// used on its own initiative, but the user asking for one round of updates is a
// different thing, so the refresh is forced past both the tier gate and the
// per-account interval.
//
// Sequential on purpose. Riot's limits are per key, not per account, so running
// these in parallel only means hitting the ceiling sooner.
func (s *Service) SnapshotAll() (SnapshotResultDTO, error) {
	if err := security.RequireUnlocked(); err != nil {
		return SnapshotResultDTO{}, err
	}
	links, err := basic.ReadAllRiotAccountLinks(PlatformKey)
	if err != nil {
		return SnapshotResultDTO{}, err
	}

	// Sorted so a run is reproducible and the log reads in a stable order.
	ids := make([]string, 0, len(links))
	for id := range links {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := SnapshotResultDTO{Total: len(ids)}
	for i, uniqueID := range ids {
		link := links[uniqueID]
		if strings.TrimSpace(link.RiotID) == "" {
			out.Skipped++
			continue
		}
		if i > 0 {
			time.Sleep(betweenAccounts)
		}

		card, cerr := s.getCard(uniqueID, true)
		switch {
		case cerr != nil:
			out.Failed++
			out.Errors = append(out.Errors, link.RiotID+": "+cerr.Error())
		case card.Error != "":
			out.Failed++
			out.Errors = append(out.Errors, link.RiotID+": "+card.Error)
			// Riot asking for a pause applies to the key, so the accounts after this
			// one would fail the same way. Stopping leaves the quota for the retry.
			if strings.Contains(strings.ToLower(card.Error), "rate limited") {
				out.StoppedEarly = true
				logRiot().Warn("bulk snapshot stopped early: rate limited",
					"done", out.Refreshed, "remaining", len(ids)-i-1)
				return out, nil
			}
		default:
			out.Refreshed++
			// The picture is part of the profile, and this is the only path that
			// refreshes it outside a save.
			basic.RefreshPlatformProfileImage(PlatformKey, uniqueID)
		}
	}

	logRiot().Info("bulk snapshot finished",
		"total", out.Total, "refreshed", out.Refreshed, "skipped", out.Skipped, "failed", out.Failed)
	return out, nil
}

// SnapshotAvailable reports whether a bulk snapshot could do anything: it needs
// a key, since the running client only ever answers for one account.
func (s *Service) SnapshotAvailable() (bool, error) {
	key, err := apiKey()
	if err != nil {
		if errors.Is(err, riot.ErrNoAPIKey) {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(key) != "", nil
}
