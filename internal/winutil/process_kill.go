//go:build windows

package winutil

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"account-switcher/internal/crashlog"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const servicePrefix = "SERVICE:"

const (
	gracefulExitMaxWait         = 12 * time.Second
	gracefulCombinedExitMaxWait = 5 * time.Second
)

// KillByName terminates processes by image name (e.g. "steam.exe") or stops Windows services
// when the name is prefixed with SERVICE:.
// beforeElectronSynth, when non-nil, runs before Electron Alt+F4 (e.g. launch platform + wait for foreground).
func KillByName(names []string, method ClosingMethod, beforeElectronSynth func() error) error {
	_, err := KillByNameWithOptions(names, KillOptions{
		Method:              method,
		AllowForce:          true,
		BeforeElectronSynth: beforeElectronSynth,
	})
	return err
}

// KillOptions controls how hard the app tries to close a platform.
//
// The defaults exist because closing is not free: a program that is killed
// before it has finished shutting down never writes out what it was holding, and
// for a game launcher that is the login session the switcher is about to read.
// Whether to go that far is therefore the user's call, not a constant.
type KillOptions struct {
	Method ClosingMethod
	// GraceTimeout is how long a process gets to close on its own. Zero uses the
	// method's own default.
	GraceTimeout time.Duration
	// AllowForce decides what happens when that time runs out: terminate the
	// process, or leave it running and say so.
	AllowForce bool
	// BeforeElectronSynth runs before the Electron Alt+F4 synthesis.
	BeforeElectronSynth func() error
}

// KillResult reports what it took.
type KillResult struct {
	// Forced lists images that ignored the polite request and were terminated.
	// Anything they had not yet written to disk is gone.
	Forced []string
	// Survived lists images still running because force was not permitted.
	Survived []string
}

// KillByNameWithOptions terminates processes by image name, or stops Windows
// services when the name is prefixed with SERVICE:.
func KillByNameWithOptions(names []string, opts KillOptions) (KillResult, error) {
	var result KillResult
	if len(names) == 0 {
		return result, nil
	}
	m := opts.Method
	if m == "" {
		m = ClosingCombined
	}
	var resultMu sync.Mutex
	log.Printf("winutil: kill begin method=%s targets=%d allowForce=%v grace=%s",
		m, len(names), opts.AllowForce, opts.GraceTimeout)
	var wg sync.WaitGroup
	for _, name := range names {
		raw := strings.TrimSpace(name)
		if raw == "" {
			continue
		}
		wg.Add(1)
		go func(raw string) {
			defer crashlog.Capture()
			defer wg.Done()
			if strings.HasPrefix(strings.ToUpper(raw), strings.ToUpper(servicePrefix)) {
				svcName := strings.TrimSpace(raw[len(servicePrefix):])
				log.Printf("winutil: stopping service=%s", svcName)
				if err := stopWindowsService(svcName); err != nil {
					log.Printf("winutil: stop service failed service=%s err=%v; trying process kill fallback", svcName, err)
					_ = taskKillIM(svcName+".exe", true)
				}
				log.Printf("winutil: stop service done service=%s", svcName)
				return
			}
			base := filepath.Base(raw)
			if !strings.HasSuffix(strings.ToLower(base), ".exe") {
				base = raw + ".exe"
			}
			log.Printf("winutil: stopping process=%s method=%s", base, m)
			switch m {
			case ClosingTaskKill:
				_ = taskKillIM(base, true)
			case ClosingElectron:
				var prior windows.HWND
				if opts.BeforeElectronSynth != nil {
					prior = foregroundHWND()
					if err := opts.BeforeElectronSynth(); err != nil {
						log.Printf("winutil: electron prepare err=%v", err)
					}
					requestElectronChromiumExit(base, prior, true)
				} else {
					requestElectronChromiumExit(base, 0, false)
				}
				_ = taskKillIM(base, false)
				waitForElectronImageExit(base, electronExitMaxWait, len(names))
				_ = taskKillIM(base, true)
			case ClosingClose:
				finishGracefully(base, pickGrace(opts.GraceTimeout, gracefulExitMaxWait), len(names), opts.AllowForce, &resultMu, &result)
			default: // Combined
				finishGracefully(base, pickGrace(opts.GraceTimeout, gracefulCombinedExitMaxWait), len(names), opts.AllowForce, &resultMu, &result)
			}
			log.Printf("winutil: stop process done process=%s", base)
		}(raw)
	}
	wg.Wait()
	log.Printf("winutil: kill completed method=%s forced=%v survived=%v", m, result.Forced, result.Survived)
	return result, nil
}

// pickGrace uses the caller's timeout when they set one, else the method default.
func pickGrace(configured, fallback time.Duration) time.Duration {
	if configured > 0 {
		return configured
	}
	return fallback
}

// finishGracefully asks a process to close, waits, and then either terminates it
// or reports that it is still there.
//
// The check between the two is the whole point. Terminating unconditionally,
// which is what this used to do, means a launcher that was merely slow to start
// or that minimises to tray instead of exiting gets killed mid-write, and the
// session it was holding is lost with no indication that anything happened.
func finishGracefully(base string, grace time.Duration, targetCount int, allowForce bool, mu *sync.Mutex, result *KillResult) {
	requestGracefulProcessExit(base)
	waitForImageExit(base, grace, 100*time.Millisecond, targetCount)

	if !IsExeRunning(base) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if !allowForce {
		log.Printf("winutil: %s did not close within %s and force is off; leaving it running", base, grace)
		result.Survived = append(result.Survived, base)
		return
	}
	log.Printf("winutil: %s did not close within %s; terminating", base, grace)
	_ = taskKillIM(base, true)
	result.Forced = append(result.Forced, base)
}

// requestGracefulProcessExit closes every top-level window for matching PIDs (visible + hidden),
// then non-force taskkill. Electron tray apps often hide the real browser root HWND after the UI closes.
func requestGracefulProcessExit(exeImage string) {
	postWMCloseToMatchingProcesses(exeImage)
	_ = taskKillIM(exeImage, false)
}

func postWMCloseToMatchingProcesses(exeImage string) {
	pids, err := allPIDsForImageName(exeImage)
	if err != nil {
		log.Printf("winutil: list pids image=%s err=%v", exeImage, err)
		return
	}
	for _, pid := range pids {
		postGracefulQuitForPID(pid)
	}
}

// postGracefulQuitForPID asks every top-level HWND owned by pid to quit, including hidden hosts.
// Electron tray-only builds can keep invisible Chrome_WidgetWin_* roots; missing those leaves the process running.
func postGracefulQuitForPID(pid uint32) {
	postGracefulQuitPass(pid)
	time.Sleep(200 * time.Millisecond)
	postGracefulQuitPass(pid)
}

var gracefulQuitCb uintptr

func init() {
	gracefulQuitCb = syscall.NewCallback(func(hwnd, lParam uintptr) uintptr {
		targetPID := uint32(lParam)
		var windowPID uint32
		r0, _, _ := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if r0 == 0 {
			return 1
		}
		if windowPID != targetPID {
			return 1
		}
		owner, _, _ := procGetWindow.Call(hwnd, uintptr(winGWOwner))
		if owner != 0 {
			return 1
		}
		procPostMessageW.Call(hwnd, uintptr(winWMSysCommand), uintptr(winSCClose), 0)
		procPostMessageW.Call(hwnd, uintptr(winWMClose), 0, 0)
		return 1
	})
}

func postGracefulQuitPass(pid uint32) {
	if err := procEnumWindows.Find(); err != nil {
		return
	}
	_, _, _ = procEnumWindows.Call(gracefulQuitCb, uintptr(pid))
}

func syncSendCloseToHWNDs(hwnds []windows.HWND) {
	for _, h := range hwnds {
		hw := uintptr(h)
		procSendMessageW.Call(hw, uintptr(winWMSysCommand), uintptr(winSCClose), 0)
		procSendMessageW.Call(hw, uintptr(winWMClose), 0, 0)
	}
}

func stopWindowsService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Control(svc.Stop)
	return err
}

func taskKillIM(name string, force bool) error {
	args := []string{"/C", "taskkill"}
	if force {
		args = append(args, "/F")
	}
	args = append(args, "/T", "/IM", name)
	cmd := exec.Command("cmd.exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		s := strings.TrimSpace(string(out))
		if strings.Contains(s, "not running") || strings.Contains(s, "could not find") || strings.Contains(s, "not found") {
			return nil
		}
		return fmt.Errorf("taskkill: %w: %s", err, s)
	}
	return nil
}
