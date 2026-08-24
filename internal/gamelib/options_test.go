package gamelib

import (
	"context"
	"testing"
)

// Artwork and online library listings are separate permissions, and they were
// once the same flag. That conflation meant the remote half of the art chain
// never ran unless the user found an opt-in labelled for something else, and
// never ran at all on the platforms with no local artwork to fall back on.

func withResolver(t *testing.T, key string, fn func(opts Options)) {
	t.Helper()
	prev, had := ResolverFor(key)
	Register(ResolverFunc{Key: key, Fn: func(_ context.Context, opts Options) (Result, error) {
		fn(opts)
		return Result{}, nil
	}})
	t.Cleanup(func() {
		if had {
			Register(prev)
			return
		}
		registryMu.Lock()
		delete(registered, key)
		registryMu.Unlock()
	})
}

// Fetching art must not require the account-data opt-in.
func TestResolvePlatform_ArtworkIsIndependentOfLibraryEnrichment(t *testing.T) {
	var got Options
	withResolver(t, "TestPlatform", func(o Options) { got = o })

	if _, err := ResolvePlatform(context.Background(), "TestPlatform", false, true); err != nil {
		t.Fatal(err)
	}
	if got.AllowNetwork {
		t.Error("library enrichment was enabled when it was not asked for")
	}
	if !got.AllowArtwork {
		t.Error("artwork was disabled even though it was allowed")
	}
}

// And the reverse: asking for library data must not silently force artwork on
// when the caller said no, which is what offline mode relies on.
func TestResolvePlatform_ArtworkCanBeRefusedIndependently(t *testing.T) {
	var got Options
	withResolver(t, "TestPlatform", func(o Options) { got = o })

	if _, err := ResolvePlatform(context.Background(), "TestPlatform", true, false); err != nil {
		t.Fatal(err)
	}
	if !got.AllowNetwork {
		t.Error("library enrichment was refused when it was asked for")
	}
	if got.AllowArtwork {
		t.Error("artwork ran despite being refused")
	}
}

// The per-platform account context must survive both flags being set.
func TestResolvePlatform_KeepsAccountContext(t *testing.T) {
	prev := OptionsForPlatform
	OptionsForPlatform = func(string) Options {
		return Options{KnownAccounts: map[string]string{"a": "A"}, ActiveAccountID: "a"}
	}
	t.Cleanup(func() { OptionsForPlatform = prev })

	var got Options
	withResolver(t, "TestPlatform", func(o Options) { got = o })

	if _, err := ResolvePlatform(context.Background(), "TestPlatform", true, true); err != nil {
		t.Fatal(err)
	}
	if len(got.KnownAccounts) != 1 || got.ActiveAccountID != "a" {
		t.Errorf("account context lost: %+v", got)
	}
	if !got.AllowNetwork || !got.AllowArtwork {
		t.Errorf("flags lost: %+v", got)
	}
}
