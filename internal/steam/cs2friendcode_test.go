package steam

import "testing"

// Rabscuttle's account, the vector both independent reverse-engineered
// implementations pin themselves to (emily33901/js-csfriendcode,
// not-wlan/csgo-friendcode). Nothing about the encoding is derivable, so a
// wrong shuffle only shows up against a known pair.
func TestCS2FriendCode(t *testing.T) {
	const (
		id64 = "76561197960287930"
		acc  = uint32(22202)
		want = "SUCVS-FADA"
	)

	if got := CS2FriendCode(acc); got != want {
		t.Errorf("CS2FriendCode(%d) = %q, want %q", acc, got, want)
	}

	f, err := FormatsFromID64(id64)
	if err != nil {
		t.Fatalf("FormatsFromID64(%s): %v", id64, err)
	}
	if f.CS2FriendCode != want {
		t.Errorf("CS2FriendCode via formats = %q, want %q", f.CS2FriendCode, want)
	}
	// The Steam friend code Steam's own "Add a Friend" page shows is the
	// account ID, so it must stay in step with SteamID32.
	if f.FriendCode != f.ID32 {
		t.Errorf("FriendCode %q != ID32 %q", f.FriendCode, f.ID32)
	}
}
