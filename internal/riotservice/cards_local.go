package riotservice

import (
	"context"
	"strings"

	"account-switcher/internal/basic"
	"account-switcher/internal/riot"
	"account-switcher/internal/security"
)

// AccountCards returns a card for every linked account, built entirely from what
// is already stored.
//
// Whether an account is linked, and everything last captured for it, sits on
// disk in the same record the account list itself comes from. Answering that
// through GetCard meant the question could not be asked without a call that may
// reach a game client or the web API, which is why a context menu, built the
// instant it opens, showed a linked account as unlinked: the answer had not
// arrived yet, and an open menu holds plain values rather than a live view, so
// it could not correct itself afterwards either.
//
// No network, no client, no key. Fresh figures are what Refresh is for.
func (s *Service) AccountCards() (map[string]CardDTO, error) {
	if err := security.RequireUnlocked(); err != nil {
		return nil, err
	}
	links, err := basic.ReadAllRiotAccountLinks(PlatformKey)
	if err != nil {
		return nil, err
	}

	hasKey := false
	if key, kerr := apiKey(); kerr == nil {
		hasKey = strings.TrimSpace(key) != ""
	}

	out := make(map[string]CardDTO, len(links))
	for uniqueID, link := range links {
		if strings.TrimSpace(link.RiotID) == "" {
			continue
		}
		card := CardDTO{
			RiotID: link.RiotID,
			Region: link.Region,
			Linked: true,
			HasKey: hasKey,
		}

		id, perr := riot.ParseID(link.RiotID)
		region, rerr := riot.LookupRegion(link.Region)
		switch {
		case perr != nil:
			// Stored but unusable. Said on the card rather than dropped, or the
			// account reads as never linked and there is nothing to explain why.
			card.Error = perr.Error()
		case rerr != nil:
			card.Error = rerr.Error()
		default:
			for _, l := range riot.AllLinks(id, region) {
				card.Links = append(card.Links, LinkDTO{Site: l.Site, Title: string(l.Title), URL: l.URL})
			}
			s.fillFromSnapshot(context.Background(), &card, link)
		}
		out[uniqueID] = card
	}
	return out, nil
}
