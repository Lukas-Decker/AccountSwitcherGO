// Package appconfig holds the addresses of every external service the app can
// talk to.
//
// They live in one place, and they are variables rather than constants, because
// this build ships with no backend of its own. The services the code was
// originally written against belong to the upstream project, and pointing at
// someone else's infrastructure is not something a fork should do by default.
//
// Every one of these may therefore be empty, and callers must treat empty as
// "this feature is switched off" rather than as an error. A fork that runs its
// own services can fill them in at build time without touching any call site:
//
//	go build -ldflags "-X account-switcher/internal/appconfig.SteamAppListURL=https://example.com/apps.json"
package appconfig

import "strings"

var (
	// LaunchAPIURLTemplate is the version-check endpoint used as a fallback
	// when the signed GitHub updater cannot answer. %s takes the URL-escaped
	// current version. Empty disables the fallback, leaving GitHub as the only
	// update source.
	LaunchAPIURLTemplate = ""

	// SteamAppListXZURL and SteamAppListURL serve Steam's full appid to name
	// catalogue, compressed and plain. They only supply display names: with
	// both empty, installed games are still named from their appmanifests and
	// the rest fall back to the public community profile.
	SteamAppListXZURL = ""
	SteamAppListURL   = ""

	// CrowdinTranslatorsURL serves the translator credits shown in the
	// language settings. Empty hides the credits button.
	CrowdinTranslatorsURL = ""

	// CrowdinProjectURL is where the "help translate" link points. Empty hides
	// the link.
	CrowdinProjectURL = ""

	// GitHubRepository is the "owner/name" the signed updater checks releases
	// against, and the source of every user-facing GitHub link.
	GitHubRepository = "KeinNameVorhanden/AccountSwitcherGO"

	// GitHubBranch is the branch documentation and data files are read from.
	GitHubBranch = "main"
)

// Configured reports whether an endpoint has been set for this build.
func Configured(endpoint string) bool {
	return strings.TrimSpace(endpoint) != ""
}

// RepositoryURL is the project's page.
func RepositoryURL() string {
	if !Configured(GitHubRepository) {
		return ""
	}
	return "https://github.com/" + strings.Trim(strings.TrimSpace(GitHubRepository), "/")
}

// ReleasesPageURL is where a user is sent to download an update by hand.
func ReleasesPageURL() string {
	base := RepositoryURL()
	if base == "" {
		return ""
	}
	return base + "/releases/latest"
}

// WikiURL builds a link into the project wiki. page is appended verbatim, so a
// caller may include an anchor.
func WikiURL(page string) string {
	base := RepositoryURL()
	if base == "" {
		return ""
	}
	return base + "/wiki/" + strings.TrimPrefix(strings.TrimSpace(page), "/")
}

// RawFileURL builds a raw.githubusercontent.com link to a file on the
// configured branch.
func RawFileURL(path string) string {
	repo := strings.Trim(strings.TrimSpace(GitHubRepository), "/")
	branch := strings.TrimSpace(GitHubBranch)
	if repo == "" || branch == "" {
		return ""
	}
	return "https://raw.githubusercontent.com/" + repo + "/refs/heads/" + branch + "/" +
		strings.TrimPrefix(strings.TrimSpace(path), "/")
}
