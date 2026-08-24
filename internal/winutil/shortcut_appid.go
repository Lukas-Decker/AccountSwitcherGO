package winutil

import "strings"

// ShortcutAppUserModelID builds a stable Windows shell identity for a .lnk file.
// Unique IDs prevent Start Menu / pinned tiles from sharing one icon when shortcuts target the same exe.
//
// The prefix changed with the rename. Windows keys taskbar pinning and
// notification identity off this string, so shortcuts written by an older build
// keep their own grouping until they are rewritten, which the settings toggles
// do whenever a shortcut is recreated.
func ShortcutAppUserModelID(parts ...string) string {
	var b strings.Builder
	b.WriteString("AccountSwitcher.App")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		b.WriteByte('.')
		for _, r := range p {
			switch {
			case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
				b.WriteRune(r)
			case r == '_', r == '-':
				b.WriteRune(r)
			default:
				b.WriteRune('_')
			}
		}
	}
	out := b.String()
	if len(out) > 128 {
		out = out[:128]
	}
	return out
}
