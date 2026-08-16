package platform

import "strings"

// The account card's configuration, mirroring frontend/src/lib/accountCard/types.ts.
//
// The backend stores and validates this but never renders from it: the card is
// drawn in the frontend, and the presets are frozen values in its code. Only
// the preset's name travels, so preset refinements ship with an update rather
// than being pinned to whatever a settings file was written with.

const CardConfigVersion = 1

type CardBlockConfig struct {
	Kind    string `json:"kind"`
	Enabled bool   `json:"enabled"`
	// Empty means the block's own default.
	Display string `json:"display,omitempty"`
}

type CardRow struct {
	Blocks []CardBlockConfig `json:"blocks"`
}

type CardLayout struct {
	MinWidth  int `json:"minWidth"`
	MaxWidth  int `json:"maxWidth"`
	MinHeight int `json:"minHeight"`
	// In em rather than px, matching the card's long-standing avatar sizing.
	AvatarEm float64 `json:"avatarEm"`
	// Multiplies the card's text size, so a larger preset reads larger rather
	// than only sitting in more whitespace.
	FontScale float64 `json:"fontScale"`

	Rows             []CardRow `json:"rows"`
	StatusBadgeStyle string    `json:"statusBadgeStyle,omitempty"`
}

// AccountCardConfig is stored globally in AppSettings and, optionally, per
// platform. A nil pointer on a platform means "inherit", which is how a
// platform's own layout survives its customisation toggle being turned off.
type AccountCardConfig struct {
	Version int    `json:"version"`
	Preset  string `json:"preset"`

	// Explicit user decisions layered over the active preset's defaults.
	Blocks   map[string]bool   `json:"blocks,omitempty"`
	Displays map[string]string `json:"displays,omitempty"`

	StatusBadgeStyle string `json:"statusBadgeStyle,omitempty"`
	// Per-state colour overrides. An absent state keeps the theme's own colour.
	Colors map[string]string `json:"colors,omitempty"`

	Custom *CardLayout `json:"custom,omitempty"`
}

var validPresets = map[string]bool{"small": true, "medium": true, "large": true, "custom": true}

var validBlockKinds = map[string]bool{
	"avatar": true, "accountLogin": true, "displayName": true, "tags": true,
	"note": true, "gameStats": true, "platformId": true, "lastUsed": true,
	"statusLine": true, "badges": true,
}

var validDisplays = map[string]bool{"text": true, "icon": true, "iconText": true}

var validBadgeStyles = map[string]bool{"border": true, "corner": true, "block": true}

var validColorStates = map[string]bool{"rest": true, "hover": true, "selected": true, "current": true}

// isHexColor accepts only a plain hex colour. These values end up in a
// stylesheet, so anything that is not a colour has no business reaching one.
func isHexColor(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) != 4 && len(v) != 7 && len(v) != 9 {
		return false
	}
	if v[0] != '#' {
		return false
	}
	for _, c := range v[1:] {
		isDigit := c >= '0' && c <= '9'
		isLower := c >= 'a' && c <= 'f'
		isUpper := c >= 'A' && c <= 'F'
		if !isDigit && !isLower && !isUpper {
			return false
		}
	}
	return true
}

// DefaultAccountCardConfig is the card as it has always shipped: the small
// preset, with nothing overridden.
func DefaultAccountCardConfig() AccountCardConfig {
	return AccountCardConfig{Version: CardConfigVersion, Preset: "small"}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 {
	if !(v > 0) || v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// NormalizeAccountCardConfig brings a stored config into range and drops
// anything unrecognised while keeping the rest, so a config written by a newer
// build degrades to something usable instead of being discarded.
func NormalizeAccountCardConfig(c AccountCardConfig) AccountCardConfig {
	out := AccountCardConfig{Version: CardConfigVersion}

	preset := strings.TrimSpace(c.Preset)
	if !validPresets[preset] {
		preset = "small"
	}
	out.Preset = preset

	if len(c.Blocks) > 0 {
		blocks := map[string]bool{}
		for k, v := range c.Blocks {
			if validBlockKinds[k] {
				blocks[k] = v
			}
		}
		if len(blocks) > 0 {
			out.Blocks = blocks
		}
	}

	if len(c.Displays) > 0 {
		displays := map[string]string{}
		for k, v := range c.Displays {
			if validBlockKinds[k] && validDisplays[v] {
				displays[k] = v
			}
		}
		if len(displays) > 0 {
			out.Displays = displays
		}
	}

	if validBadgeStyles[c.StatusBadgeStyle] {
		out.StatusBadgeStyle = c.StatusBadgeStyle
	}

	if len(c.Colors) > 0 {
		colors := map[string]string{}
		for k, v := range c.Colors {
			if validColorStates[k] && isHexColor(v) {
				colors[k] = strings.ToLower(strings.TrimSpace(v))
			}
		}
		if len(colors) > 0 {
			out.Colors = colors
		}
	}

	if c.Custom != nil {
		layout := normalizeCardLayout(*c.Custom)
		out.Custom = &layout
	}

	return out
}

func normalizeCardLayout(l CardLayout) CardLayout {
	out := CardLayout{
		MinWidth:  clampInt(l.MinWidth, 72, 320),
		MaxWidth:  clampInt(l.MaxWidth, 72, 320),
		MinHeight: clampInt(l.MinHeight, 80, 400),
		AvatarEm:  clampFloat(l.AvatarEm, 1.5, 12),
		FontScale: clampFloat(l.FontScale, 0.8, 2),
	}
	// A max below the min would make the grid track invalid. Widen rather than
	// reject: the user's intent is still readable from the two numbers.
	if out.MaxWidth < out.MinWidth {
		out.MinWidth, out.MaxWidth = out.MaxWidth, out.MinWidth
	}

	if validBadgeStyles[l.StatusBadgeStyle] {
		out.StatusBadgeStyle = l.StatusBadgeStyle
	} else {
		out.StatusBadgeStyle = "border"
	}

	seen := map[string]bool{}
	for _, row := range l.Rows {
		var blocks []CardBlockConfig
		for _, b := range row.Blocks {
			if !validBlockKinds[b.Kind] || seen[b.Kind] {
				continue
			}
			seen[b.Kind] = true
			display := b.Display
			if !validDisplays[display] {
				display = ""
			}
			blocks = append(blocks, CardBlockConfig{Kind: b.Kind, Enabled: b.Enabled, Display: display})
		}
		if len(blocks) > 0 {
			out.Rows = append(out.Rows, CardRow{Blocks: blocks})
		}
	}

	return out
}
