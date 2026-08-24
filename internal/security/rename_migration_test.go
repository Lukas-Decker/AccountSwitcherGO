package security

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"account-switcher/internal/paths"
)

// The rename changed the associated data bound into every ciphertext and the
// extension every blob is stored under. Both are load-bearing: get either wrong
// and a user's saved accounts become permanently unreadable. These tests build
// a vault the way the previous build would have, then check the current one
// still opens it.

const (
	migPlatform = "Example"
	migUniqueID = "uid-1"
	migName     = "Account One"
	migPassword = "secret"
)

var migPayload = []byte{0, 1, 2, 3, 255}

// seedEncryptedAccount creates one encrypted account through the normal path.
func seedEncryptedAccount(t *testing.T) string {
	t.Helper()
	root := resetSecurityTest(t)

	accountDir := filepath.Join(root, "LoginCache", migPlatform, paths.SanitizePathSegment(migName))
	writeTestFile(t, filepath.Join(accountDir, "data.bin"), migPayload)
	writeTestJSON(t, filepath.Join(root, "LoginCache", migPlatform, "ids.json"), map[string]any{
		"ids": map[string]string{migUniqueID: migName},
	})

	if err := SetAppPassword(migPassword); err != nil {
		t.Fatal(err)
	}
	if err := EnableSavedAccountEncryption(migPassword); err != nil {
		t.Fatal(err)
	}
	return root
}

// rewriteBlobAsLegacy re-seals the account's blob with the pre-rename
// associated data and stores it under the pre-rename extension, which is
// exactly what a vault written by the previous build looks like.
func rewriteBlobAsLegacy(t *testing.T) {
	t.Helper()

	key, err := defaultManager.unlockedMasterKey()
	if err != nil {
		t.Fatal(err)
	}
	current, err := accountBlobPath(migPlatform, migUniqueID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	var blob encryptedAccountBlob
	if err := json.Unmarshal(raw, &blob); err != nil {
		t.Fatal(err)
	}
	plain, err := decryptAccountBlobFile(key, current, migPlatform, migUniqueID)
	if err != nil {
		t.Fatal(err)
	}
	nonce, ciphertext, err := sealWithKey(key, plain, legacyAccountBlobAAD(migPlatform, migUniqueID))
	if err != nil {
		t.Fatal(err)
	}
	blob.Nonce = encode(nonce)
	blob.Ciphertext = encode(ciphertext)
	out, err := json.MarshalIndent(blob, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	legacy, err := legacyAccountBlobPath(migPlatform, migUniqueID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, out, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(current); err != nil {
		t.Fatal(err)
	}
}

// A vault written before the rename must still restore, or the update that
// shipped the rename would have destroyed every saved account.
func TestLegacyVaultStillRestores(t *testing.T) {
	seedEncryptedAccount(t)
	rewriteBlobAsLegacy(t)

	if !AccountBlobValid(migPlatform, migUniqueID) {
		t.Fatal("a blob written before the rename no longer validates")
	}

	dir, cleanup, err := AccountRestoreDir(migPlatform, migUniqueID, migName, "")
	if err != nil {
		t.Fatalf("restore from a pre-rename vault failed: %v", err)
	}
	defer cleanup()

	got, err := os.ReadFile(filepath.Join(dir, "data.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, migPayload) {
		t.Fatalf("restored %v, want %v", got, migPayload)
	}
}

// Reading a legacy blob upgrades it in passing, so the fallback is needed once
// per account rather than forever.
func TestLegacyVaultIsUpgradedOnRead(t *testing.T) {
	seedEncryptedAccount(t)
	rewriteBlobAsLegacy(t)

	current, err := accountBlobPath(migPlatform, migUniqueID)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := legacyAccountBlobPath(migPlatform, migUniqueID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(current); !os.IsNotExist(err) {
		t.Fatalf("test setup left a current-format blob behind: %v", err)
	}

	dir, cleanup, err := AccountRestoreDir(migPlatform, migUniqueID, migName, "")
	if err != nil {
		t.Fatal(err)
	}
	cleanup()
	_ = dir

	if _, err := os.Stat(current); err != nil {
		t.Fatalf("blob was not upgraded to the current name: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("pre-rename blob was left behind: %v", err)
	}
	// And it must still open now that it has been rewritten.
	if !AccountBlobValid(migPlatform, migUniqueID) {
		t.Fatal("upgraded blob does not validate")
	}
}

// The security file's own verifier and wrapped key carry the same associated
// data, so a password set before the rename has to keep working.
func TestLegacySecurityFileStillUnlocks(t *testing.T) {
	resetSecurityTest(t)

	if err := SetAppPassword(migPassword); err != nil {
		t.Fatal(err)
	}
	sf, ok, err := loadSecurityFile()
	if err != nil || !ok {
		t.Fatalf("load security file: ok=%v err=%v", ok, err)
	}

	// Re-seal the verifier and wrapped key the way the previous build did.
	salt, err := decode(sf.Salt)
	if err != nil {
		t.Fatal(err)
	}
	derived := deriveKey(migPassword, salt, sf.KDF)
	master, err := unlockWithPassword(migPassword)
	if err != nil {
		t.Fatal(err)
	}
	vNonce, vCipher, err := sealWithKey(derived, []byte("tcno-security-ok"), []byte(legacySecurityVerifierAAD))
	if err != nil {
		t.Fatal(err)
	}
	wNonce, wrapped, err := sealWithKey(derived, master, []byte(legacyWrappedKeyAAD))
	if err != nil {
		t.Fatal(err)
	}
	sf.VerifierNonce = encode(vNonce)
	sf.VerifierCiphertext = encode(vCipher)
	sf.WrappedVaultKeyNonce = encode(wNonce)
	sf.WrappedVaultKeyCiphertext = encode(wrapped)
	if err := saveSecurityFile(sf); err != nil {
		t.Fatal(err)
	}

	got, err := unlockWithPassword(migPassword)
	if err != nil {
		t.Fatalf("password set before the rename no longer unlocks: %v", err)
	}
	if !bytes.Equal(got, master) {
		t.Fatal("unlocked a different master key than the one that was wrapped")
	}

	// The wrong password must still be rejected, rather than the legacy
	// fallback turning into a way past the check.
	if _, err := unlockWithPassword("not-the-password"); err == nil {
		t.Fatal("wrong password accepted")
	}
}

// Once unlocked, the security file is rewritten under the current associated
// data so the legacy path stops being exercised.
func TestLegacySecurityFileIsResealed(t *testing.T) {
	resetSecurityTest(t)

	if err := SetAppPassword(migPassword); err != nil {
		t.Fatal(err)
	}
	sf, ok, err := loadSecurityFile()
	if err != nil || !ok {
		t.Fatalf("load security file: ok=%v err=%v", ok, err)
	}
	salt, err := decode(sf.Salt)
	if err != nil {
		t.Fatal(err)
	}
	derived := deriveKey(migPassword, salt, sf.KDF)
	master, err := unlockWithPassword(migPassword)
	if err != nil {
		t.Fatal(err)
	}
	vNonce, vCipher, err := sealWithKey(derived, []byte("tcno-security-ok"), []byte(legacySecurityVerifierAAD))
	if err != nil {
		t.Fatal(err)
	}
	wNonce, wrapped, err := sealWithKey(derived, master, []byte(legacyWrappedKeyAAD))
	if err != nil {
		t.Fatal(err)
	}
	sf.VerifierNonce = encode(vNonce)
	sf.VerifierCiphertext = encode(vCipher)
	sf.WrappedVaultKeyNonce = encode(wNonce)
	sf.WrappedVaultKeyCiphertext = encode(wrapped)
	if err := saveSecurityFile(sf); err != nil {
		t.Fatal(err)
	}

	if _, err := unlockWithPassword(migPassword); err != nil {
		t.Fatal(err)
	}

	after, ok, err := loadSecurityFile()
	if err != nil || !ok {
		t.Fatalf("reload security file: ok=%v err=%v", ok, err)
	}
	nonce, err := decode(after.VerifierNonce)
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := decode(after.VerifierCiphertext)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := openWithKey(derived, nonce, cipher, []byte(securityVerifierAAD)); err != nil {
		t.Fatalf("security file was not re-sealed under the current AAD: %v", err)
	}
}
