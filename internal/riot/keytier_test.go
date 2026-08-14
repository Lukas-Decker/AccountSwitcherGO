package riot

import "testing"

// Riot returns the rate-limit buckets in no particular order. A real
// development key was seen answering "100:120,20:1", which an exact string
// match reads as an unfamiliar quota and so as a more privileged key: the
// switcher then polls a key that expires daily and has almost no allowance.
func TestClassifyAppRateLimitIgnoresBucketOrder(t *testing.T) {
	for _, header := range []string{
		"20:1,100:120",
		"100:120,20:1",
		"100:120, 20:1",
		"020:1,100:120",
	} {
		if got := classifyAppRateLimit(header); got != TierDevelopment {
			t.Errorf("classifyAppRateLimit(%q) = %q, want %q", header, got, TierDevelopment)
		}
	}

	// A larger allowance was granted deliberately and may be polled.
	for _, header := range []string{
		"500:10,30000:600",
		"20:1,100:120,500:600",
	} {
		if got := classifyAppRateLimit(header); got != TierElevated {
			t.Errorf("classifyAppRateLimit(%q) = %q, want %q", header, got, TierElevated)
		}
	}

	if got := classifyAppRateLimit(""); got != TierUnknown {
		t.Errorf("empty header = %q, want %q", got, TierUnknown)
	}
	if got := classifyAppRateLimit("nonsense"); got != TierUnknown {
		t.Errorf("unparseable header = %q, want %q", got, TierUnknown)
	}
}

func TestOnlyElevatedKeysArePolled(t *testing.T) {
	if TierDevelopment.AllowsLiveRefresh() {
		t.Error("a development key must not be polled: it expires daily and its quota is tiny")
	}
	if TierUnknown.AllowsLiveRefresh() {
		t.Error("an unclassified key must not be polled")
	}
	if !TierElevated.AllowsLiveRefresh() {
		t.Error("an elevated key should allow live refresh")
	}
}
