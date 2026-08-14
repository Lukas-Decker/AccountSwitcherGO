//go:build windows

package credstore

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modadvapi32     = windows.NewLazySystemDLL("advapi32.dll")
	procCredReadW   = modadvapi32.NewProc("CredReadW")
	procCredWriteW  = modadvapi32.NewProc("CredWriteW")
	procCredDeleteW = modadvapi32.NewProc("CredDeleteW")
	procCredFree    = modadvapi32.NewProc("CredFree")
)

// CREDENTIALW. The blob is encrypted by Windows against the logged-on user, so
// another account on the same machine cannot read it back.
type credentialW struct {
	flags              uint32
	typ                uint32
	targetName         *uint16
	comment            *uint16
	lastWritten        windows.Filetime
	credentialBlobSize uint32
	credentialBlob     *byte
	persist            uint32
	attributeCount     uint32
	attributes         uintptr
	targetAlias        *uint16
	userName           *uint16
}

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
	// CRED_MAX_CREDENTIAL_BLOB_SIZE. Comfortably above any API key, but a caller
	// passing something large should be told rather than have it truncated.
	credMaxBlobSize = 5 * 512
)

// targetName is the entry name shown in Credential Manager.
func targetName(key string) string { return Service + ":" + key }

func get(key string) (string, error) {
	target, err := windows.UTF16PtrFromString(targetName(key))
	if err != nil {
		return "", err
	}
	var cred *credentialW
	r, _, callErr := procCredReadW.Call(
		uintptr(unsafe.Pointer(target)),
		credTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&cred)),
	)
	if r == 0 {
		if callErr == syscall.Errno(windows.ERROR_NOT_FOUND) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("credstore: CredReadW: %w", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(cred)))

	if cred.credentialBlobSize == 0 || cred.credentialBlob == nil {
		return "", ErrNotFound
	}
	blob := unsafe.Slice(cred.credentialBlob, cred.credentialBlobSize)
	return string(blob), nil
}

func set(key, secret string) error {
	if len(secret) > credMaxBlobSize {
		return fmt.Errorf("credstore: secret is %d bytes, limit is %d", len(secret), credMaxBlobSize)
	}
	target, err := windows.UTF16PtrFromString(targetName(key))
	if err != nil {
		return err
	}
	// UserName is shown in Credential Manager and cannot be empty for a generic
	// credential, so it names the app rather than anything about the user.
	user, err := windows.UTF16PtrFromString(Service)
	if err != nil {
		return err
	}

	blob := []byte(secret)
	cred := credentialW{
		typ:                credTypeGeneric,
		targetName:         target,
		persist:            credPersistLocalMachine,
		credentialBlobSize: uint32(len(blob)),
		userName:           user,
	}
	if len(blob) > 0 {
		cred.credentialBlob = &blob[0]
	}

	r, _, callErr := procCredWriteW.Call(uintptr(unsafe.Pointer(&cred)), 0)
	// blob must outlive the call: without this the garbage collector is free to
	// move or reclaim it while Windows is still reading through the raw pointer.
	runtime.KeepAlive(blob)
	if r == 0 {
		return fmt.Errorf("credstore: CredWriteW: %w", callErr)
	}
	return nil
}

func del(key string) error {
	target, err := windows.UTF16PtrFromString(targetName(key))
	if err != nil {
		return err
	}
	r, _, callErr := procCredDeleteW.Call(uintptr(unsafe.Pointer(target)), credTypeGeneric, 0)
	if r == 0 {
		if callErr == syscall.Errno(windows.ERROR_NOT_FOUND) {
			return nil
		}
		return fmt.Errorf("credstore: CredDeleteW: %w", callErr)
	}
	return nil
}

// available reports whether advapi32 exposes the credential calls. Windows has
// had them since XP, so this only fails somewhere very unusual.
func available() bool {
	return procCredReadW.Find() == nil && procCredWriteW.Find() == nil
}
