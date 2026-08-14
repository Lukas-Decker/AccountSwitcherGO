//go:build darwin

package credstore

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// The macOS backend drives the security(1) tool rather than linking the Keychain
// framework, which would drag cgo into a build that otherwise does not need it.
// security ships with the OS, so there is nothing to install.

func securityPath() (string, error) {
	return exec.LookPath("security")
}

func get(key string) (string, error) {
	bin, err := securityPath()
	if err != nil {
		return "", ErrUnsupported
	}
	// -w prints only the password, so nothing has to be parsed out of a dump.
	out, err := exec.Command(bin, "find-generic-password", "-s", Service, "-a", key, "-w").Output()
	if err != nil {
		var ee *exec.ExitError
		// security exits 44 for "not found"; anything else is a real failure.
		if errors.As(err, &ee) && ee.ExitCode() == 44 {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("credstore: security find-generic-password: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func set(key, secret string) error {
	bin, err := securityPath()
	if err != nil {
		return ErrUnsupported
	}
	// -U updates in place instead of failing when the entry already exists.
	// The secret goes in on stdin via -w so it never appears in the process list,
	// where any other user on the machine could read it.
	cmd := exec.Command(bin, "add-generic-password", "-s", Service, "-a", key, "-U", "-w")
	cmd.Stdin = strings.NewReader(secret)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("credstore: security add-generic-password: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func del(key string) error {
	bin, err := securityPath()
	if err != nil {
		return ErrUnsupported
	}
	err = exec.Command(bin, "delete-generic-password", "-s", Service, "-a", key).Run()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 44 {
			return nil
		}
		return fmt.Errorf("credstore: security delete-generic-password: %w", err)
	}
	return nil
}

func available() bool {
	_, err := securityPath()
	return err == nil
}
