package app

import (
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	buildinfo "account-switcher/build"
	"account-switcher/internal/basic"
	"account-switcher/internal/buildmode"
	"account-switcher/internal/cli"
	"account-switcher/internal/crashlog"
	"account-switcher/internal/discordrpc"
	"account-switcher/internal/ipc"
	"account-switcher/internal/paths"
	"account-switcher/internal/platform"
	"account-switcher/internal/screenprivacy"
	"account-switcher/internal/security"
	"account-switcher/internal/shortcuts"
	"account-switcher/internal/steam"
	"account-switcher/internal/streamer"
	"account-switcher/internal/tray"
	"account-switcher/internal/updatecheck"
	"account-switcher/internal/updatertheme"
	"account-switcher/internal/winutil"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

type RunGUIParams struct {
	Parsed           cli.Parsed
	GuiSettings      platform.AppSettings
	Services         []application.Service
	Dispatch         *Dispatch
	DiscordRPC       *discordrpc.Manager
	StartupToast     string
	EmbeddedAssets   fs.FS
	TrayIconPNG      []byte
	Done             chan struct{}
}

func ResolvedLogLevel(p cli.Parsed) slog.Level {
	lvl := p.EffectiveSlogLevel()
	if buildmode.IsDebugBuild() && !p.LogLevelSet {
		lvl = slog.LevelDebug
	}
	return lvl
}

func mainWindowOptions(guiSettings platform.AppSettings, parsed cli.Parsed) application.WebviewWindowOptions {
	winOpts := application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Account Switcher",
		MinWidth:  760,
		MinHeight: 520,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour:           application.NewRGB(27, 38, 54),
		URL:                        "/",
		Frameless:                  true,
		EnableFileDrop:             true,
		DevToolsEnabled:            true,
		DefaultContextMenuDisabled: false,
		KeyBindings: map[string]func(application.Window){
			"Ctrl+Shift+I": func(window application.Window) { window.OpenDevTools() },
			"F11":          func(window application.Window) { window.ToggleFullscreen() },
		},
		Permissions: map[application.PermissionType]application.Permission{
			application.PermissionCamera:        application.PermissionDeny,
			application.PermissionMicrophone:    application.PermissionDeny,
			application.PermissionGeolocation:   application.PermissionDeny,
			application.PermissionNotifications: application.PermissionDeny,
			application.PermissionClipboardRead: application.PermissionDeny,
		},
	}
	// Stamped at creation rather than after: a window that opens capturable and is
	// protected a frame later has already been recorded.
	screenprivacy.Apply(&winOpts)

	// Reopen at the size the user left, so resizing the window sticks.
	geometry := platform.WindowGeometryFromSettings(guiSettings)
	if geometry.Valid() {
		winOpts.Width = geometry.Width
		winOpts.Height = geometry.Height
	}

	if guiSettings.StartProgramCentered {
		winOpts.InitialPosition = application.WindowCentered
	} else if geometry.Valid() {
		winOpts.InitialPosition = application.WindowXY
		winOpts.X = geometry.X
		winOpts.Y = geometry.Y
	} else {
		winOpts.InitialPosition = application.WindowXY
		winOpts.X = 96
		winOpts.Y = 96
	}
	if parsed.StartInTray {
		winOpts.Hidden = true
	}
	return winOpts
}

func githubUpdaterConfig(guiSettings platform.AppSettings) github.Config {
	return github.Config{
		Repository:    "TCNOco/Account-Switcher",
		Prerelease:    guiSettings.PrereleaseUpdates,
		ChecksumAsset: "SHA256SUMS",
		AssetMatcher:  updatecheck.GitHubAssetMatcher,
	}
}

func RunGUI(params RunGUIParams) {
	parsed := params.Parsed
	guiSettings := params.GuiSettings
	disp := params.Dispatch

	if parsed.Kind == cli.KindOpenPage {
		platform.SetStartupNavigateHint(parsed.RouteJSONForOpenPage())
	}

	syncProtocolRegistration()

	// Drop shipped assets an older version left in the data folder, so the app's
	// own images cannot be shadowed by stale copies of themselves.
	pruneStaleWwwrootAssets(params.EmbeddedAssets)

	params.DiscordRPC.Start()

	// Before the window exists, so a broadcaster that is already running has been
	// adopted by the time the first account list paints.
	platform.InitStreamerMode(guiSettings)
	screenprivacy.SetEnabled(guiSettings.HideFromScreenshots)
	defer streamer.Shutdown()

	wailsLvl := ResolvedLogLevel(parsed)
	if !parsed.LogLevelSet && wailsLvl < slog.LevelInfo {
		wailsLvl = slog.LevelInfo
	}
	wailsLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: wailsLvl}))
	notifier := notifications.New()
	platform.SetNativeNotifier(notifier)
	services := append([]application.Service{}, params.Services...)
	services = append(services, application.NewService(notifier))

	var wailsApp *application.App
	appOpts := application.Options{
		Name:        "Account Switcher",
		Description: "A Superfast open-source account switcher",
		LogLevel:    wailsLvl,
		Logger:      wailsLogger,
		Services:    services,
		Assets: application.AssetOptions{
			Handler: newCompositeAssetHandler(params.EmbeddedAssets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.accountswitcher.app",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				handleForwardedCLI(wailsApp, disp, argvWithoutExecutable(data.Args))
			},
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	}
	if runtime.GOOS == "windows" {
		if cacheDir, err := paths.WebViewCacheDir(); err != nil {
			log.Printf("webview cache dir: %v", err)
		} else if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			log.Printf("webview cache dir: %v", err)
		} else {
			configureWindowsWebViewCache(&appOpts, cacheDir)
		}
	}

	wailsApp = application.New(appOpts)
	if err := platform.SyncAutostartPreference(wailsApp, guiSettings.StartTrayWithWindows); err != nil {
		wailsApp.Logger.Warn("autostart sync", "error", err)
	}

	currentVersion := buildinfo.Version()

	if platform.UpdateChecksEnabled && currentVersion != "" && !guiSettings.OfflineMode {
		gh, err := github.New(githubUpdaterConfig(guiSettings))
		if err != nil {
			wailsApp.Logger.Error("updater: provider", "error", err)
		} else {
			updaterWindow := updatertheme.NewBuiltinWindow()
			updatertheme.SetWindow(updaterWindow)
			if err := wailsApp.Updater.Init(updater.Config{
				CurrentVersion: currentVersion,
				Providers:      []updater.Provider{gh},
				Window:         updaterWindow,
			}); err != nil {
				wailsApp.Logger.Error("updater: init", "error", err)
			} else {
				platform.EnableAutoRestartAfterUpdate(wailsApp)
			}
		}

	}

	if toast := strings.TrimSpace(params.StartupToast); toast != "" {
		EmitToast("success", toast, "", 6000)
	}
	var ipcStop func()
	wailsApp.OnShutdown(func() {
		params.DiscordRPC.Stop()
		if ipcStop != nil {
			ipcStop()
		}
		if params.Done != nil {
			close(params.Done)
		}
	})

	ipcStop, err := ipc.StartGUIServer(func(argv []string) {
		handleForwardedCLI(wailsApp, disp, argv)
	})
	if err != nil {
		log.Printf("ipc server: %v", err)
	}

	winOpts := mainWindowOptions(guiSettings, parsed)
	win := wailsApp.Window.NewWithOptions(winOpts)
	screenprivacy.Follow(win)
	rememberWindowGeometry(win, guiSettings)
	registerNotificationResponseHandler(wailsApp, win, notifier)
	win.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		if len(files) == 0 {
			return
		}
		details := event.Context().DropTargetDetails()
		wailsApp.Event.Emit(shortcuts.FilesDroppedEvent, shortcuts.FilesDroppedPayload{
			Files: files,
			Target: shortcuts.FileDropTargetDetails{
				ElementID: details.ElementID,
				ClassList: append([]string(nil), details.ClassList...),
				X:         details.X,
				Y:         details.Y,
			},
		})
	})

	trayMgr := tray.NewManager(wailsApp, win, tray.Deps{
		SwapBasic: func(platformKey, uniqueID string) error {
			return disp.BasicSvc.SwapToAccount(platformKey, uniqueID, nil)
		},
		SwapSteam: func(steamID64 string, personaState int) error {
			return disp.SteamSvc.SwapToSteamAccount(steamID64, personaState, nil)
		},
	})
	trayMgr.RegisterCloseHook()
	tray.SetMenuRefresh(trayMgr.RefreshMenu)
	if !security.AppLocked() {
		basic.SyncAllTrayKnownAccounts()
		steam.SyncTrayKnownAccounts()
	}
	trayMgr.Start(params.TrayIconPNG)

	basic.SetLiveAccountIDResolver(func(platformKey string) (string, error) {
		if strings.EqualFold(strings.TrimSpace(platformKey), "Steam") {
			return steam.CurrentLiveSteamID64()
		}
		return basic.CurrentLiveUniqueID(basic.FlowDeps{PS: params.Dispatch.PlatformSvc}, platformKey)
	})
	params.Dispatch.BasicSvc.StartGameStatsProcessMonitor()
	steam.StartSteamAppListMonitor()

	ctx := wailsApp.Context()
	go func() {
		defer crashlog.Capture()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		last := ""
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			current := platform.CurrentWindowsAccentColor()
			if current != "" && current != last {
				last = current
				_ = wailsApp.Event.Emit(platform.WindowsAccentChangedEvent, current)
			}
		}
	}()

	err = wailsApp.Run()
	if err != nil {
		slog.Error("app run", "err", err)
	}
}

func argvWithoutExecutable(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	return append([]string(nil), args[1:]...)
}

func handleForwardedCLI(app *application.App, disp *Dispatch, argv []string) {
	if app == nil || disp == nil {
		return
	}
	idx, idxErr := cli.LoadPlatformIndex()
	idxPtr := idx
	if idxErr != nil {
		idxPtr = nil
	}
	p, parseErr := cli.Parse(argv, idxPtr)
	if parseErr != nil {
		application.InvokeAsync(func() {
			EmitToast("error", "CLI", parseErr.Error(), 0)
		})
		return
	}
	application.InvokeAsync(func() {
		dispatchCLIInGUI(app, p, disp)
	})
}

func registerNotificationResponseHandler(app *application.App, win *application.WebviewWindow, notifier *notifications.NotificationService) {
	if notifier == nil {
		return
	}
	notifier.OnNotificationResponse(func(result notifications.NotificationResult) {
		if result.Error != nil {
			if app != nil {
				app.Logger.Warn("notification response", "error", result.Error)
			}
			return
		}
		application.InvokeAsync(func() {
			if win != nil {
				win.Show().Focus()
			}
		})
	})
}

func dispatchCLIInGUI(app *application.App, p cli.Parsed, disp *Dispatch) {
	if p.StartInTray {
		application.InvokeSync(func() {
			w := app.Window.Current()
			if w != nil {
				_ = w.Hide()
			}
		})
	}
	switch p.Kind {
	case cli.KindSwapSteam:
		if err := disp.SteamSvc.SwapToSteamAccount(p.SteamID64, p.PersonaState, p.PassthroughLaunchArgs); err != nil {
			EmitToast("error", "Steam", err.Error(), 0)
			return
		}
		if err := disp.LaunchAfterSwap(p); err != nil {
			EmitToast("error", "i18n:Button_Launch", err.Error(), 0)
		}
	case cli.KindSwapBasic:
		if err := basic.SwapTo(basic.FlowDeps{PS: disp.PlatformSvc}, p.PlatformKey, p.UniqueID, p.PassthroughLaunchArgs); err != nil {
			EmitToast("error", "i18n:CLI_Swap", err.Error(), 0)
			return
		}
		if err := disp.LaunchAfterSwap(p); err != nil {
			EmitToast("error", "i18n:Button_Launch", err.Error(), 0)
		}
	case cli.KindLogout:
		if err := disp.RunLogout(p); err != nil {
			EmitToast("error", "i18n:CLI_Logout", err.Error(), 0)
		}
	case cli.KindOpenPage:
		application.InvokeSync(func() {
			w := app.Window.Current()
			if w != nil {
				w.Show().Focus()
			}
		})
		j := p.RouteJSONForOpenPage()
		if j != "" {
			app.Event.Emit("navigate", j)
		}
	default:
		application.InvokeSync(func() {
			w := app.Window.Current()
			if w != nil {
				w.Show().Focus()
			}
		})
		_ = app.Event.Emit("navigate", `{"page":"home"}`)
	}
}

func syncProtocolRegistration() {
	exeDir, err := platform.ResolveExeDir()
	if err != nil {
		return
	}
	s, err := platform.LoadAppSettings(exeDir)
	if err != nil || !s.ProtocolEnabled {
		return
	}
	self, err := os.Executable()
	if err != nil {
		return
	}
	_ = winutil.RegisterProtocol(filepath.Clean(self))
}
