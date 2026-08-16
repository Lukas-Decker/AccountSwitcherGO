package riot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestParseLockfile(t *testing.T) {
	creds, err := ParseLockfile([]byte("Riot Client:84172:50428:abcdefghijklmnopqrstuv:https\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if creds.Port != 50428 || creds.Protocol != "https" {
		t.Errorf("parsed %+v", creds)
	}
	if got := creds.BaseURL(); got != "https://127.0.0.1:50428" {
		t.Errorf("base URL = %q", got)
	}
	// The username half is fixed, so only the password varies.
	if !strings.HasPrefix(creds.authHeader(), "Basic ") {
		t.Errorf("auth header = %q", creds.authHeader())
	}

	for _, bad := range []string{
		"",
		"too:few:fields",
		"Riot Client:1:notaport:pw:https",
		"Riot Client:1:50428::https",
	} {
		if _, err := ParseLockfile([]byte(bad)); err == nil {
			t.Errorf("ParseLockfile(%q) was accepted", bad)
		}
	}
}

// TLS verification is disabled for this client because the League Client serves
// a self-signed certificate. That is only safe while it cannot reach anything
// but loopback, so the restriction is the thing worth testing: without it, an
// unverified connection could be pointed anywhere.
func TestLCUClientRefusesNonLoopback(t *testing.T) {
	c := NewLCUClient(LCUCredentials{Port: 50428, Password: "x", Protocol: "https"})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.http.Do(req); err == nil {
		t.Fatal("a non-loopback request was allowed")
	} else if !strings.Contains(err.Error(), "non-loopback") {
		t.Errorf("refused for the wrong reason: %v", err)
	}
}

// The wallet is asked for by naming the currencies. A bare request answers 400,
// and an unknown name answers with the whole wallet rather than an error, so the
// names in the query string are what make the reply the two figures wanted.
func TestWalletPathNamesBothCurrencies(t *testing.T) {
	if !strings.HasPrefix(walletPath, "/lol-inventory/v1/wallet?currencyTypes=") {
		t.Fatalf("wallet path = %q", walletPath)
	}
	decoded, err := url.QueryUnescape(strings.SplitN(walletPath, "=", 2)[1])
	if err != nil {
		t.Fatalf("unescape: %v", err)
	}
	var names []string
	if err := json.Unmarshal([]byte(decoded), &names); err != nil {
		t.Fatalf("currencyTypes is not a JSON array: %v (%q)", err, decoded)
	}
	want := map[string]bool{currencyBlueEssence: false, currencyRiotPoints: false}
	for _, n := range names {
		if _, ok := want[n]; ok {
			want[n] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("%s was not asked for: %q", name, decoded)
		}
	}
}

func TestCurrentWalletReadsBothCurrencies(t *testing.T) {
	// The client answers with capitals on one key and not the other, and includes
	// currencies nobody asked about when it feels like it.
	body := `{"RP":22670,"lol_blue_essence":27220,"lol_orange_essence":500}`
	c, done := lcuStub(t, body, http.StatusOK)
	defer done()

	w, err := c.CurrentWallet(context.Background())
	if err != nil {
		t.Fatalf("CurrentWallet: %v", err)
	}
	if w.BlueEssence != 27220 || w.RiotPoints != 22670 {
		t.Errorf("wallet = %+v, want 27220 BE and 22670 RP", w)
	}
}

func TestCurrentWalletRejectsAWalletWithNeitherCurrency(t *testing.T) {
	// A rename must not read as a balance of zero.
	c, done := lcuStub(t, `{"lol_orange_essence":500}`, http.StatusOK)
	defer done()

	if w, err := c.CurrentWallet(context.Background()); err == nil {
		t.Errorf("a wallet with neither currency was accepted as %+v", w)
	}
}

// lcuStub serves one canned reply over loopback, which is the only address the
// LCU client will talk to.
func lcuStub(t *testing.T, body string, status int) (*LCUClient, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	c := NewLCUClient(LCUCredentials{Port: port, Password: "stub", Protocol: "http"})
	return c, srv.Close
}

func TestFindLockfileReportsNotRunning(t *testing.T) {
	// No manifest and no directories: the client cannot be running, and that is a
	// normal state rather than a failure.
	if _, err := FindLockfile(t.TempDir()+"/absent.json", t.TempDir()); !errors.Is(err, ErrLCUNotRunning) {
		t.Errorf("err = %v, want ErrLCUNotRunning", err)
	}
}
