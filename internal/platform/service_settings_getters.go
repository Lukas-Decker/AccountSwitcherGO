package platform

import (
	"account-switcher/internal/winutil"
)

func (p *PlatformService) GetLanguage() (string, error) {
	var lang string
	err := p.withSettingsRead(func(s *AppSettings) error {
		lang = s.Language
		if lang == "" {
			lang = "en-US"
		}
		return nil
	})
	return lang, err
}

func (p *PlatformService) GetTheme() (string, error) {
	var val string
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = sanitizeThemeID(s.Theme)
		return nil
	})
	return val, err
}

func (p *PlatformService) GetThemeAccentPreset() (string, error) {
	var val string
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = sanitizeThemeAccentPreset(s.ThemeAccentPreset)
		return nil
	})
	return val, err
}

func (p *PlatformService) GetThemeAccentCustom() (string, error) {
	var val string
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = sanitizeHexColor(s.ThemeAccentCustom)
		return nil
	})
	return val, err
}

func (p *PlatformService) GetThemeHueRotate() (int, error) {
	var val int
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = normalizeThemeHueRotate(s.ThemeHueRotate)
		return nil
	})
	return val, err
}

func (p *PlatformService) GetRoundedCorners() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.RoundedCorners
		return nil
	})
	return val, err
}

func (p *PlatformService) GetDiscordAppID() (string, error) {
	var val string
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = sanitizeDiscordAppID(s.DiscordAppID)
		return nil
	})
	return val, err
}

func (p *PlatformService) GetWindowsAccentColor() string {
	return CurrentWindowsAccentColor()
}

func (p *PlatformService) GetAppVersion() string {
	return appVersionFromBuildConfig()
}

func (p *PlatformService) GetProtocolEnabled() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.ProtocolEnabled
		return nil
	})
	return val, err
}

func (p *PlatformService) GetOfflineMode() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.OfflineMode
		return nil
	})
	return val, err
}

func (p *PlatformService) GetPrereleaseUpdates() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.PrereleaseUpdates
		return nil
	})
	return val, err
}

func (p *PlatformService) GetDiscordRpc() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.DiscordRpc
		return nil
	})
	return val, err
}

func (p *PlatformService) GetExitToTray() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.ExitToTray
		return nil
	})
	return val, err
}

func (p *PlatformService) GetMinimizeOnSwitch() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.MinimizeOnSwitch
		return nil
	})
	return val, err
}

func (p *PlatformService) GetStartTrayWithWindows() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.StartTrayWithWindows
		return nil
	})
	return val, err
}

func (p *PlatformService) GetStartProgramCentered() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.StartProgramCentered
		return nil
	})
	return val, err
}

func (p *PlatformService) GetAnimationsEnabled() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.AnimationsEnabled
		return nil
	})
	if err != nil {
		return true, err
	}
	return val, nil
}

func (p *PlatformService) GetStreamerMode() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.StreamerMode
		return nil
	})
	return val, err
}

func (p *PlatformService) GetAutoStreamerMode() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.AutoStreamerMode
		return nil
	})
	return val, err
}

func (p *PlatformService) GetHideFromScreenshots() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.HideFromScreenshots
		return nil
	})
	return val, err
}

func (p *PlatformService) GetDebugLogging() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.DebugLogging
		return nil
	})
	return val, err
}

func (p *PlatformService) GetSkipElevatePrompt() (bool, error) {
	var val bool
	err := p.withSettingsRead(func(s *AppSettings) error {
		val = s.SkipElevatePrompt
		return nil
	})
	return val, err
}

func (p *PlatformService) GetDesktopHomeShortcutExists() (bool, error) {
	return winutil.HomeDesktopShortcutExists(), nil
}
