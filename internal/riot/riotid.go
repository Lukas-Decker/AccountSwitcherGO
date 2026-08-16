// Package riot reads public Riot Games account data and builds links to the
// community sites that display it.
//
// The package is deliberately free of any dependency on the application around
// it: callers supply an *http.Client, a way to obtain an API key, and somewhere
// to put images. That keeps it usable from another program, and keeps this
// program's offline mode, proxy and caching decisions in one place rather than
// duplicated here.
//
// Read-only. Nothing here writes to a Riot account or automates a game client.
package riot

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Riot's own limits on the two halves of a Riot ID.
const (
	MinGameNameLen = 3
	MaxGameNameLen = 16
	MinTagLineLen  = 3
	MaxTagLineLen  = 5
)

var (
	ErrEmptyRiotID = errors.New("riot: empty Riot ID")
	ErrMissingTag  = errors.New("riot: Riot ID needs a #TAG")
	ErrGameNameLen = fmt.Errorf("riot: game name must be %d-%d characters", MinGameNameLen, MaxGameNameLen)
	ErrTagLineLen  = fmt.Errorf("riot: tagline must be %d-%d characters", MinTagLineLen, MaxTagLineLen)
	ErrTagHasHash  = errors.New("riot: tagline cannot contain '#'")
)

// ID is a parsed Riot ID: the game name and tagline as the user actually has
// them, with no substitutions applied.
//
// Game names are permissive. Spaces, accents and non-Latin scripts are all legal,
// and the same account is written three different ways depending on where the
// link points. Storing the true characters and encoding per destination is the
// only approach that survives that; slugifying once into dashes or underscores
// loses information that cannot be recovered.
type ID struct {
	GameName string
	TagLine  string
}

// sanitizeIDInput drops the invisible characters a copied Riot ID arrives
// wrapped in, and normalises the spaces.
//
// Riot IDs mix scripts, so the League client, op.gg and most chat apps wrap them
// in bidi isolates (U+2066..U+2069) to stop the surrounding text reordering
// around them. Copying one brings those along invisibly, and counting them
// against the 16-character limit rejects a name that is plainly short enough:
// "Average Tibbers#TR1" is 15 characters and was reported as 17.
//
// Non-breaking and other exotic spaces become plain ones for the same class of
// reason: they look like a space, Riot stores a space, and a name pasted with
// U+00A0 in it would be stored as something no lookup can match.
func sanitizeIDInput(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Cf, r):
			// Format characters: bidi controls, zero-width spaces, the BOM. None
			// of them are part of any name, and all of them are invisible, so
			// there is nothing for the user to see and delete.
			continue
		case unicode.Is(unicode.Zs, r):
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// ParseID reads "GameName#TAG".
//
// The '#' is only ever a separator, so the split is on the first one and a second
// '#' is an error rather than part of the tagline. Lengths are counted in runes:
// a name of five kanji is five characters, not fifteen bytes.
func ParseID(s string) (ID, error) {
	s = sanitizeIDInput(s)
	if s == "" {
		return ID{}, ErrEmptyRiotID
	}
	name, tag, found := strings.Cut(s, "#")
	if !found {
		return ID{}, ErrMissingTag
	}
	name = strings.TrimSpace(name)
	tag = strings.TrimSpace(tag)
	if strings.Contains(tag, "#") {
		return ID{}, ErrTagHasHash
	}
	if n := utf8.RuneCountInString(name); n < MinGameNameLen || n > MaxGameNameLen {
		return ID{}, fmt.Errorf("%w (got %d)", ErrGameNameLen, n)
	}
	if n := utf8.RuneCountInString(tag); n < MinTagLineLen || n > MaxTagLineLen {
		return ID{}, fmt.Errorf("%w (got %d)", ErrTagLineLen, n)
	}
	return ID{GameName: name, TagLine: tag}, nil
}

// String renders the canonical "GameName#TAG" form for display and storage.
func (id ID) String() string {
	return id.GameName + "#" + id.TagLine
}

// Valid reports whether the ID has both halves.
func (id ID) Valid() bool {
	return id.GameName != "" && id.TagLine != ""
}

// APIPathSegments returns the two values as Riot's account-v1 endpoint wants
// them, each escaped as a single path segment.
//
// url.PathEscape, not QueryEscape: a space in a path segment is %20, and
// QueryEscape would turn it into '+', which Riot reads as a literal plus and
// answers 404 for.
func (id ID) APIPathSegments() (gameName, tagLine string) {
	return url.PathEscape(id.GameName), url.PathEscape(id.TagLine)
}

// OPGGSlug renders the "{name}-{tag}" form op.gg uses in its path.
//
// The separator is a dash, and a game name may itself contain one, so the result
// is ambiguous on its face; SplitOPGGSlug resolves it by splitting on the last
// dash, which is what op.gg itself does.
func (id ID) OPGGSlug() string {
	return url.PathEscape(id.GameName + "-" + id.TagLine)
}

// SplitOPGGSlug recovers a Riot ID from an op.gg slug.
//
// Splits on the LAST dash: "Bo-Bo-EUW" is the account "Bo-Bo#EUW", not "Bo"
// with a tagline of "Bo-EUW". Only the tagline is guaranteed dash-free, so only
// the final separator is unambiguous.
func SplitOPGGSlug(slug string) (ID, error) {
	decoded, err := url.PathUnescape(strings.TrimSpace(slug))
	if err != nil {
		return ID{}, fmt.Errorf("riot: decode op.gg slug: %w", err)
	}
	i := strings.LastIndex(decoded, "-")
	if i <= 0 || i == len(decoded)-1 {
		return ID{}, ErrMissingTag
	}
	return ParseID(decoded[:i] + "#" + decoded[i+1:])
}

// TrackerSegment renders the single path segment tracker.gg expects, in which
// the '#' survives as an encoded character rather than becoming a separator.
func (id ID) TrackerSegment() string {
	return url.PathEscape(id.GameName) + "%23" + url.PathEscape(id.TagLine)
}
