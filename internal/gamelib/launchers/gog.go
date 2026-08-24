package launchers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"account-switcher/internal/gamelib"

	"github.com/tidwall/gjson"
	_ "modernc.org/sqlite"
)

// GOGPlatformKey matches the Platforms.json entry.
const GOGPlatformKey = "GOG Galaxy"

// GOG resolves the GOG Galaxy library.
func GOG() gamelib.Resolver {
	return gamelib.ResolverFunc{Key: GOGPlatformKey, Fn: resolveGOG}
}

// galaxyDBPath is Galaxy's library database, shared by every user of the
// machine and keyed internally by GOG user id.
func galaxyDBPath() string {
	pd := programData()
	if pd == "" {
		return ""
	}
	return filepath.Join(pd, "GOG.com", "Galaxy", "storage", "galaxy-2.0.db")
}

// resolveGOG reads Galaxy's own library database.
//
// Galaxy is the one launcher here that keeps a real per-account library on
// disk: owned titles, installs, and playtime all carry the GOG user id, and
// that id is the same value the switcher stores as the account's unique id. So
// unlike Epic or Ubisoft, ownership here is read rather than guessed.
func resolveGOG(ctx context.Context, opts gamelib.Options) (gamelib.Result, error) {
	res := gamelib.Result{PlatformKey: GOGPlatformKey}

	dbPath := galaxyDBPath()
	if !fileExists(dbPath) {
		res.Unsupported = true
		return res, nil
	}

	// Galaxy holds the database open, and a WAL-mode reader on a live file can
	// see a torn view or fail outright. A snapshot copy is read instead, which
	// also guarantees this never writes to or locks Galaxy's own storage.
	snapshot, cleanup, err := snapshotSQLite(dbPath)
	if err != nil {
		return res, fmt.Errorf("copy Galaxy database: %w", err)
	}
	defer cleanup()

	db, err := sql.Open("sqlite", "file:"+strings.ReplaceAll(snapshot, `\`, `/`)+"?mode=ro")
	if err != nil {
		return res, fmt.Errorf("open Galaxy database: %w", err)
	}
	defer func() { _ = db.Close() }()

	b := gamelib.NewBuilder()
	titles := gogTitles(db)
	installed := gogInstalled(db)

	// Owned rows come first so that every account's library is present before
	// the install and playtime passes attach their extra facts.
	owned := gogOwnedByUser(db)
	for userID, keys := range owned {
		name := opts.KnownAccounts[userID]
		for _, key := range keys {
			productID, ok := gogProductID(key)
			if !ok {
				continue
			}
			_, isInstalled := installed[productID]
			b.Observe(gamelib.Observation{
				PlatformKey: GOGPlatformKey,
				GameID:      productID,
				Name:        titles[key],
				AccountID:   userID,
				AccountName: name,
				Installed:   isInstalled,
				InstallPath: installed[productID],
				Source:      gamelib.SourceGOGGalaxyDB,
				Confidence:  gamelib.ConfidenceExact,
			})
		}
	}

	for productID, path := range installed {
		obs := gamelib.Observation{
			PlatformKey: GOGPlatformKey,
			GameID:      productID,
			Name:        titles["gog_"+productID],
			Installed:   true,
			InstallPath: path,
			Source:      gamelib.SourceGOGGalaxyDB,
		}
		// An install with no owned row belongs to an account Galaxy has since
		// forgotten, so fall back to the same inference the other launchers use.
		attributeInstall(&obs, opts)
		b.Observe(obs)
	}

	for _, gt := range gogGameTimes(db) {
		productID, ok := gogProductID(gt.releaseKey)
		if !ok {
			continue
		}
		b.Observe(gamelib.Observation{
			PlatformKey:     GOGPlatformKey,
			GameID:          productID,
			AccountID:       gt.userID,
			AccountName:     opts.KnownAccounts[gt.userID],
			Source:          gamelib.SourceGOGGalaxyDB,
			Confidence:      gamelib.ConfidenceStrong,
			PlaytimeMinutes: gt.minutes,
			LastPlayed:      gt.lastPlayed,
		})
	}

	applyGOGArt(ctx, b, db, installed, opts.AllowNetwork)

	res.Games = b.Games()
	return res, nil
}

// applyGOGArt resolves cover art from Galaxy's own image references.
//
// Galaxy stores artwork as URLs rather than files, so unlike Steam there is
// nothing cached on disk to copy except whatever the publisher dropped in the
// install folder. The URLs are public and need no session.
func applyGOGArt(ctx context.Context, b *gamelib.Builder, db *sql.DB, installed map[string]string, allowNetwork bool) {
	portrait, other := gogImageURLs(db)
	games := b.Games()
	if len(games) == 0 {
		return
	}
	sources := make([]artSource, 0, len(games))
	for _, g := range games {
		key := "gog_" + g.GameID
		src := artSource{
			gameID:   g.GameID,
			portrait: portrait[key],
			remote:   other[key],
		}
		// An owned but uninstalled game has no folder, and joining onto an
		// empty path would produce a bare filename that resolves against the
		// working directory rather than resolving to nothing.
		if installPath := strings.TrimSpace(installed[g.GameID]); installPath != "" {
			// GOG writes the game's own icon beside the executable as
			// goggame-<id>.ico, which is exact and needs no network.
			src.local = append([]string{filepath.Join(installPath, "goggame-"+g.GameID+".ico")}, installDirIcons(installPath)...)
			src.exe = exeForIcon(installPath, "")
		}
		sources = append(sources, src)
	}
	applyLauncherArt(ctx, b, GOGPlatformKey, sources, gamelib.SourceGOGGalaxyDB, allowNetwork)
}

// gogImageURLs maps release key to artwork URLs, split by shape.
//
// The images live inside the same JSON blobs as the titles, one row per field
// per game, so they come out of GamePieces the same way. verticalCover is the
// only one drawn for a 2:3 tile, so it is kept apart from the rest: it is worth
// a round trip even when the game's install folder has an icon sitting in it,
// and the others are not.
func gogImageURLs(db *sql.DB) (portrait, other map[string][]string) {
	portrait = map[string][]string{}
	other = map[string][]string{}
	rows, err := db.Query(`
		SELECT gp.releaseKey, gp.value
		FROM GamePieces gp
		JOIN GamePieceTypes gpt ON gpt.id = gp.gamePieceTypeId
		WHERE gpt.type IN ('originalImages', 'images', 'meta', 'originalMeta')`)
	if err != nil {
		return portrait, other
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		if url := strings.TrimSpace(gjson.Get(value, "verticalCover").String()); url != "" {
			portrait[key] = appendUniqueURL(portrait[key], url)
		}
		// The rest are square or wide, in descending order of how well they
		// survive being cropped to a portrait tile.
		for _, field := range []string{"squareIcon", "logo", "icon", "background"} {
			url := strings.TrimSpace(gjson.Get(value, field).String())
			if url == "" {
				continue
			}
			other[key] = appendUniqueURL(other[key], url)
		}
	}
	return portrait, other
}

// appendUniqueURL adds a URL once, along with the extension-suffixed variants
// GOG needs.
//
// Some rows carry a bare image id with no extension, which is a template the
// client completes; the two common completions are tried rather than guessing
// one, since a miss simply falls through to the next candidate.
func appendUniqueURL(list []string, url string) []string {
	candidates := []string{url}
	if filepath.Ext(url) == "" {
		candidates = []string{url + ".png", url + ".jpg", url}
	}
	for _, c := range candidates {
		if !slices.Contains(list, c) {
			list = append(list, c)
		}
	}
	return list
}

// snapshotSQLite copies a database and its write-ahead log to a temporary file
// so it can be read without touching the original.
func snapshotSQLite(path string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "gamelib-sqlite-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	dst := filepath.Join(dir, filepath.Base(path))
	if err := copyFileBytes(path, dst); err != nil {
		cleanup()
		return "", func() {}, err
	}
	// Without the sidecars the copy can be missing recent writes, or can look
	// corrupt to the reader; both are silent wrong answers rather than errors.
	for _, suffix := range []string{"-wal", "-shm"} {
		if fileExists(path + suffix) {
			_ = copyFileBytes(path+suffix, dst+suffix)
		}
	}
	return dst, cleanup, nil
}

func copyFileBytes(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o600)
}

// gogProductID turns a Galaxy release key into a plain GOG product id.
//
// Galaxy indexes integrated launchers in the same tables, so the library also
// holds steam_730 and epic_... rows. Those belong to the platforms that own
// them and are dropped here rather than duplicated under GOG.
func gogProductID(releaseKey string) (string, bool) {
	key := strings.TrimSpace(releaseKey)
	rest, ok := strings.CutPrefix(key, "gog_")
	if !ok || strings.TrimSpace(rest) == "" {
		return "", false
	}
	return rest, true
}

// gogTitles maps release key to display title.
//
// Galaxy stores metadata as JSON blobs in GamePieces, one row per field per
// game, so the title has to be pulled out of the blob. The piece type name has
// changed across Galaxy versions, hence matching several.
func gogTitles(db *sql.DB) map[string]string {
	out := map[string]string{}
	rows, err := db.Query(`
		SELECT gp.releaseKey, gp.value
		FROM GamePieces gp
		JOIN GamePieceTypes gpt ON gpt.id = gp.gamePieceTypeId
		WHERE gpt.type IN ('title', 'originalTitle', 'meta', 'originalMeta')`)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		title := strings.TrimSpace(gjson.Get(value, "title").String())
		if title == "" {
			continue
		}
		// 'title' is the localised name and wins; 'meta' rows only fill a gap.
		if _, seen := out[key]; !seen {
			out[key] = title
		}
	}
	return out
}

// gogInstalled maps product id to install path for everything on disk.
func gogInstalled(db *sql.DB) map[string]string {
	out := map[string]string{}
	rows, err := db.Query(`SELECT productId, installationPath FROM InstalledBaseProducts`)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var productID string
		var path sql.NullString
		if err := rows.Scan(&productID, &path); err != nil {
			continue
		}
		productID = strings.TrimSpace(productID)
		if productID == "" {
			continue
		}
		out[productID] = strings.TrimSpace(path.String)
	}
	return out
}

// gogOwnedByUser maps GOG user id to the release keys that user owns.
//
// Galaxy has moved this between tables across versions, so each known source is
// tried and the results are combined. Any one of them being absent on a given
// install is normal, not an error.
func gogOwnedByUser(db *sql.DB) map[string][]string {
	out := map[string]map[string]struct{}{}
	queries := []string{
		`SELECT userId, releaseKey FROM UserReleaseProperties`,
		`SELECT userId, gameReleaseKey FROM ProductPurchaseDates`,
		`SELECT userId, releaseKey FROM UserReleaseTags`,
		`SELECT userId, releaseKey FROM GamePieces WHERE userId IS NOT NULL`,
	}
	for _, q := range queries {
		rows, err := db.Query(q)
		if err != nil {
			continue
		}
		for rows.Next() {
			var userID sql.NullString
			var key sql.NullString
			if err := rows.Scan(&userID, &key); err != nil {
				continue
			}
			uid := strings.TrimSpace(userID.String)
			rk := strings.TrimSpace(key.String)
			if uid == "" || uid == "0" || rk == "" {
				continue
			}
			if out[uid] == nil {
				out[uid] = map[string]struct{}{}
			}
			out[uid][rk] = struct{}{}
		}
		_ = rows.Close()
	}

	res := make(map[string][]string, len(out))
	for uid, keys := range out {
		list := make([]string, 0, len(keys))
		for k := range keys {
			list = append(list, k)
		}
		res[uid] = list
	}
	return res
}

type gogGameTime struct {
	userID     string
	releaseKey string
	minutes    int64
	lastPlayed time.Time
}

// gogGameTimes reads per-account playtime.
func gogGameTimes(db *sql.DB) []gogGameTime {
	rows, err := db.Query(`SELECT userId, releaseKey, minutesInGame, lastPlayedDate FROM GameTimes`)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	var out []gogGameTime
	for rows.Next() {
		var userID, releaseKey sql.NullString
		var minutes, lastPlayed sql.NullInt64
		if err := rows.Scan(&userID, &releaseKey, &minutes, &lastPlayed); err != nil {
			continue
		}
		uid := strings.TrimSpace(userID.String)
		rk := strings.TrimSpace(releaseKey.String)
		if uid == "" || rk == "" {
			continue
		}
		gt := gogGameTime{userID: uid, releaseKey: rk}
		if minutes.Valid && minutes.Int64 > 0 {
			gt.minutes = minutes.Int64
		}
		if lastPlayed.Valid && lastPlayed.Int64 > 0 {
			gt.lastPlayed = time.Unix(lastPlayed.Int64, 0).UTC()
		}
		out = append(out, gt)
	}
	return out
}
