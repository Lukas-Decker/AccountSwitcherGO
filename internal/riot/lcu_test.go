package riot

import (
	"context"
	"errors"
	"net/http"
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

func TestFindLockfileReportsNotRunning(t *testing.T) {
	// No manifest and no directories: the client cannot be running, and that is a
	// normal state rather than a failure.
	if _, err := FindLockfile(t.TempDir()+"/absent.json", t.TempDir()); !errors.Is(err, ErrLCUNotRunning) {
		t.Errorf("err = %v, want ErrLCUNotRunning", err)
	}
}
