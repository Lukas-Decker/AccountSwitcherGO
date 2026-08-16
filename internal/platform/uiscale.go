package platform

// Interface scale.
//
// Windows scales the webview itself whenever the display is set above 100%, so
// most high-DPI machines already look right without any help. What it does not
// cover is a high resolution monitor left at 100%, which is common on desktops
// and is where the interface ends up physically tiny. The scale exists for
// that case, and is stored rather than derived so a user who disagrees with the
// automatic choice can overrule it.

const (
	// UIScaleAuto means the frontend decides from the display it is on.
	UIScaleAuto = 0

	uiScaleMin = 0.75
	uiScaleMax = 2.0
)

// NormalizeUIScale keeps a stored scale usable. Anything outside the supported
// range is clamped rather than rejected, and anything at or below zero is taken
// to mean automatic.
func NormalizeUIScale(scale float64) float64 {
	if scale <= 0 {
		return UIScaleAuto
	}
	if scale < uiScaleMin {
		return uiScaleMin
	}
	if scale > uiScaleMax {
		return uiScaleMax
	}
	// Rounded to whole percent so the value that comes back out of a settings
	// file is the same one the slider put in.
	return float64(int(scale*100+0.5)) / 100
}
