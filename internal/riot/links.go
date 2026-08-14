package riot

import "strings"

// Title is one of the Riot games an account can be looked up for.
type Title string

const (
	TitleLoL      Title = "lol"
	TitleTFT      Title = "tft"
	TitleValorant Title = "valorant"
)

// Link is one external profile page for an account.
type Link struct {
	// Site is the display name, e.g. "op.gg".
	Site string
	// Title is the game the link is about.
	Title Title
	// URL is ready to open.
	URL string
}

// Links returns every external profile page known for id in the given title.
//
// Pure string building: no request is made and no API key is involved, so this
// works for any account the user can name, whether or not a key is configured.
//
// Each site is fed from the encoder that matches how it reads a Riot ID. They
// genuinely disagree: op.gg joins the halves with a dash, tracker.gg keeps an
// encoded '#', and both need the original characters rather than a slug.
func Links(id ID, region Region, title Title) []Link {
	if !id.Valid() {
		return nil
	}
	slug := id.OPGGSlug()
	r := region.OPGG

	switch title {
	case TitleLoL:
		return []Link{
			{Site: "op.gg", Title: title, URL: "https://op.gg/lol/summoners/" + r + "/" + slug},
			{Site: "u.gg", Title: title, URL: "https://u.gg/lol/profile/" + region.Platform + "/" + slug + "/overview"},
			{Site: "deeplol.gg", Title: title, URL: "https://www.deeplol.gg/summoner/" + r + "/" + slug},
			{Site: "porofessor.gg", Title: title, URL: "https://porofessor.gg/live/" + r + "/" + slug},
		}
	case TitleTFT:
		return []Link{
			{Site: "op.gg", Title: title, URL: "https://op.gg/tft/summoners/" + r + "/" + slug},
			{Site: "lolchess.gg", Title: title, URL: "https://lolchess.gg/profile/" + r + "/" + slug},
		}
	case TitleValorant:
		// VALORANT's official API is far more restricted than the League ones, so
		// this title is links only. tracker.gg is region-agnostic: the Riot ID is
		// unique across regions and the site resolves it itself.
		return []Link{
			{Site: "tracker.gg", Title: title, URL: "https://tracker.gg/valorant/profile/riot/" + id.TrackerSegment() + "/overview"},
			{Site: "op.gg", Title: title, URL: "https://op.gg/valorant/profile/" + id.TrackerSegment()},
		}
	}
	return nil
}

// AllLinks returns the links for every supported title, in a stable order.
func AllLinks(id ID, region Region) []Link {
	var out []Link
	for _, t := range []Title{TitleLoL, TitleTFT, TitleValorant} {
		out = append(out, Links(id, region, t)...)
	}
	return out
}

// ParseTitle reads a title name, defaulting to League.
func ParseTitle(s string) Title {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "tft", "teamfight tactics":
		return TitleTFT
	case "valorant", "val":
		return TitleValorant
	default:
		return TitleLoL
	}
}
