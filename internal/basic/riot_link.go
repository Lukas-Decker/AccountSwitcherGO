package basic

import "strings"

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
		f.RiotAccounts[uniqueID] = link
	}
	return writeIdsFile(platformKey, f)
}
