package gameart

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// withIGDB installs credentials for the duration of one test.
func withIGDB(t *testing.T, id, secret string) {
	t.Helper()
	SetIGDBCredentials(id, secret)
	t.Cleanup(func() { SetIGDBCredentials("", "") })
}

// igdbStub serves the token endpoint and records the Apicalypse queries it was
// asked, so a test can assert on what was actually sent.
type igdbStub struct {
	srv     *httptest.Server
	queries []string
	tokens  int
}

func newIGDBStub(t *testing.T, handle func(path, query string) string) *igdbStub {
	t.Helper()
	st := &igdbStub{}
	st.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/oauth2/token") {
			st.tokens++
			_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":5000000}`))
			return
		}
		if r.Header.Get("Client-ID") == "" || r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("missing auth headers on %s", r.URL.Path)
		}
		body, _ := readAllString(r)
		st.queries = append(st.queries, body)
		_, _ = w.Write([]byte(handle(r.URL.Path, body)))
	}))
	t.Cleanup(st.srv.Close)
	t.Cleanup(igdbEndpointsForTest(st.srv.URL, st.srv.URL+"/oauth2/token"))
	return st
}

func readAllString(r *http.Request) (string, error) {
	defer func() { _ = r.Body.Close() }()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			return sb.String(), nil
		}
	}
}

// Nothing may be sent anywhere until both halves of the credential exist.
func TestIGDB_SilentWithoutCredentials(t *testing.T) {
	SetIGDBCredentials("", "")

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	defer igdbEndpointsForTest(srv.URL, srv.URL+"/oauth2/token")()

	if IGDBEnabled() {
		t.Fatal("enabled with no credentials")
	}
	if got := IGDBCandidates(context.Background(), srv.Client(), "730", "Counter-Strike 2"); got != nil {
		t.Errorf("returned %v with no credentials", got)
	}
	// An id without a secret is not a credential either.
	SetIGDBCredentials("id-only", "")
	t.Cleanup(func() { SetIGDBCredentials("", "") })
	if IGDBEnabled() {
		t.Error("enabled with only a client id")
	}
	if hits != 0 {
		t.Errorf("made %d requests with no credentials", hits)
	}
}

// A Steam app id maps through the external games table, which is exact, rather
// than through a title search that several games could match.
func TestIGDB_SteamAppIDUsesTheExternalGamesTable(t *testing.T) {
	withIGDB(t, "cid", "secret")

	st := newIGDBStub(t, func(path, query string) string {
		switch {
		case strings.HasSuffix(path, "/external_games"):
			return `[{"game":1234}]`
		case strings.HasSuffix(path, "/games"):
			return `[{"cover":{"image_id":"co1r76"},"artworks":[{"image_id":"ar9k"}]}]`
		}
		return `[]`
	})

	got := IGDBCandidates(context.Background(), st.srv.Client(), "730", "Counter-Strike 2")
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want a cover and an artwork: %+v", len(got), got)
	}
	if got[0].Tier != TierPortrait || !strings.Contains(got[0].URL, "t_cover_big/co1r76.jpg") {
		t.Errorf("first candidate = %+v, want the cover as portrait", got[0])
	}
	if got[1].Tier != TierWide || !strings.Contains(got[1].URL, "ar9k") {
		t.Errorf("second candidate = %+v, want the artwork as wide", got[1])
	}

	joined := strings.Join(st.queries, "\n")
	if !strings.Contains(joined, `uid = "730"`) || !strings.Contains(joined, "category = 1") {
		t.Errorf("did not address the Steam app id exactly:\n%s", joined)
	}
	if strings.Contains(joined, `search "`) {
		t.Errorf("fell back to a title search despite having an app id:\n%s", joined)
	}
}

// Everything that is not Steam has to be found by title.
func TestIGDB_FallsBackToSearchByName(t *testing.T) {
	withIGDB(t, "cid", "secret")

	st := newIGDBStub(t, func(path, query string) string {
		if strings.HasSuffix(path, "/external_games") {
			return `[]`
		}
		if strings.Contains(query, `search "`) {
			return `[{"id":99}]`
		}
		return `[{"cover":{"image_id":"cov99"}}]`
	})

	got := IGDBCandidates(context.Background(), st.srv.Client(), "", "Fortnite")
	if len(got) != 1 || got[0].Tier != TierPortrait {
		t.Fatalf("got %+v, want one portrait", got)
	}
	if !strings.Contains(strings.Join(st.queries, "\n"), `search "Fortnite"`) {
		t.Errorf("did not search by name: %v", st.queries)
	}
}

// The token is minted once and reused; a pass over a library must not
// re-authenticate per game.
func TestIGDB_TokenIsReused(t *testing.T) {
	withIGDB(t, "cid", "secret")

	st := newIGDBStub(t, func(path, query string) string {
		if strings.HasSuffix(path, "/external_games") {
			return `[{"game":1}]`
		}
		return `[{"cover":{"image_id":"c"}}]`
	})

	for i := 0; i < 3; i++ {
		IGDBCandidates(context.Background(), st.srv.Client(), "730", "")
	}
	if st.tokens != 1 {
		t.Errorf("minted %d tokens, want 1", st.tokens)
	}
}

// Apicalypse has no parameter binding, so a title containing a quote would
// otherwise close the string literal and change the query.
func TestIGDBEscape(t *testing.T) {
	t.Parallel()

	got := igdbEscape(`Marvel's "Game"; drop`)
	if strings.Contains(got, `";`) {
		t.Errorf("escape left a statement terminator: %q", got)
	}
	if strings.Count(got, `\"`) != 2 {
		t.Errorf("quotes not escaped: %q", got)
	}
	if strings.ContainsAny(got, ";\n\r") {
		t.Errorf("escape left a query separator: %q", got)
	}
	// It must still be embeddable in a JSON string, which is how it reaches
	// the wire.
	if _, err := json.Marshal(got); err != nil {
		t.Errorf("escaped value is not encodable: %v", err)
	}
}

// A refused request drops the cached token so the next call mints a fresh one
// rather than failing for as long as the stale one is held.
func TestIGDB_RefusalClearsTheCachedToken(t *testing.T) {
	withIGDB(t, "cid", "secret")

	st := newIGDBStub(t, func(path, query string) string { return `[]` })
	// Prime a token.
	if _, ok := igdbAccessToken(context.Background(), st.srv.Client()); !ok {
		t.Fatal("could not mint a token")
	}

	// Now refuse everything.
	defer igdbEndpointsForTest(st.srv.URL, st.srv.URL+"/oauth2/token")()
	refuse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/oauth2/token") {
			_, _ = w.Write([]byte(`{"access_token":"tok2","expires_in":5000000}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer refuse.Close()
	defer igdbEndpointsForTest(refuse.URL, refuse.URL+"/oauth2/token")()

	IGDBCandidates(context.Background(), refuse.Client(), "730", "")

	igdbCredMu.RLock()
	held := igdbToken
	igdbCredMu.RUnlock()
	if held != "" {
		t.Errorf("kept token %q after a refusal", held)
	}
}

// Both archives are asked, and the best shape across the two wins.
func TestArchiveCandidates_CombinesBothArchives(t *testing.T) {
	withKey(t, "sgdb-key")
	withIGDB(t, "cid", "secret")

	// SteamGridDB has only a wide hero for this game.
	sgdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/heroes/") {
			_, _ = w.Write([]byte(`{"success":true,"data":[{"url":"https://cdn.example/hero.png"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
	}))
	defer sgdb.Close()
	defer sgdbBaseForTest(sgdb.URL)()

	// IGDB has the cover.
	st := newIGDBStub(t, func(path, query string) string {
		if strings.HasSuffix(path, "/external_games") {
			return `[{"game":7}]`
		}
		return `[{"cover":{"image_id":"cov7"}}]`
	})

	got := ArchiveCandidates(context.Background(), st.srv.Client(), ArchiveRef{SteamAppID: "730", Name: "CS2"})
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want one per shape: %+v", len(got), got)
	}
	if got[0].Tier != TierPortrait || !strings.Contains(got[0].URL, "cov7") {
		t.Errorf("best candidate = %+v, want IGDB's cover", got[0])
	}
	if got[1].Tier != TierWide || !strings.Contains(got[1].URL, "hero.png") {
		t.Errorf("second candidate = %+v, want SteamGridDB's hero", got[1])
	}
}

// With neither archive configured the composed layer must stay silent.
func TestArchiveCandidates_SilentWithNothingConfigured(t *testing.T) {
	SetSteamGridDBKey("")
	SetIGDBCredentials("", "")

	if AnyArchiveEnabled() {
		t.Fatal("reported an archive as enabled with no credentials")
	}
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()

	if got := ArchiveCandidates(context.Background(), srv.Client(), ArchiveRef{SteamAppID: "730", Name: "CS2"}); got != nil {
		t.Errorf("got %+v, want nothing", got)
	}
	if hits != 0 {
		t.Errorf("made %d requests with nothing configured", hits)
	}
}

// A reference with neither an id nor a name identifies nothing.
func TestArchiveCandidates_RequiresSomethingToLookUp(t *testing.T) {
	withKey(t, "k")
	if got := ArchiveCandidates(context.Background(), http.DefaultClient, ArchiveRef{}); got != nil {
		t.Errorf("got %+v, want nothing", got)
	}
}
