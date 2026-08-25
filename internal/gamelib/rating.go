package gamelib

import (
	"regexp"
	"strings"
)

// Adult content is hidden by default, and the filter has to be turned off to
// see it.
//
// There is no single authority for this. Steam publishes content descriptors
// but only through a per-app store call, which is thousands of requests for a
// real library and rate limited besides. IGDB has an erotic theme but only for
// accounts that have supplied credentials. So the default answer comes from the
// title itself, which is free, offline, and right about the obvious cases, and
// anything it gets wrong the user can correct per game.
//
// The bar for matching here is deliberately high. A false positive hides a game
// someone owns and leaves them hunting for it, which is worse than a false
// negative they can hide themselves.

// nsfwTitlePatterns match titles that announce themselves.
//
// Whole words only, so "Assassin" does not match "ass" and "Analogue" does not
// match "anal". Anything subtler than this is not decidable from a name and is
// left to the catalogues and to the user.
var nsfwTitlePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bhentai\b`),
	regexp.MustCompile(`(?i)\bnsfw\b`),
	regexp.MustCompile(`(?i)\beroge\b`),
	regexp.MustCompile(`(?i)\bporn(o|ographic)?\b`),
	regexp.MustCompile(`(?i)\bsex(y|ual)?\s+(simulator|game|adventure|story)\b`),
	regexp.MustCompile(`(?i)\badults?\s*only\b`),
	regexp.MustCompile(`(?i)\b18\+`),
	regexp.MustCompile(`(?i)\buncensored\b`),
	regexp.MustCompile(`(?i)\bwaifu\s+(sex|nude|naked)`),
	regexp.MustCompile(`(?i)\bnude\b`),
	regexp.MustCompile(`(?i)\bnudity\b`),
	regexp.MustCompile(`(?i)\bstrip\s*(poker|club)\b`),
}

// looksAdult reports whether a title announces itself as adult content.
func looksAdult(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, re := range nsfwTitlePatterns {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// applyRatings marks each game and applies the user's own answers.
//
// A stored override always wins: the heuristic is a starting point, and once
// somebody has told the app what a game is there is no reason to keep guessing.
func applyRatings(platformKey string, games []Game) []Game {
	prefs := PrefsFor(platformKey)
	for i := range games {
		adult := looksAdult(games[i].Name)
		if v, ok := prefs.NSFWOverride[games[i].GameID]; ok {
			adult = v
			games[i].AdultOverridden = true
		}
		games[i].Adult = adult
	}
	return games
}

// applyPrefs folds the stored per-game choices into a resolved list.
//
// Hidden games are marked rather than dropped, because the view offers a filter
// to show them again and doing that must not need a rescan.
func applyPrefs(platformKey string, games []Game) []Game {
	prefs := PrefsFor(platformKey)
	hidden := make(map[string]struct{}, len(prefs.Hidden))
	for _, id := range prefs.Hidden {
		hidden[id] = struct{}{}
	}
	for i := range games {
		if _, ok := hidden[games[i].GameID]; ok {
			games[i].Hidden = true
		}
		if pinned := strings.TrimSpace(prefs.ArtOverride[games[i].GameID]); pinned != "" {
			games[i].ArtURL = pinned
			games[i].ArtPinned = true
		}
	}
	return applyRatings(platformKey, games)
}
