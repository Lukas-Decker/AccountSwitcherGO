//go:build !windows && !darwin

package credstore

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// The Linux and BSD backend drives secret-tool, the command line front end to
// the freedesktop Secret Service that GNOME Keyring and KWallet both implement.
// Talking D-Bus directly would mean carrying a D-Bus client for three calls.
//
// A headless machine typically has no Secret Service at all, which is why the
// absence of secret-tool reports ErrUnsupported rather than an error: there is
// nowhere to put a secret, and the caller should say so instead of pretending.

func secretToolPath() (string, error) {
	return exec.LookPath("secret-tool")
}

func get(key string) (string, error) {
	bin, err := secretToolPath()
	if err != nil {
		return "", ErrUnsupported
	}
	out, err := exec.Command(bin, "lookup", "service", Service, "account", key).Output()
	if err != nil {
		var ee *exec.ExitError
		// A missing entry exits non-zero with nothing on stdout.
		if errors.As(err, &ee) && len(bytes.TrimSpace(out)) == 0 {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("credstore: secret-tool lookup: %w", err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return "", ErrNotFound
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func set(key, secret string) error {
	bin, err := secretToolPath()
	if err != nil {
		return ErrUnsupported
	}
	// secret-tool store reads the secret from stdin, keeping it out of argv.
	cmd := exec.Command(bin, "store", "--label", Service+": "+key, "service", Service, "account", key)
	cmd.Stdin = strings.NewReader(secret)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("credstore: secret-tool store: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func del(key string) error {
	bin, err := secretToolPath()
	if err != nil {
		return ErrUnsupported
	}
	// Clearing an entry that was never stored is reported as success, matching
	// the other backends.
	if err := exec.Command(bin, "clear", "service", Service, "account", key).Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil
		}
		return fmt.Errorf("credstore: secret-tool clear: %w", err)
	}
	return nil
}

func available() bool {
	_, err := secretToolPath()
	return err == nil
}
