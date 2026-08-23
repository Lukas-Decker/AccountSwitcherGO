//go:build windows

package winutil

import (
	"sort"

	"golang.org/x/sys/windows/registry"
)

// RegistrySubKeyNames lists the immediate subkey names under keyPath, sorted.
//
// Launchers that keep no library file record their installs as one subkey per
// game, so enumerating subkeys is the only way to find what Ubisoft, Rockstar,
// and the EA app have put on this machine. A missing key is not an error worth
// surfacing, since it just means the launcher is not installed.
func RegistrySubKeyNames(keyPath string) ([]string, error) {
	hive, sub, err := parseHiveSubKeyPath(keyPath)
	if err != nil {
		return nil, err
	}
	key, err := registry.OpenKey(hive, sub, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, err
	}
	defer key.Close()

	names, err := key.ReadSubKeyNames(0)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

// RegistryStringValue reads one string value, returning "" when the key or the
// value is absent. It suits the scan paths, where a launcher key existing but
// lacking the value it usually holds is ordinary rather than exceptional.
func RegistryStringValue(keyPath, valueName string) string {
	hive, sub, err := parseHiveSubKeyPath(keyPath)
	if err != nil {
		return ""
	}
	key, err := registry.OpenKey(hive, sub, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()

	v, _, err := readRegistryValueAt(key, valueName)
	if err != nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
