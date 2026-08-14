package basic

import (
	"strings"

	"account-switcher/internal/profileimage"
)

// Riot links live in the same per-platform record as tags, so an account that is
// deleted or superseded takes its link with it rather than leaving an entry
// behind for whichever account later reuses the id.

// ReadRiotAccountLink returns the Riot ID and region stored for one account.
func ReadRiotAccountLink(platformKey, uniqueID string) (RiotAccountLink, bool, error) {
	uniqueID = strings.TrimSpace(uniqueID)
	if uniqueID == "" {
		return RiotAccountLink{}, false, nil
	}
	f, err := readIdsFile(platformKey)
	if err != nil {
		return RiotAccountLink{}, false, err
	}
	link, ok := f.RiotAccounts[uniqueID]
	return link, ok, nil
}

// ReadAllRiotAccountLinks returns every stored link for a platform, so a list can
// be rendered without a file read per row.
func ReadAllRiotAccountLinks(platformKey string) (map[string]RiotAccountLink, error) {
	f, err := readIdsFile(platformKey)
	if err != nil {
		return nil, err
	}
	if f.RiotAccounts == nil {
		return map[string]RiotAccountLink{}, nil
	}
	out := make(map[string]RiotAccountLink, len(f.RiotAccounts))
	for k, v := range f.RiotAccounts {
		out[k] = v
	}
	return out, nil
}

// MergeRiotAccountSnapshot records captured profile data against an account.
//
// The Riot ID is only adopted when the account has none or was filled in
// automatically before: a name the user typed is theirs, and a client that
// happens to be signed in as somebody else must not quietly rewrite it.
func MergeRiotAccountSnapshot(platformKey, uniqueID string, captured RiotAccountLink) error {
	uniqueID = strings.TrimSpace(uniqueID)
	if uniqueID == "" {
		return nil
	}
	f, err := readIdsFile(platformKey)
	if err != nil {
		return err
	}
	if f.RiotAccounts == nil {
		f.RiotAccounts = map[string]RiotAccountLink{}
	}
	existing := f.RiotAccounts[uniqueID]

	next := existing
	if !existing.Manual && strings.TrimSpace(captured.RiotID) != "" {
		next.RiotID = strings.TrimSpace(captured.RiotID)
		if strings.TrimSpace(captured.Region) != "" {
			next.Region = strings.TrimSpace(captured.Region)
		}
	}
	next.Level = captured.Level
	next.IconID = captured.IconID
	next.Ranks = captured.Ranks
	next.CapturedAt = captured.CapturedAt

	f.RiotAccounts[uniqueID] = next
	return writeIdsFile(platformKey, f)
}

// WriteRiotAccountLink stores the link for one account. An empty Riot ID clears
// it, so the UI needs no separate removal call.
func WriteRiotAccountLink(platformKey, uniqueID string, link RiotAccountLink) error {
	uniqueID = strings.TrimSpace(uniqueID)
	if uniqueID == "" {
		return nil
	}
	f, err := readIdsFile(platformKey)
	if err != nil {
		return err
	}
	link.RiotID = strings.TrimSpace(link.RiotID)
	link.Region = strings.TrimSpace(link.Region)

	if link.RiotID == "" {
		delete(f.RiotAccounts, uniqueID)
	} else {
		if f.RiotAccounts == nil {
			f.RiotAccounts = map[string]RiotAccountLink{}
		}
		// Typed by hand, so auto-capture leaves it alone from here on. Any snapshot
		// already held is kept: it still describes this account.
		link.Manual = true
		if prev, ok := f.RiotAccounts[uniqueID]; ok {
			link.Level, link.IconID, link.Ranks, link.CapturedAt = prev.Level, prev.IconID, prev.Ranks, prev.CapturedAt
		}
		f.RiotAccounts[uniqueID] = link
	}
	return writeIdsFile(platformKey, f)
}

// RefreshPlatformProfileImage re-fetches an account's avatar from whatever the
// platform hook names as its source.
//
// Exported because the automated image path only runs as part of a save. An
// account whose picture changed since it was last saved has no other way back to
// the current one, short of switching to it and saving again.
//
// Honours the manual marker, so a picture the user chose stays chosen.
func RefreshPlatformProfileImage(platformKey, uniqueID string) bool {
	platformKey = strings.TrimSpace(platformKey)
	uniqueID = strings.TrimSpace(uniqueID)
	if platformKey == "" || uniqueID == "" {
		return false
	}
	if profileimage.HasManualProfileMarker(platformKey, uniqueID) {
		return false
	}
	url := platformProfileImageURL(platformKey, uniqueID)
	if url == "" {
		return false
	}
	queueProfileImageDownload(platformKey, uniqueID, url, 0)
	return true
}
