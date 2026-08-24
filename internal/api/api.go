package api

import (
	"net/url"
	"strings"

	"account-switcher/internal/appconfig"
)

func UserAgent(version string) string {
	return "account-switcher/" + strings.TrimSpace(version)
}

// VersionCheckURL builds the launch API check for a version, or returns "" when
// no launch API is configured for this build.
//
// The debug flag is kept in the signature because the dev and production
// endpoint files still choose between them; a build with no launch API simply
// gets nothing either way.
func VersionCheckURL(version string, debug bool) string {
	if !appconfig.Configured(appconfig.LaunchAPIURLTemplate) {
		return ""
	}
	v := url.QueryEscape(strings.TrimSpace(version))
	sep := "?"
	if strings.Contains(appconfig.LaunchAPIURLTemplate, "?") {
		sep = "&"
	}
	out := appconfig.LaunchAPIURLTemplate + sep
	if debug {
		out += "debug&"
	}
	return out + "v=" + v
}
