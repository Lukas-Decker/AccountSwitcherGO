// Package riotservice is the application's glue around the portable riot
// package: it supplies the HTTP client, the API key and the image cache, and
// exposes the result to the frontend.
//
// The split is deliberate. internal/riot knows how to talk to Riot and nothing
// about this program; everything here is the part that would have to be
// rewritten to move it somewhere else.
package riotservice

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"account-switcher/internal/appclient"
	"account-switcher/internal/basic"
	"account-switcher/internal/credstore"
	"account-switcher/internal/gamestatsimage"
	"account-switcher/internal/riot"
	"account-switcher/internal/security"
)

// PlatformKey is the platform these accounts belong to.
const PlatformKey = "Riot Games"

// credKey names the entry in the OS credential store. One key covers every
// region, which is how Riot issues them.
const credKey = "riot-api-key"

// iconCacheSubdir is where profile icons and rank emblems are cached, under the
// same wwwroot tree the game stats images already use.
const iconCacheSubdir = "riot"

// iconMaxAgeDays bounds how long a cached icon is reused. Profile icons change
// when the user changes them, which is rare, so a week is generous and still
// self-correcting.
const iconMaxAgeDays = 7

// lookupTimeout bounds a manual refresh. Three sequential calls to Riot plus an
// icon fetch, so it is the whole card rather than one request.
const lookupTimeout = 20 * time.Second

// logRiot names this package's lines in the shared log.
func logRiot() *slog.Logger {
	return slog.Default().With("component", "riot")
}

// Service is the Wails-facing surface.
type Service struct{}

// New returns a Service.
func New() *Service { return &Service{} }

// client builds a riot.Client backed by the app's shared HTTP client, so offline
// mode applies here exactly as it does everywhere else.
func (s *Service) client() *riot.Client {
	return riot.NewClient(appclient.Shared, apiKey)
}

// apiKeySource is the credential store, or a stand-in during tests.
//
// The store is machine-global, so without this seam a unit test's behaviour
// depends on whether the developer happens to have a key installed: the same
// test passes on one machine and makes live API calls on another.
var apiKeySource = credentialStoreKey

// apiKey reads the key from whatever source is configured.
func apiKey() (string, error) { return apiKeySource() }

// credentialStoreKey reads the key from the OS credential store at call time.
func credentialStoreKey() (string, error) {
	key, err := credstore.Get(credKey)
	if errors.Is(err, credstore.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(key), nil
}

// LinkDTO is one external profile link.
type LinkDTO struct {
	Site  string `json:"site"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// RegionDTO is one selectable region.
type RegionDTO struct {
	Platform string `json:"platform"`
	Display  string `json:"display"`
}

// RankDTO is one ranked queue's standing.
type RankDTO struct {
	Queue string `json:"queue"`
	Tier  string `json:"tier"`
	// Rank is the division numeral, empty for the tiers that have none.
	// Carried separately from Display so a snapshot can be rebuilt without
	// parsing the rendered string back apart.
	Rank         string `json:"rank"`
	LeaguePoints int    `json:"leaguePoints"`
	Display      string `json:"display"`
	EmblemURL    string `json:"emblemUrl"`
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	WinRate      int    `json:"winRate"`
	HasGames     bool   `json:"hasGames"`
}

// CardDTO is everything the account card renders.
//
// Partial results are expected rather than exceptional: the links half needs no
// key at all, so a card without one is still worth showing. Error carries why
// the keyed half is missing without failing the whole call.
type CardDTO struct {
	RiotID string `json:"riotId"`
	Region string `json:"region"`
	Linked bool   `json:"linked"`
	HasKey bool   `json:"hasKey"`
	Level  int    `json:"level"`
	// IconID is Riot's own id for the profile icon. Carried alongside the URL
	// because the URL is a local cache path once resolved, and the avatar sync
	// needs the id to rebuild it later.
	IconID  int       `json:"iconId"`
	IconURL string    `json:"iconUrl"`
	Ranks   []RankDTO `json:"ranks"`
	Links   []LinkDTO `json:"links"`
	Error   string    `json:"error"`
	// Currency balances from the League Client. HasWallet distinguishes a real
	// balance of zero from never having read one, and WalletAt says when the
	// figures were read, since they move whenever the account is played and the
	// client is the only thing that can report them.
	HasWallet   bool   `json:"hasWallet"`
	BlueEssence int    `json:"blueEssence"`
	RiotPoints  int    `json:"riotPoints"`
	WalletAt    string `json:"walletAt"`
	// Source is where the figures came from: "client", "api" or "snapshot".
	Source string `json:"source"`
	// CapturedAt is when a snapshot was taken, empty when the data is live. The
	// UI shows it so a rank from weeks ago is not read as current.
	CapturedAt string `json:"capturedAt"`
}

// Regions lists the regions the UI offers.
func (s *Service) Regions() []RegionDTO {
	all := riot.Regions()
	out := make([]RegionDTO, 0, len(all))
	for _, r := range all {
		out = append(out, RegionDTO{Platform: r.Platform, Display: r.Display})
	}
	return out
}

// HasAPIKey reports whether a key is stored, without returning it. The key is
// never handed back to the frontend: it goes in, and only its presence comes out.
func (s *Service) HasAPIKey() (bool, error) {
	key, err := apiKey()
	if err != nil {
		return false, err
	}
	return key != "", nil
}

// CredentialStoreAvailable reports whether this machine can store a key at all,
// so the UI can say so before the user pastes one in.
func (s *Service) CredentialStoreAvailable() bool { return credstore.Available() }

// SetAPIKey stores the key, or clears it when given an empty string.
func (s *Service) SetAPIKey(key string) error {
	if err := security.RequireUnlocked(); err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return credstore.Delete(credKey)
	}
	return credstore.Set(credKey, key)
}

// GetAccountLink returns the Riot ID and region stored for a saved account.
func (s *Service) GetAccountLink(uniqueID string) (basic.RiotAccountLink, error) {
	link, _, err := basic.ReadRiotAccountLink(PlatformKey, uniqueID)
	return link, err
}

// SetAccountLink stores the Riot ID and region for a saved account. An empty
// Riot ID clears the link.
func (s *Service) SetAccountLink(uniqueID, riotID, region string) error {
	if err := security.RequireUnlocked(); err != nil {
		return err
	}
	riotID = strings.TrimSpace(riotID)
	if riotID != "" {
		// Parsed before storing so a malformed ID is rejected while the user is
		// looking at the field, not on some later refresh.
		if _, err := riot.ParseID(riotID); err != nil {
			return err
		}
		if _, err := riot.LookupRegion(region); err != nil {
			return err
		}
	}
	return basic.WriteRiotAccountLink(PlatformKey, uniqueID, basic.RiotAccountLink{
		RiotID: riotID,
		Region: strings.TrimSpace(region),
	})
}

// GetCard returns the card for a saved account, fetching live data when a key is
// configured.
func (s *Service) GetCard(uniqueID string) (CardDTO, error) {
	return s.getCard(uniqueID, false)
}

// RefreshCard is the user asking directly, which overrides the throttle and lets
// a development key fill in a snapshot it would not be polled for.
func (s *Service) RefreshCard(uniqueID string) (CardDTO, error) {
	return s.getCard(uniqueID, true)
}

func (s *Service) getCard(uniqueID string, force bool) (CardDTO, error) {
	if err := security.RequireUnlocked(); err != nil {
		return CardDTO{}, err
	}
	link, ok, err := basic.ReadRiotAccountLink(PlatformKey, uniqueID)
	if err != nil {
		return CardDTO{}, err
	}
	if !ok || strings.TrimSpace(link.RiotID) == "" {
		return CardDTO{}, nil
	}

	card := CardDTO{RiotID: link.RiotID, Region: link.Region, Linked: true}

	id, err := riot.ParseID(link.RiotID)
	if err != nil {
		card.Error = err.Error()
		return card, nil
	}
	region, err := riot.LookupRegion(link.Region)
	if err != nil {
		card.Error = err.Error()
		return card, nil
	}

	// Links first: they need no key and no network, so the card is useful even
	// when everything below fails.
	for _, l := range riot.AllLinks(id, region) {
		card.Links = append(card.Links, LinkDTO{Site: l.Site, Title: string(l.Title), URL: l.URL})
	}

	// Whatever was captured the last time this account was used. Shown as-is when
	// nothing better is available, and replaced when it is.
	s.fillFromSnapshot(ctx0(), &card, link)

	client := s.client()
	card.HasKey = client.HasKey()

	// The running client is the better source when it happens to be signed in as
	// this account: current, free, and no key involved.
	if s.fillFromRunningClient(&card, id) {
		// Kept, exactly as an API answer is. This is the only source available
		// without a key, so letting it evaporate when the client closes would
		// leave the account with nothing to show and no way to get it back.
		s.storeSnapshot(uniqueID, card)
		return card, nil
	}

	if !card.HasKey || !s.liveRefreshAllowed(uniqueID, force) {
		// Either no key, a development key that is not polled, or a refresh that
		// came round too soon. The snapshot already filled in stands.
		return card, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()
	if err := s.fillLiveData(ctx, client, &card, id, region); err != nil {
		// Info, not Debug: this is the line that explains an empty card.
		logRiot().Info("Riot API lookup failed", "riotId", card.RiotID, "region", card.Region, "err", err)
		card.Error = err.Error()
		return card, nil
	}
	logRiot().Info("Riot API lookup succeeded",
		"riotId", card.RiotID, "level", card.Level, "ranks", len(card.Ranks))
	// Kept, so the figures survive the session and the next read has something to
	// show without spending another call.
	s.storeSnapshot(uniqueID, card)
	return card, nil
}

// fillLiveData performs the three keyed calls and caches the icon.
//
// Ordered by dependency: the Riot ID buys a PUUID, the PUUID buys the profile
// and the ranks. A failure at any step leaves what was already collected in
// place rather than discarding the card.
func (s *Service) fillLiveData(ctx context.Context, client *riot.Client, card *CardDTO, id riot.ID, region riot.Region) error {
	acc, err := client.AccountByRiotID(ctx, id, region)
	if err != nil {
		return err
	}
	// Riot is the authority on capitalisation, so the stored spelling is
	// refreshed from what it returns.
	card.RiotID = acc.ID().String()

	summoner, err := client.SummonerByPUUID(ctx, acc.PUUID, region)
	if err != nil {
		return err
	}
	// Replaces the snapshot rather than adding to it: both describe the same
	// account, and appending would list every rank twice.
	card.Ranks = nil
	card.Source = "api"
	card.CapturedAt = ""
	card.Level = summoner.SummonerLevel
	card.IconID = summoner.ProfileIconID
	card.IconURL = s.cachedImage(ctx, riot.ProfileIconURLLatest(summoner.ProfileIconID))

	entries, err := client.LeagueEntriesByPUUID(ctx, acc.PUUID, region)
	if err != nil {
		return err
	}
	s.appendRanks(ctx, card, entries, riot.QueueSoloDuo, riot.QueueFlex)

	// TFT is a separate API, and a separate failure: an account that plays League
	// but not TFT is the common case, so a miss here must not discard the League
	// standings already collected.
	tft, err := client.TFTLeagueEntriesByPUUID(ctx, acc.PUUID, region)
	if err != nil {
		logRiot().Debug("tft ranks unavailable", "err", err)
		return nil
	}
	s.appendRanks(ctx, card, tft, riot.QueueTFT, riot.QueueTFTDoubleUp)
	return nil
}

// appendRanks adds the named queues, in the order given, skipping any the
// account has no standing in.
func (s *Service) appendRanks(ctx context.Context, card *CardDTO, entries []riot.LeagueEntry, queues ...string) {
	for _, queue := range queues {
		e, ok := riot.EntryForQueue(entries, queue)
		if !ok {
			continue
		}
		rate, hasGames := e.WinRate()
		card.Ranks = append(card.Ranks, RankDTO{
			Queue:        queue,
			Tier:         e.Tier,
			Rank:         e.Rank,
			LeaguePoints: e.LeaguePoints,
			Display:      e.Display(),
			EmblemURL:    s.cachedImage(ctx, riot.RankedEmblemURL(e.Tier)),
			Wins:         e.Wins,
			Losses:       e.Losses,
			WinRate:      int(rate + 0.5),
			HasGames:     hasGames,
		})
	}
}

// cachedImage publishes a remote image through the existing disk cache and
// returns the local URL, or "" when it could not be fetched.
//
// Reuses the game stats cache rather than adding another: it already keys by URL
// hash, ages entries out and serves from wwwroot, which is all this needs.
func (s *Service) cachedImage(ctx context.Context, remoteURL string) string {
	if strings.TrimSpace(remoteURL) == "" {
		return ""
	}
	local, err := gamestatsimage.DownloadIfNeeded(ctx, appclient.Shared, iconCacheSubdir, remoteURL, iconMaxAgeDays)
	if err != nil {
		return ""
	}
	return local
}

// ctx0 is a short-lived context for cache lookups that only touch disk.
func ctx0() context.Context { return context.Background() }

// fillWallet copies stored balances onto the card, if any were ever read.
func fillWallet(card *CardDTO, link basic.RiotAccountLink) {
	if link.WalletAt.IsZero() {
		return
	}
	card.HasWallet = true
	card.BlueEssence = link.BlueEssence
	card.RiotPoints = link.RiotPoints
	card.WalletAt = link.WalletAt.UTC().Format(time.RFC3339)
}

// fillFromSnapshot populates the card from what was last captured for the
// account, and records when that was.
func (s *Service) fillFromSnapshot(ctx context.Context, card *CardDTO, link basic.RiotAccountLink) {
	// Before the check below: the balances have their own time, so a record that
	// has them and nothing else is still worth showing.
	fillWallet(card, link)
	if link.CapturedAt.IsZero() {
		return
	}
	card.Source = "snapshot"
	card.CapturedAt = link.CapturedAt.UTC().Format(time.RFC3339)
	card.Level = link.Level
	card.IconID = link.IconID
	if link.IconID > 0 {
		card.IconURL = s.cachedImage(ctx, riot.ProfileIconURLLatest(link.IconID))
	}
	for _, r := range link.Ranks {
		e := riot.LeagueEntry{
			QueueType:    r.Queue,
			Tier:         r.Tier,
			Rank:         r.Rank,
			LeaguePoints: r.LeaguePoints,
			Wins:         r.Wins,
			Losses:       r.Losses,
		}
		rate, hasGames := e.WinRate()
		card.Ranks = append(card.Ranks, RankDTO{
			Queue:        r.Queue,
			Tier:         r.Tier,
			Rank:         r.Rank,
			LeaguePoints: r.LeaguePoints,
			Display:      e.Display(),
			EmblemURL:    s.cachedImage(ctx, riot.RankedEmblemURL(r.Tier)),
			Wins:         r.Wins,
			Losses:       r.Losses,
			WinRate:      int(rate + 0.5),
			HasGames:     hasGames,
		})
	}
}
