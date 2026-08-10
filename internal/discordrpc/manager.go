package discordrpc

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"account-switcher/internal/platform"

	richgo "github.com/hugolgst/rich-go/client"
)

const (
	refreshPeriod        = 30 * time.Second
	discordLargeImageKey = "switcher"
	discordSmallImageKey = "switcher_small"
)

type Manager struct {
	mu        sync.Mutex
	refreshMu sync.Mutex

	initialized bool
	appID       string
	startedAt   time.Time
	stopCh      chan struct{}

	lastDetails string
	lastState   string
}

func logRPC() *slog.Logger {
	return slog.Default().With("component", "discord-rpc")
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Start() {
	m.mu.Lock()
	if m.stopCh != nil {
		m.mu.Unlock()
		logRPC().Debug("start skipped: manager already running")
		return
	}
	m.stopCh = make(chan struct{})
	stopCh := m.stopCh
	m.mu.Unlock()

	logRPC().Info("manager started", "refreshPeriod", refreshPeriod.String())
	go m.runPeriodic(stopCh)
	m.Refresh()
}

func (m *Manager) Stop() {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()

	m.mu.Lock()
	stopCh := m.stopCh
	m.stopCh = nil
	m.mu.Unlock()
	if stopCh != nil {
		close(stopCh)
	}
	logRPC().Info("manager stopping")
	m.shutdown()
}

func (m *Manager) RefreshAsync() {
	go m.Refresh()
}

func (m *Manager) Refresh() {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()

	settings, err := loadCurrentSettings()
	if err != nil {
		logRPC().Warn("refresh skipped: failed to load settings", "err", err)
		return
	}
	if settings.OfflineMode || !settings.DiscordRpc {
		logRPC().Info("refresh gate: rpc disabled", "offlineMode", settings.OfflineMode, "discordRpc", settings.DiscordRpc)
		m.shutdown()
		return
	}
	// Discord names the presence after whoever owns the application id, so
	// without one of our own there is nothing to publish under.
	appID := strings.TrimSpace(settings.DiscordAppID)
	if appID == "" {
		logRPC().Info("refresh gate: no Discord application id configured")
		m.shutdown()
		return
	}
	if err := m.ensureStarted(appID); err != nil {
		logRPC().Warn("refresh skipped: rpc start failed", "err", err)
		return
	}

	activity := richgo.Activity{
		State:      "",
		Details:    "Currently switching accounts",
		LargeImage: discordLargeImageKey,
		LargeText:  "Account Switcher",
		SmallImage: discordSmallImageKey,
		SmallText:  "Account Switcher",
		Timestamps: &richgo.Timestamps{Start: &m.startedAt},
	}

	if activity.Details == m.lastDetails && activity.State == m.lastState {
		if err := richgo.SetActivity(activity); err != nil {
			logRPC().Warn("set activity failed", "err", err)
		}
		return
	}

	if err := richgo.SetActivity(activity); err != nil {
		logRPC().Warn("set activity failed", "err", err)
		return
	}
	m.lastDetails = activity.Details
	m.lastState = activity.State
	logRPC().Debug("activity updated", "details", activity.Details, "state", activity.State)
}

func (m *Manager) ensureStarted(appID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.initialized && m.appID == appID {
		return nil
	}
	if m.initialized {
		// A different application means a different name in Discord, so the old
		// connection has to go before the new one is opened.
		richgo.Logout()
		m.initialized = false
	}
	if err := richgo.Login(appID); err != nil {
		return err
	}
	now := time.Now()
	m.startedAt = now
	m.initialized = true
	m.appID = appID
	logRPC().Info("rpc client initialized", "appID", appID)
	return nil
}

func (m *Manager) shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.appID = ""
	if !m.initialized {
		return
	}
	m.lastDetails = ""
	m.lastState = ""
	if err := clearPresenceDiscord(); err != nil {
		logRPC().Warn("clear presence before logout failed", "err", err)
	} else {
		logRPC().Info("presence cleared (SET_ACTIVITY null)")
	}
	richgo.Logout()
	m.initialized = false
	m.startedAt = time.Time{}
	logRPC().Info("rpc client logged out")
}

func (m *Manager) runPeriodic(stopCh <-chan struct{}) {
	ticker := time.NewTicker(refreshPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			m.Refresh()
		}
	}
}

func loadCurrentSettings() (platform.AppSettings, error) {
	exeDir, err := platform.ResolveExeDir()
	if err != nil {
		return platform.AppSettings{}, err
	}
	settings, err := platform.LoadAppSettings(exeDir)
	if err != nil {
		return platform.AppSettings{}, err
	}
	settings.Language = strings.TrimSpace(settings.Language)
	return settings, nil
}
