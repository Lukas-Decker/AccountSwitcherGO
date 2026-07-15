package platform

func (p *PlatformService) GetPlatformSettings(platformKey string) (PlatformSettings, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return LoadPlatformSettings(platformKey)
}

func (p *PlatformService) SavePlatformSettings(platformKey string, s PlatformSettings) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return SavePlatformSettings(platformKey, s)
}

func (p *PlatformService) ResetPlatformSettings(platformKey string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return resetPlatformJSONToDefaults(platformKey)
}

