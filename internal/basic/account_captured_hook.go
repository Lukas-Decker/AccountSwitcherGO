package basic

import "log/slog"

// Hooks for platform-specific enrichment that this package must not import.
//
// The Riot service reads the per-account links stored here, so it depends on
// this package and cannot be depended on in return. Registering the direction of
// travel from main keeps the cycle from forming and keeps Riot's rules out of
// the generic account flow.

// accountCapturedHook is called after an account is saved or switched to, with
// the platform's own chance to record what is live right now.
var accountCapturedHook func(platformKey, uniqueID string)

// profileImageSourceHook supplies a remote profile image URL for an account when
// the platform can name one.
var profileImageSourceHook func(platformKey, uniqueID string) string

// SetAccountCapturedHook registers the post-save/post-switch callback.
func SetAccountCapturedHook(fn func(platformKey, uniqueID string)) {
	accountCapturedHook = fn
}

// SetProfileImageSourceHook registers the profile image resolver.
func SetProfileImageSourceHook(fn func(platformKey, uniqueID string) string) {
	profileImageSourceHook = fn
}

// notifyAccountCaptured runs the capture hook off the caller's path.
//
// In the background because it can reach a local game client, and a switch must
// not wait on software that may be mid-startup or wedged.
func notifyAccountCaptured(platformKey, uniqueID string) {
	fn := accountCapturedHook
	if fn == nil || platformKey == "" || uniqueID == "" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Warn("account captured hook panicked", "platform", platformKey, "err", r)
			}
		}()
		fn(platformKey, uniqueID)
	}()
}

// platformProfileImageURL asks the hook for an image URL, if one is registered.
func platformProfileImageURL(platformKey, uniqueID string) string {
	fn := profileImageSourceHook
	if fn == nil {
		return ""
	}
	return fn(platformKey, uniqueID)
}
