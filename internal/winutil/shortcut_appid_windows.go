//go:build windows

package winutil

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

const gpsReadWrite = 0x2

var (
	modPropsys                            = windows.NewLazySystemDLL("propsys.dll")
	modShell32                            = windows.NewLazySystemDLL("shell32.dll")
	modShlwapi                            = windows.NewLazySystemDLL("shlwapi.dll")
	modOle32                              = windows.NewLazySystemDLL("ole32.dll")
	procInitPropVariantFromString         = modPropsys.NewProc("InitPropVariantFromString")
	procPropVariantClear                  = modPropsys.NewProc("PropVariantClear")
	procPropVariantClearOle32             = modOle32.NewProc("PropVariantClear")
	procSHStrDupW                         = modShlwapi.NewProc("SHStrDupW")
	procSHGetPropertyStoreFromParsingName = modShell32.NewProc("SHGetPropertyStoreFromParsingName")
)

// vtLPWSTR is VT_LPWSTR: a PROPVARIANT holding a wide string that the variant
// owns and frees through CoTaskMemFree.
const vtLPWSTR = 31

// propVariantStringOffset is where the union starts in a PROPVARIANT: the type
// tag plus three reserved words, on both 32- and 64-bit.
const propVariantStringOffset = 8

type propertyKey struct {
	fmtid windows.GUID
	pid   uint32
}

var pkeyAppUserModelID = propertyKey{
	fmtid: windows.GUID{
		Data1: 0x9f4c2855, Data2: 0x9f79, Data3: 0x4f39,
		Data4: [8]byte{0xa8, 0xd0, 0xe1, 0xd4, 0x2d, 0xe1, 0xd5, 0xf3},
	},
	pid: 5,
}

type propVariant struct {
	data [propVariantSize]byte
}

type iPropertyStoreVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
	getCount       uintptr
	getAt          uintptr
	getValue       uintptr
	setValue       uintptr
	commit         uintptr
}

type iPropertyStore struct {
	vtbl *iPropertyStoreVtbl
}

func (ps *iPropertyStore) release() {
	syscall.SyscallN(ps.vtbl.release, uintptr(unsafe.Pointer(ps)))
}

func (ps *iPropertyStore) setValue(key *propertyKey, val *propVariant) error {
	hr, _, _ := syscall.SyscallN(
		ps.vtbl.setValue,
		uintptr(unsafe.Pointer(ps)),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(val)),
	)
	return hresultErr(hr)
}

func (ps *iPropertyStore) commit() error {
	hr, _, _ := syscall.SyscallN(ps.vtbl.commit, uintptr(unsafe.Pointer(ps)))
	return hresultErr(hr)
}

func hresultErr(hr uintptr) error {
	if hr == 0 {
		return nil
	}
	return ole.NewError(hr)
}

// initPropVariantFromString builds a PROPVARIANT holding appID.
//
// InitPropVariantFromString is an inline helper in the Windows SDK headers, so
// propsys.dll does not always export it. Calling a missing entry point through
// a lazy proc panics rather than returning an error, which took the whole app
// down whenever a shortcut was written. It is used when present and built by
// hand otherwise.
func initPropVariantFromString(s string) (propVariant, error) {
	var pv propVariant
	ws, err := windows.UTF16PtrFromString(s)
	if err != nil {
		return pv, err
	}

	if procInitPropVariantFromString.Find() == nil {
		hr, _, _ := procInitPropVariantFromString.Call(
			uintptr(unsafe.Pointer(ws)),
			uintptr(unsafe.Pointer(&pv)),
		)
		if e := hresultErr(hr); e != nil {
			return pv, e
		}
		return pv, nil
	}

	if err := procSHStrDupW.Find(); err != nil {
		return pv, fmt.Errorf("no way to allocate a PROPVARIANT string: %w", err)
	}
	// SHStrDupW allocates with CoTaskMemAlloc, which is what PropVariantClear
	// frees, so ownership still works out. The copy is received into a typed
	// pointer so it stays visible to the collector as a pointer throughout.
	var dup *uint16
	hr, _, _ := procSHStrDupW.Call(
		uintptr(unsafe.Pointer(ws)),
		uintptr(unsafe.Pointer(&dup)),
	)
	if e := hresultErr(hr); e != nil {
		return pv, e
	}
	if dup == nil {
		return pv, fmt.Errorf("SHStrDupW returned no string")
	}
	*(*uint16)(unsafe.Pointer(&pv.data[0])) = vtLPWSTR
	*(**uint16)(unsafe.Pointer(&pv.data[propVariantStringOffset])) = dup
	return pv, nil
}

func clearPropVariant(pv *propVariant) {
	if pv == nil {
		return
	}
	// Either export frees it; ole32 carries this one on every version.
	if procPropVariantClear.Find() == nil {
		procPropVariantClear.Call(uintptr(unsafe.Pointer(pv)))
		return
	}
	if procPropVariantClearOle32.Find() == nil {
		procPropVariantClearOle32.Call(uintptr(unsafe.Pointer(pv)))
	}
}

func setShortcutAppUserModelID(lnkPath, appID string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil
	}
	if len(appID) > 128 {
		appID = appID[:128]
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	const rpcEChangedMode = uintptr(0x80010106)
	var needUninit bool
	if err := ole.CoInitialize(0); err != nil {
		oe, ok := err.(*ole.OleError)
		if !ok {
			return fmt.Errorf("com init: %w", err)
		}
		switch oe.Code() {
		case 1:
			needUninit = true
		case rpcEChangedMode:
			needUninit = false
		default:
			return fmt.Errorf("com init: %w", err)
		}
	} else {
		needUninit = true
	}
	if needUninit {
		defer ole.CoUninitialize()
	}

	pathPtr, err := windows.UTF16PtrFromString(lnkPath)
	if err != nil {
		return err
	}

	// Calling through a lazy proc that cannot be resolved panics, so check first
	// and report a missing entry point as the error it is.
	if err := procSHGetPropertyStoreFromParsingName.Find(); err != nil {
		return fmt.Errorf("SHGetPropertyStoreFromParsingName unavailable: %w", err)
	}

	iid := ole.NewGUID("{886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99}")
	// The interface pointer is received straight into a typed variable. Taking a
	// uintptr out first and converting that back to a pointer is what "possible
	// misuse of unsafe.Pointer" objects to: between the two statements the value
	// is a plain integer that the garbage collector does not recognise as a
	// reference.
	var ps *iPropertyStore
	hr, _, _ := procSHGetPropertyStoreFromParsingName.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		gpsReadWrite,
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&ps)),
	)
	if e := hresultErr(hr); e != nil {
		return fmt.Errorf("SHGetPropertyStoreFromParsingName: %w", e)
	}
	if ps == nil {
		return fmt.Errorf("SHGetPropertyStoreFromParsingName: nil store")
	}
	defer ps.release()

	pv, err := initPropVariantFromString(appID)
	if err != nil {
		return err
	}
	defer clearPropVariant(&pv)

	if err := ps.setValue(&pkeyAppUserModelID, &pv); err != nil {
		return fmt.Errorf("SetValue AppUserModelID: %w", err)
	}
	if err := ps.commit(); err != nil {
		return fmt.Errorf("Commit: %w", err)
	}
	return nil
}
