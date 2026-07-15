package platform

import (
	buildinfo "account-switcher/build"
)

func appVersionFromBuildConfig() string {
	return buildinfo.Version()
}
