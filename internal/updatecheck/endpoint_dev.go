//go:build !production

package updatecheck

import "account-switcher/internal/api"

func updateAPIURL(version string) string {
	return api.VersionCheckURL(version, true)
}
