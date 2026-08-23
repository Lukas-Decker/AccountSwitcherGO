package steam

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Jleagle/steam-go/steamvdf"
)

// readVDFFile parses a VDF or ACF file, turning a malformed one into an error
// instead of a crash.
//
// steamvdf indexes into its token buffer without checking, so a truncated or
// corrupt file panics rather than failing. That is not a hypothetical: Steam
// rewrites appmanifests in place during an update, and a machine that lost
// power mid-write leaves exactly such a file behind. The library scan reads
// every manifest it finds, so one bad file must cost that one game and nothing
// more.
func readVDFFile(path string) (steamvdf.KeyValue, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return steamvdf.KeyValue{}, err
	}
	return readVDFBytes(raw, path)
}

func readVDFBytes(raw []byte, path string) (kv steamvdf.KeyValue, err error) {
	defer func() {
		if r := recover(); r != nil {
			kv = steamvdf.KeyValue{}
			err = fmt.Errorf("malformed VDF %q: %v", path, r)
			steamLog.Debug("skipping malformed VDF", slog.String("path", path), slog.Any("panic", r))
		}
	}()
	return steamvdf.ReadBytes(raw)
}
