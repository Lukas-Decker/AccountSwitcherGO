//go:build !windows

package winutil

// RegistrySubKeyNames is only supported on Windows.
func RegistrySubKeyNames(keyPath string) ([]string, error) {
	return nil, ErrUnsupported
}

// RegistryStringValue is only supported on Windows and reads as empty elsewhere.
func RegistryStringValue(keyPath, valueName string) string {
	return ""
}
