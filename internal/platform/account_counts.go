package platform

// startupAccountCountResolver returns per-platform saved account totals for UI skeleton hints.
// When nil, GetStartup omits disk-backed counts.
var startupAccountCountResolver func(platformNames []string) map[string]int

// SetStartupAccountCountResolver wires startup account totals from basic/steam (registered from main).
func SetStartupAccountCountResolver(fn func(platformNames []string) map[string]int) {
	startupAccountCountResolver = fn
}

func resolveStartupAccountCounts(platformNames []string) map[string]int {
	if startupAccountCountResolver == nil {
		return map[string]int{}
	}
	return startupAccountCountResolver(platformNames)
}



// startupTagCountResolver returns per-platform tag & tagged-account totals for UI skeleton hints.
var startupTagCountResolver func(platformNames []string) map[string]PlatformTagCountInfo

// SetStartupTagCountResolver wires startup tag totals from basic/steam (registered from main).
func SetStartupTagCountResolver(fn func(platformNames []string) map[string]PlatformTagCountInfo) {
	startupTagCountResolver = fn
}

func resolveStartupTagCounts(platformNames []string) map[string]PlatformTagCountInfo {
	if startupTagCountResolver == nil {
		return map[string]PlatformTagCountInfo{}
	}
	return startupTagCountResolver(platformNames)
}
