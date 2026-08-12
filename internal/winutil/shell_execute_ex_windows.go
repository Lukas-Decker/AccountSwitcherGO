//go:build windows

package winutil

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SHELLEXECUTEINFOW, laid out for the Ex form of ShellExecute.
//
// The plain ShellExecuteW cannot answer the only question that matters here:
// whether a process actually started. It returns as soon as the request is
// queued, which for the runas verb is before the UAC prompt has even been shown,
// let alone answered. Ex with SEE_MASK_NOCLOSEPROCESS hands back a handle to the
// launched process instead, so there is nothing to guess at and nothing to poll.
type shellExecuteInfoW struct {
	cbSize         uint32
	fMask          uint32
	hwnd           windows.HWND
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       windows.Handle
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      windows.Handle
	dwHotKey       uint32
	hIconOrMonitor windows.Handle
	hProcess       windows.Handle
}

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskNoAsync        = 0x00000100
	seeMaskFlagNoUI       = 0x00000400
	swShowNormal          = 5
)

var procShellExecuteExW = modshell32.NewProc("ShellExecuteExW")

// shellExecuteAndWaitForStart launches file with the given verb and reports a
// handle to the process that started.
//
// Blocks for as long as the shell takes, which for runas is as long as the user
// takes to answer UAC. Declining it comes back as ERROR_CANCELLED, so a refusal
// is distinguishable from a failure to launch.
//
// The caller owns the returned handle.
func shellExecuteAndWaitForStart(verb, file, params, dir string) (windows.Handle, error) {
	verbPtr, err := windows.UTF16PtrFromString(verb)
	if err != nil {
		return 0, err
	}
	filePtr, err := windows.UTF16PtrFromString(file)
	if err != nil {
		return 0, err
	}
	var paramsPtr *uint16
	if params != "" {
		if paramsPtr, err = windows.UTF16PtrFromString(params); err != nil {
			return 0, err
		}
	}
	dirPtr, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}

	info := shellExecuteInfoW{
		cbSize: uint32(unsafe.Sizeof(shellExecuteInfoW{})),
		// NOASYNC keeps the call from returning before the shell has finished with
		// it, which is what makes hProcess meaningful for a process that exits.
		fMask:        seeMaskNoCloseProcess | seeMaskNoAsync | seeMaskFlagNoUI,
		lpVerb:       verbPtr,
		lpFile:       filePtr,
		lpParameters: paramsPtr,
		lpDirectory:  dirPtr,
		nShow:        swShowNormal,
	}

	r, _, callErr := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return 0, fmt.Errorf("ShellExecuteExW %s: %w", verb, callErr)
		}
		return 0, fmt.Errorf("ShellExecuteExW %s failed", verb)
	}
	if info.hProcess == 0 {
		// The shell handed the request to an already-running instance rather than
		// starting anything. Not something either restart path can act on.
		return 0, fmt.Errorf("ShellExecuteExW %s started no process", verb)
	}
	return info.hProcess, nil
}
