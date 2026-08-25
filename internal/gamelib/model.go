// Package gamelib resolves which games exist on this machine and which account
// on which platform each one belongs to.
//
// Every launcher records ownership differently, and most of them record it
// badly: some name the owning account outright, some only leave per-account
// traces that imply the game was run, and some keep nothing per-account at all.
// Rather than pretend those are the same fact, a resolver reports what it
// actually saw as an [Observation] carrying its [Source] and [Confidence], and
// [Builder] merges the observations into one game list. A caller can then tell
// "this account installed this game" from "some account on this machine once
// touched this game", which is the difference between a switch that works and
// one that logs into the wrong account.
package gamelib

import (
	"sort"
	"strings"
	"time"
)

// Confidence grades how firmly a source ties a game to an account. Merging
// keeps the highest-graded claim per account, so a guess never overwrites a
// fact the launcher wrote down itself.
type Confidence int

const (
	// ConfidenceNone marks a game with no account attached at all.
	ConfidenceNone Confidence = iota
	// ConfidenceInferred is context, not evidence: the platform has exactly one
	// known account, so an account-less install probably belongs to it.
	ConfidenceInferred
	// ConfidenceWeak comes from leftovers that imply use rather than ownership,
	// such as a per-account save folder that survives an uninstall.
	ConfidenceWeak
	// ConfidenceStrong comes from a launcher's own per-account library data.
	ConfidenceStrong
	// ConfidenceExact is the launcher naming the owning account for this game.
	ConfidenceExact
)

// String renders a confidence for the UI and for logs.
func (c Confidence) String() string {
	switch c {
	case ConfidenceInferred:
		return "inferred"
	case ConfidenceWeak:
		return "weak"
	case ConfidenceStrong:
		return "strong"
	case ConfidenceExact:
		return "exact"
	default:
		return "none"
	}
}

// Source identifies the file, database, or endpoint an observation came from,
// so a surprising result can be traced back to the thing that claimed it.
type Source string

// Steam sources, strongest first.
const (
	// SourceSteamAppManifest is appmanifest_<id>.acf, whose LastOwner field is
	// the SteamID64 that installed the game.
	SourceSteamAppManifest Source = "steam:appmanifest"
	// SourceSteamCommunityXML is the public community profile game list, the
	// only keyless view of a library that was never installed here.
	SourceSteamCommunityXML Source = "steam:community-xml"
	// SourceSteamLocalConfig is userdata/<id32>/config/localconfig.vdf, which
	// keeps playtime and last-played per app for that one account.
	SourceSteamLocalConfig Source = "steam:localconfig"
	// SourceSteamSharedConfig is the roaming sharedconfig.vdf, which holds the
	// account's own categories, favourites, and hidden flags per app.
	SourceSteamSharedConfig Source = "steam:sharedconfig"
	// SourceSteamUserdata is a bare userdata/<id32>/<appid> folder, which only
	// proves the account ran the app here at some point.
	SourceSteamUserdata Source = "steam:userdata"
	// SourceSteamAppList is the downloaded app catalogue, used for names only.
	SourceSteamAppList Source = "steam:applist"
)

// Launcher sources for the other platforms.
const (
	SourceEpicManifest    Source = "epic:manifest"
	SourceGOGGalaxyDB     Source = "gog:galaxy-db"
	SourceEAInstallReg    Source = "ea:registry"
	SourceOriginManifest  Source = "origin:mfst"
	SourceUbisoftReg      Source = "ubisoft:registry"
	SourceBattleNetConfig Source = "battlenet:config"
	SourceBattleNetReg    Source = "battlenet:registry"
	SourceRiotMetadata    Source = "riot:metadata"
	SourceRockstarReg     Source = "rockstar:registry"
	SourceOculusManifest  Source = "oculus:manifest"
	// SourceDescriptorExe is the platform's own executable from Platforms.json,
	// which is all there is to find for a launcher that ships a single game.
	SourceDescriptorExe Source = "descriptor:exe"
)

// Observation is one source's claim about one game, optionally tied to one
// account. Resolvers emit these instead of finished games so that a game seen
// by four sources stays one entry with four pieces of evidence.
type Observation struct {
	PlatformKey string
	// GameID is unique within the platform: a Steam appid, an Epic AppName, a
	// GOG product id. Resolvers must keep it stable across runs.
	GameID string
	Name   string
	// ArtURL is a path the webview can load, already published into wwwroot.
	ArtURL string

	Installed   bool
	InstallPath string
	SizeOnDisk  int64

	// AccountID is empty when the source knows the game but not the owner.
	AccountID   string
	AccountName string

	Source     Source
	Confidence Confidence

	// InstalledBy marks this account as the one that put the game on disk,
	// which is a stronger claim than merely owning it.
	InstalledBy bool

	PlaytimeMinutes int64
	LastPlayed      time.Time
}

// Ownership is one account's resolved claim on a game, after merging.
type Ownership struct {
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
	// Source and Confidence describe the strongest claim seen for this account.
	Source     Source `json:"source"`
	Confidence string `json:"confidence"`
	// InstalledBy is true when this account is the one that installed the game.
	InstalledBy     bool   `json:"installedBy"`
	PlaytimeMinutes int64  `json:"playtimeMinutes"`
	LastPlayed      string `json:"lastPlayed"`

	confidence Confidence
	lastPlayed time.Time
}

// Game is one resolved title with every account known to have it.
type Game struct {
	PlatformKey string `json:"platformKey"`
	GameID      string `json:"gameId"`
	Name        string `json:"name"`
	ArtURL      string `json:"artUrl"`

	Installed   bool   `json:"installed"`
	InstallPath string `json:"installPath"`
	SizeOnDisk  int64  `json:"sizeOnDisk"`

	// Hidden is set when the user has hidden this game. It is marked rather
	// than dropped so unhiding needs no rescan.
	Hidden bool `json:"hidden"`
	// NSFW marks sexual content the view keeps behind a filter by default.
	// Narrower than an age rating on purpose: a game rated 18+ for violence is
	// not what this hides.
	NSFW bool `json:"nsfw"`
	// NSFWOverridden says the flag came from the user rather than a guess, so
	// the view can offer to undo it rather than re-asserting it.
	NSFWOverridden bool `json:"nsfwOverridden"`
	// ArtPinned says the artwork is the user's choice, not the chain's.
	ArtPinned bool `json:"artPinned"`
	// ArtOptions are the other artwork the chain could have used, so the user
	// can pick a different one. Empty when only one source produced anything.
	ArtOptions []ArtOption `json:"artOptions"`

	Owners []Ownership `json:"owners"`
	// Sources lists every source that contributed, for troubleshooting a game
	// that resolved to an unexpected account or to none.
	Sources []Source `json:"sources"`
}

// ArtOption is one artwork the chain published or could publish for a game.
type ArtOption struct {
	// URL is a wwwroot path the view can show immediately.
	URL string `json:"url"`
	// Tier names the shape, so the picker can say what each option is.
	Tier string `json:"tier"`
	// Source is where it came from, for the same reason.
	Source string `json:"source"`
}

// InstalledOwner returns the account that installed the game, when a source
// named one. Switching before launching an installed game should target this
// account, since it is the one the launcher already has the files licensed to.
func (g Game) InstalledOwner() (Ownership, bool) {
	for _, o := range g.Owners {
		if o.InstalledBy {
			return o, true
		}
	}
	return Ownership{}, false
}

// Builder accumulates observations and merges them into games.
//
// It is not safe for concurrent use; resolvers run in parallel but each fills
// its own builder, and [Merge] joins the results.
type Builder struct {
	games map[string]*Game
	// owners is keyed by game key then account id so that repeated claims about
	// the same pair collapse instead of stacking up.
	owners map[string]map[string]*Ownership
	order  []string
}

// NewBuilder returns an empty builder.
func NewBuilder() *Builder {
	return &Builder{
		games:  map[string]*Game{},
		owners: map[string]map[string]*Ownership{},
	}
}

func gameKey(platformKey, gameID string) string {
	return strings.ToLower(strings.TrimSpace(platformKey)) + "\x00" + strings.TrimSpace(gameID)
}

// Observe folds one claim into the builder. Observations with no platform or no
// game id are dropped, since nothing downstream could address them.
func (b *Builder) Observe(obs Observation) {
	obs.PlatformKey = strings.TrimSpace(obs.PlatformKey)
	obs.GameID = strings.TrimSpace(obs.GameID)
	if obs.PlatformKey == "" || obs.GameID == "" {
		return
	}
	key := gameKey(obs.PlatformKey, obs.GameID)

	g, ok := b.games[key]
	if !ok {
		g = &Game{PlatformKey: obs.PlatformKey, GameID: obs.GameID}
		b.games[key] = g
		b.owners[key] = map[string]*Ownership{}
		b.order = append(b.order, key)
	}

	// A later source only fills gaps in the game's own fields. Names in
	// particular arrive from several places, and the first non-empty one wins,
	// which is why resolvers emit their best catalogue before their fallbacks.
	if name := strings.TrimSpace(obs.Name); name != "" && g.Name == "" {
		g.Name = name
	}
	if art := strings.TrimSpace(obs.ArtURL); art != "" && g.ArtURL == "" {
		g.ArtURL = art
	}
	if obs.Installed {
		g.Installed = true
	}
	if p := strings.TrimSpace(obs.InstallPath); p != "" && g.InstallPath == "" {
		g.InstallPath = p
	}
	if obs.SizeOnDisk > g.SizeOnDisk {
		g.SizeOnDisk = obs.SizeOnDisk
	}
	if obs.Source != "" && !containsSource(g.Sources, obs.Source) {
		g.Sources = append(g.Sources, obs.Source)
	}

	accountID := strings.TrimSpace(obs.AccountID)
	if accountID == "" {
		return
	}

	cur, seen := b.owners[key][accountID]
	if !seen {
		cur = &Ownership{AccountID: accountID}
		b.owners[key][accountID] = cur
	}
	// The strongest claim wins the attribution, but weaker ones still carry
	// facts the strong one lacks: appmanifest names the owner and knows nothing
	// about playtime, localconfig is the reverse.
	if !seen || obs.Confidence > cur.confidence {
		cur.confidence = obs.Confidence
		cur.Source = obs.Source
	}
	if name := strings.TrimSpace(obs.AccountName); name != "" && cur.AccountName == "" {
		cur.AccountName = name
	}
	if obs.InstalledBy {
		cur.InstalledBy = true
	}
	if obs.PlaytimeMinutes > cur.PlaytimeMinutes {
		cur.PlaytimeMinutes = obs.PlaytimeMinutes
	}
	if obs.LastPlayed.After(cur.lastPlayed) {
		cur.lastPlayed = obs.LastPlayed
	}
}

func containsSource(list []Source, s Source) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Games returns the merged list, sorted so the entries most worth acting on
// come first: installed games, then the ones more accounts own, then by name.
func (b *Builder) Games() []Game {
	out := make([]Game, 0, len(b.order))
	for _, key := range b.order {
		g := b.games[key]
		owners := make([]Ownership, 0, len(b.owners[key]))
		for _, o := range b.owners[key] {
			o.Confidence = o.confidence.String()
			if !o.lastPlayed.IsZero() {
				o.LastPlayed = o.lastPlayed.UTC().Format(time.RFC3339)
			}
			owners = append(owners, *o)
		}
		sortOwners(owners)
		g.Owners = owners
		if g.Name == "" {
			g.Name = g.GameID
		}
		out = append(out, *g)
	}
	SortGames(out)
	return out
}

// sortOwners puts the installing account first, then the firmest claims, then
// the most played, so a picker's first entry is its best guess.
func sortOwners(owners []Ownership) {
	sort.SliceStable(owners, func(i, j int) bool {
		a, b := owners[i], owners[j]
		if a.InstalledBy != b.InstalledBy {
			return a.InstalledBy
		}
		if a.confidence != b.confidence {
			return a.confidence > b.confidence
		}
		if a.PlaytimeMinutes != b.PlaytimeMinutes {
			return a.PlaytimeMinutes > b.PlaytimeMinutes
		}
		return strings.ToLower(a.AccountName) < strings.ToLower(b.AccountName)
	})
}

// SortGames orders a merged list for display.
func SortGames(games []Game) {
	sort.SliceStable(games, func(i, j int) bool {
		a, b := games[i], games[j]
		if a.Installed != b.Installed {
			return a.Installed
		}
		if len(a.Owners) != len(b.Owners) {
			return len(a.Owners) > len(b.Owners)
		}
		if !strings.EqualFold(a.Name, b.Name) {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		return a.PlatformKey < b.PlatformKey
	})
}

// Merge folds several resolver results into one list, re-running the same
// precedence rules so a game owned on two platforms stays two entries while a
// game two resolvers both saw stays one.
func Merge(lists ...[]Game) []Game {
	b := NewBuilder()
	for _, list := range lists {
		for _, g := range list {
			base := Observation{
				PlatformKey: g.PlatformKey,
				GameID:      g.GameID,
				Name:        g.Name,
				ArtURL:      g.ArtURL,
				Installed:   g.Installed,
				InstallPath: g.InstallPath,
				SizeOnDisk:  g.SizeOnDisk,
			}
			for _, o := range g.Owners {
				obs := base
				obs.AccountID = o.AccountID
				obs.AccountName = o.AccountName
				obs.Source = o.Source
				obs.Confidence = confidenceFromString(o.Confidence)
				obs.InstalledBy = o.InstalledBy
				obs.PlaytimeMinutes = o.PlaytimeMinutes
				obs.LastPlayed = parseRFC3339(o.LastPlayed)
				b.Observe(obs)
			}
			for _, s := range g.Sources {
				obs := base
				obs.Source = s
				b.Observe(obs)
			}
			if len(g.Owners) == 0 && len(g.Sources) == 0 {
				b.Observe(base)
			}
		}
	}
	return b.Games()
}

func confidenceFromString(s string) Confidence {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "inferred":
		return ConfidenceInferred
	case "weak":
		return ConfidenceWeak
	case "strong":
		return ConfidenceStrong
	case "exact":
		return ConfidenceExact
	default:
		return ConfidenceNone
	}
}

func parseRFC3339(s string) time.Time {
	if strings.TrimSpace(s) == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
