package riot

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCurrentWalletAgainstRunningClient checks the wallet call against a real
// League Client, which is the only thing that can confirm the currency names and
// the query string the client actually accepts.
//
// Skipped unless the client is running, so it is a no-op on a build machine and
// a real check on a developer's.
func TestCurrentWalletAgainstRunningClient(t *testing.T) {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	manifest := filepath.Join(programData, "Riot Games", "RiotClientInstalls.json")

	client, err := ConnectLCU(manifest)
	if err != nil {
		if errors.Is(err, ErrLCUNotRunning) {
			t.Skip("League Client is not running")
		}
		t.Skipf("no usable League Client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	wallet, err := client.CurrentWallet(ctx)
	if err != nil {
		t.Fatalf("CurrentWallet against the running client: %v", err)
	}
	// The amounts belong to whoever is signed in, so only their shape is asserted.
	if wallet.BlueEssence < 0 || wallet.RiotPoints < 0 {
		t.Errorf("negative balance: %+v", wallet)
	}
	t.Logf("wallet read from the running client: BE=%d RP=%d", wallet.BlueEssence, wallet.RiotPoints)
}
