package platform

import "account-switcher/internal/updatertheme"

func (*PlatformService) SetUpdaterThemeCSS(css string) {
	updatertheme.SetCSS(css)
}
