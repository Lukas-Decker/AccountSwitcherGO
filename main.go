package main

import (
	"embed"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"account-switcher/internal/actionlog"
	"account-switcher/internal/app"
	"account-switcher/internal/appclient"
	"account-switcher/internal/applog"
	"account-switcher/internal/basic"
	"account-switcher/internal/cli"
	"account-switcher/internal/controllerinput"
	"account-switcher/internal/crashlog"
	"account-switcher/internal/discordrpc"
	"account-switcher/internal/i18n"
	"account-switcher/internal/ipc"
	"account-switcher/internal/paths"
	"account-switcher/internal/platform"
	"account-switcher/internal/riotservice"
	"account-switcher/internal/security"
	"account-switcher/internal/shortcuts"
	"account-switcher/internal/steam"
	"account-switcher/internal/tray"
	"account-switcher/internal/winutil"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/trayicon.png
var trayIconPNG []byte

// The tray menu and other native surfaces translate in Go, reading these JSON
// files. An installed copy has no source tree beside the exe, so without them
// compiled in the tray falls back to raw keys like "Tray_Exit".
//
//go:embed frontend/src/Resources/*.json
var localeResources embed.FS

var (
	platformSvc   = &platform.PlatformService{}
	basicSvc      = basic.NewBasicService(platformSvc)
	steamSvc      = steam.NewSteamService()
	controllerSvc = controllerinput.NewService()
	securitySvc   = security.NewService()
	discordRPC    = discordrpc.NewManager()
	riotSvc       = riotservice.New()
)

func init() {
	winutil.SetEmbeddedFrontendFS(assets)
	i18n.SetEmbeddedResources(localeResources, "frontend/src/Resources")

	application.RegisterEvent[string]("navigate")

	application.RegisterEvent[app.ToastPayload]("toast")
	application.RegisterEvent[string](controllerinput.EventName)
	application.RegisterEvent[steam.AccountPatch](steam.AccountUpdatedEvent)
	application.RegisterEvent[basic.AccountImagePatch](basic.AccountImageUpdatedEvent)
	application.RegisterEvent[basic.GameStatsUpdatedPatch](basic.GameStatsUpdatedEvent)
	application.RegisterEvent[string](platform.ActionBarStatusEvent)
	application.RegisterEvent[shortcuts.ListPayload](shortcuts.UpdatedEvent)
	application.RegisterEvent[shortcuts.FilesDroppedPayload](shortcuts.FilesDroppedEvent)
	application.RegisterEvent[platform.UpdateAvailablePayload](platform.AppUpdateAvailableEvent)
	application.RegisterEvent[bool](platform.UpdateCheckFailedEvent)
	application.RegisterEvent[platform.PlatformsJSONUpdatePayload](platform.PlatformsJSONUpdateFoundEvent)
	application.RegisterEvent[platform.PlatformsJSONUpdatePayload](platform.PlatformsJSONUpdatedEvent)
	application.RegisterEvent[platform.UserDataMoveProgressPayload](platform.UserDataMoveProgressEvent)

	platform.SetSteamLaunchHooks(steam.SaveFolderFromConfirmedExe, steam.ResolveSteamExePath)
	platform.SetSteamReset(steam.ResetToDefaults)
	platform.SetControllerSupportChangedHook(controllerSvc.SetEnabled)
	platform.SetDiscordPresenceRefreshHook(discordRPC.RefreshAsync)
	platform.SetPlatformLaunchers(func() error { return steam.LaunchSteamOnly(nil) }, func(platformKey string) error {
		return basic.LaunchBasic(basic.FlowDeps{PS: platformSvc}, platformKey, nil)
	})
	platform.SetPlatformLaunchAs(func(forceAdmin bool) error { return steam.LaunchSteamOnlyAs(forceAdmin, nil) }, func(platformKey string, forceAdmin bool) error {
		return basic.LaunchBasicAs(basic.FlowDeps{PS: platformSvc}, platformKey, forceAdmin, nil)
	})
	security.SetStatusChangedHook(func() {
		if !security.AppLocked() {
			basic.SyncAllTrayKnownAccounts()
			steam.SyncTrayKnownAccounts()
		}
		tray.RefreshMenuIfSet()
	})
	// Translated here, where the user's language is resolvable, so the layers
	// below can raise a toast without knowing about either i18n or the app.
	platform.SetToastHook(func(typ, _, messageKey string, vars map[string]string) {
		exeDir, err := platform.ResolveExeDir()
		language := "en-US"
		if err == nil {
			if s, lerr := platform.LoadAppSettings(exeDir); lerr == nil && strings.TrimSpace(s.Language) != "" {
				language = s.Language
			}
		}
		app.EmitToast(typ, "", i18n.T(exeDir, language, messageKey, vars), 9000)
	})
	// Riot enrichment, registered here because internal/basic cannot import the
	// service that reads its account links without forming a cycle.
	basic.SetAccountCapturedHook(func(platformKey, uniqueID string) {
		if platformKey != riotservice.PlatformKey {
			return
		}
		if _, err := riotservice.CaptureFromClient(uniqueID); err != nil {
			slog.Debug("riot capture failed", "uniqueID", uniqueID, "err", err)
		}
	})
	basic.SetProfileImageSourceHook(func(platformKey, uniqueID string) string {
		if platformKey != riotservice.PlatformKey {
			return ""
		}
		return riotservice.ProfileIconURLFor(uniqueID)
	})
	app.RegisterStartupAccountCounts()
}

func main() {
	exeDir, err := platform.ResolveExeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "exe dir:", err)
		os.Exit(1)
	}
	if err := platform.InitDataPaths(exeDir); err != nil {
		fmt.Fprintln(os.Stderr, "init data paths:", err)
		os.Exit(1)
	}
	security.CleanupTransientState()

	idx, idxErr := cli.LoadPlatformIndex()
	idxPtr := idx
	if idxErr != nil {
		log.Printf("cli platforms index: %v", idxErr)
		idxPtr = nil
	}

	parsed, err := cli.Parse(os.Args[1:], idxPtr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	lvl := app.ResolvedLogLevel(parsed)
	// The GUI build has no console of its own, so stderr only reaches anyone when
	// the app was started from a terminal. Attach to that terminal if there is one.
	winutil.AttachParentConsole()
	if dataRoot, derr := paths.DataRoot(); derr == nil {
		if lerr := applog.Init(dataRoot, lvl); lerr != nil {
			fmt.Fprintln(os.Stderr, "log file:", lerr)
		}
		defer applog.Close()
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})))
	}
	// Chunks of the codebase, the process-killing path among them, still log
	// through the standard logger. Its default output is the same dead stderr, so
	// without this those lines are written nowhere no matter what slog does.
	log.SetOutput(applog.Writer())
	actionlog.Init()

	// Verbosity follows the stored preference, and the hook lets Settings change
	// it without a restart, which is the only way to capture something already
	// happening.
	platform.SetDebugLevelHook(func(debug bool) {
		level := lvl
		if debug {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(applog.Writer(), &slog.HandlerOptions{Level: level})))
	})

	startupSettings, _ := loadStartupSettings()
	syncOfflineModeFromSettings(startupSettings)
	platform.ApplyDebugLogging(startupSettings.DebugLogging)

	defer crashlog.CaptureFatal()

	if parsed.Kind == cli.KindHelp || parsed.Help {
		fmt.Print(cli.HelpText())
		os.Exit(0)
	}

	disp := &app.Dispatch{
		SteamSvc:    steamSvc,
		BasicSvc:    basicSvc,
		PlatformSvc: platformSvc,
	}

	if parsed.IsListCommand() {
		winutil.AttachParentConsole()
		if err := disp.RunList(parsed, idx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	releaseSingleton, running, err := winutil.TryAcquireSingleton()
	if err != nil {
		fmt.Fprintln(os.Stderr, "singleton:", err)
		os.Exit(1)
	}
	if running {
		if ferr := ipc.ForwardArgs(os.Args[1:]); ferr != nil {
			fmt.Fprintln(os.Stderr, "another instance is running; IPC forward failed:", ferr)
			os.Exit(1)
		}
		os.Exit(0)
	}
	defer releaseSingleton()
	winutil.RegisterSingletonReleaser(releaseSingleton)

	platform.RunUserDataMoveCleanup(exeDir, parsed.UserDataMoveFrom, parsed.UserDataMoveTo)

	if parsed.NeedsHeadlessMutex() {
		winutil.AttachParentConsole()
		if herr := disp.RunHeadless(parsed); herr != nil {
			fmt.Fprintln(os.Stderr, herr)
			os.Exit(1)
		}
		os.Exit(0)
	}

	app.RunGUI(app.RunGUIParams{
		Parsed:         parsed,
		GuiSettings:    startupSettings,
		Services:       serviceList(),
		Dispatch:       disp,
		DiscordRPC:     discordRPC,
		StartupToast:   parsed.StartupToast,
		EmbeddedAssets: assets,
		TrayIconPNG:    trayIconPNG,
	})
}

func serviceList() []application.Service {
	return []application.Service{
		application.NewService(&FilesystemService{}),
		application.NewService(platformSvc),
		application.NewService(steamSvc),
		application.NewService(controllerSvc),
		application.NewService(basicSvc),
		application.NewService(securitySvc),
		application.NewService(riotSvc),
		application.NewService(shortcuts.NewService(platformSvc)),
	}
}

func loadStartupSettings() (platform.AppSettings, error) {
	d, err := platform.ResolveExeDir()
	if err != nil {
		return platform.AppSettings{}, err
	}
	return platform.LoadAppSettings(d)
}

func syncOfflineModeFromSettings(s platform.AppSettings) {
	appclient.SetOfflineMode(s.OfflineMode)
}
