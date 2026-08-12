package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"account-switcher/internal/appclient"
	"account-switcher/internal/screenprivacy"
	"account-switcher/internal/streamer"
	"account-switcher/internal/winutil"
)

func (p *PlatformService) SetLanguage(code string) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.Language = strings.TrimSpace(code)
		if s.Language == "" {
			s.Language = "en-US"
		}
		return nil
	})
}

func (p *PlatformService) SetTheme(themeID string) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		nextTheme := sanitizeThemeID(themeID)
		if s.Theme != nextTheme {
			s.ThemeAccentPreset = ""
			s.ThemeAccentCustom = ""
		}
		s.Theme = nextTheme
		return nil
	})
}

func (p *PlatformService) SetThemeAccentPreset(preset string) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.ThemeAccentPreset = sanitizeThemeAccentPreset(preset)
		return nil
	})
}

func (p *PlatformService) SetThemeAccentCustom(color string) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.ThemeAccentCustom = sanitizeHexColor(color)
		return nil
	})
}

func (p *PlatformService) SetThemeHueRotate(deg int) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.ThemeHueRotate = normalizeThemeHueRotate(deg)
		return nil
	})
}

func (p *PlatformService) SetRoundedCorners(enabled bool) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.RoundedCorners = enabled
		return nil
	})
}

func (p *PlatformService) SetDiscordAppID(id string) error {
	err := p.withSettingsWrite(func(s *AppSettings) error {
		s.DiscordAppID = sanitizeDiscordAppID(id)
		return nil
	})
	if err != nil {
		return err
	}
	TriggerDiscordPresenceRefresh()
	return nil
}

func (p *PlatformService) SaveHomeOrder(order []string) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		exeDir, err := ResolveExeDir()
		if err != nil {
			return err
		}
		raw, err := LoadPlatformsJSON(exeDir)
		if err != nil {
			return err
		}
		allNames, err := parsePlatformNames(raw)
		if err != nil {
			return err
		}
		disabled := sliceToSet(s.DisabledPlatforms)
		enabled := make(map[string]struct{})
		for _, n := range allNames {
			if _, hid := disabled[n]; !hid {
				enabled[n] = struct{}{}
			}
		}
		if len(order) != len(enabled) {
			return errors.New("order length does not match enabled platforms")
		}
		seen := make(map[string]struct{})
		for _, n := range order {
			if _, ok := enabled[n]; !ok {
				return errors.New("invalid platform in order: " + n)
			}
			if _, dup := seen[n]; dup {
				return errors.New("duplicate platform in order")
			}
			seen[n] = struct{}{}
		}
		s.PlatformOrder = append([]string(nil), order...)
		return nil
	})
}

func (p *PlatformService) SetDisabledPlatforms(disabled []string) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		exeDir, err := ResolveExeDir()
		if err != nil {
			return err
		}
		raw, err := LoadPlatformsJSON(exeDir)
		if err != nil {
			return err
		}
		allNames, err := parsePlatformNames(raw)
		if err != nil {
			return err
		}
		valid := make(map[string]struct{}, len(allNames))
		for _, n := range allNames {
			valid[n] = struct{}{}
		}
		nextDis := make(map[string]struct{})
		for _, n := range disabled {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			if _, ok := valid[n]; !ok {
				continue
			}
			nextDis[n] = struct{}{}
		}
		prevDis := sliceToSet(s.DisabledPlatforms)

		var order []string
		seen := make(map[string]struct{})
		for _, n := range s.PlatformOrder {
			if _, d := nextDis[n]; d {
				continue
			}
			if _, ok := valid[n]; !ok {
				continue
			}
			order = append(order, n)
			seen[n] = struct{}{}
		}
		var newlyEnabled []string
		for _, n := range allNames {
			_, was := prevDis[n]
			_, now := nextDis[n]
			if was && !now {
				if _, ok := seen[n]; !ok {
					newlyEnabled = append(newlyEnabled, n)
				}
			}
		}
		sortStringsFold(newlyEnabled)
		for _, n := range newlyEnabled {
			order = append(order, n)
			seen[n] = struct{}{}
		}
		for _, n := range allNames {
			if _, d := nextDis[n]; d {
				continue
			}
			if _, ok := seen[n]; ok {
				continue
			}
			order = append(order, n)
		}
		s.DisabledPlatforms = setToSortedSlice(nextDis)
		s.PlatformOrder = order
		return nil
	})
}

func (p *PlatformService) SetPlatformExePath(platformKey, exePath string) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		if s.PlatformExePaths == nil {
			s.PlatformExePaths = map[string]string{}
		}
		exePath = strings.TrimSpace(exePath)
		if exePath == "" {
			delete(s.PlatformExePaths, platformKey)
		} else {
			s.PlatformExePaths[platformKey] = exePath
		}
		return nil
	})
}

func (p *PlatformService) SetProtocolEnabled(enabled bool) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self = filepath.Clean(self)

	err = p.withSettingsWrite(func(s *AppSettings) error {
		s.ProtocolEnabled = enabled
		return nil
	})
	if err != nil {
		return err
	}

	if enabled {
		return winutil.RegisterProtocol(self)
	}
	_ = winutil.UnregisterProtocol()
	return nil
}

func (p *PlatformService) SetPrereleaseUpdates(enabled bool) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.PrereleaseUpdates = enabled
		return nil
	})
}

func (p *PlatformService) SetOfflineMode(enabled bool) error {
	err := p.withSettingsWrite(func(s *AppSettings) error {
		s.OfflineMode = enabled
		if enabled {
			s.DiscordRpc = false
		}
		return nil
	})
	if err != nil {
		return err
	}
	appclient.SetOfflineMode(enabled)
	TriggerDiscordPresenceRefresh()
	return nil
}

func (p *PlatformService) SetDiscordRpc(enabled bool) error {
	err := p.withSettingsWrite(func(s *AppSettings) error {
		if s.OfflineMode {
			enabled = false
		}
		s.DiscordRpc = enabled
		return nil
	})
	if err != nil {
		return err
	}
	TriggerDiscordPresenceRefresh()
	return nil
}

func (p *PlatformService) SetExitToTray(enabled bool) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.ExitToTray = enabled
		return nil
	})
}

func (p *PlatformService) SetMinimizeOnSwitch(enabled bool) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.MinimizeOnSwitch = enabled
		return nil
	})
}

func (p *PlatformService) SetStartTrayWithWindows(enabled bool) error {
	err := p.withSettingsWrite(func(s *AppSettings) error {
		s.StartTrayWithWindows = enabled
		return nil
	})
	if err != nil {
		return err
	}
	return SetAutostartPreference(enabled)
}

func (p *PlatformService) SetStartProgramCentered(enabled bool) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.StartProgramCentered = enabled
		return nil
	})
}

func (p *PlatformService) SetAnimationsEnabled(enabled bool) error {
	return p.withSettingsWrite(func(s *AppSettings) error {
		s.AnimationsEnabled = enabled
		return nil
	})
}

func (p *PlatformService) SetStreamerMode(enabled bool) error {
	if err := p.withSettingsWrite(func(s *AppSettings) error {
		s.StreamerMode = enabled
		return nil
	}); err != nil {
		return err
	}
	streamer.SetManual(enabled)
	return nil
}

func (p *PlatformService) SetAutoStreamerMode(enabled bool) error {
	if err := p.withSettingsWrite(func(s *AppSettings) error {
		s.AutoStreamerMode = enabled
		return nil
	}); err != nil {
		return err
	}
	streamer.SetAutoEnabled(enabled)
	return nil
}

func (p *PlatformService) SetHideFromScreenshots(enabled bool) error {
	if err := p.withSettingsWrite(func(s *AppSettings) error {
		s.HideFromScreenshots = enabled
		return nil
	}); err != nil {
		return err
	}
	screenprivacy.SetEnabled(enabled)
	return nil
}

func (p *PlatformService) SetDesktopHomeShortcut(create bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return winutil.SetHomeDesktopShortcut(create)
}
