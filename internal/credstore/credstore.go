// Package credstore keeps small secrets in the operating system's own
// credential store.
//
// The secrets it holds, an API key today, must not sit in the settings file:
// that file is plain JSON and gets pasted into bug reports. The OS store keeps
// them out of anything the user is likely to copy, and hands responsibility for
// the encryption to the platform.
//
// Written here rather than taken from a third-party module. The surface needed
// is three calls against one key, which is far less code than the dependency
// would add, and this way the implementation is ours to audit and change.
//
// Every backend is best-effort by design: a machine with no usable store returns
// ErrUnsupported, and callers are expected to degrade to asking the user again
// rather than quietly writing the secret somewhere weaker.
package credstore

import "errors"

var (
	// ErrNotFound reports that the key has no stored value.
	ErrNotFound = errors.New("credstore: no stored secret")
	// ErrUnsupported reports that this machine has no usable credential store.
	// Not a failure to store: a failure to have anywhere to store.
	ErrUnsupported = errors.New("credstore: no credential store available on this platform")
)

// Service namespaces the entries this application owns inside a shared store.
const Service = "Account Switcher"

// Get returns the secret stored under key, or ErrNotFound.
func Get(key string) (string, error) { return get(key) }

// Set stores secret under key, replacing any previous value.
func Set(key, secret string) error { return set(key, secret) }

// Delete removes the secret stored under key. Removing a key that is not there
// is not an error.
func Delete(key string) error { return del(key) }

// Available reports whether this machine has a working credential store, so the
// UI can say so before the user types a secret into it.
func Available() bool { return available() }
