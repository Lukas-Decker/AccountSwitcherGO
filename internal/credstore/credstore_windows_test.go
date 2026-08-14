//go:build windows

package credstore

import (
	"errors"
	"testing"
	"unsafe"
)

// CredWriteW reads the struct by raw pointer and has no version field to catch a
// mismatch, so a layout that drifts from the Windows one corrupts silently.
func TestCredentialStructMatchesWindowsLayout(t *testing.T) {
	if got := unsafe.Sizeof(credentialW{}); got != 80 {
		t.Fatalf("CREDENTIALW is %d bytes, Windows x64 expects 80", got)
	}
}

func TestSecretRoundTrip(t *testing.T) {
	if !Available() {
		t.Skip("no credential store on this machine")
	}
	const key = "credstore-test-delete-me"
	const secret = "RGAPI-00000000-1111-2222-3333-444444444444"
	t.Cleanup(func() { _ = Delete(key) })

	if err := Set(key, secret); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := Get(key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != secret {
		t.Fatalf("got %q, want %q", got, secret)
	}

	if err := Delete(key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := Get(key); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: err = %v, want ErrNotFound", err)
	}
	// Callers delete without checking first, so a missing key is not a failure.
	if err := Delete(key); err != nil {
		t.Errorf("deleting a missing key: %v", err)
	}
}
