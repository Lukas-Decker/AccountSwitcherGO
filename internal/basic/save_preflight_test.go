package basic

import (
	"os"
	"path/filepath"
	"testing"

	"account-switcher/internal/platform"
)

func TestMissingRequiredLoginInputs(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "present.ini")
	if err := os.WriteFile(present, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	absent := filepath.Join(dir, "absent.ini")

	fc := func(files map[string]string, required bool) FlowContext {
		return FlowContext{
			PlatformKey: "Test",
			Descriptor:  platform.Descriptor{LoginFiles: files, AllFilesRequired: required},
		}
	}

	if got := missingRequiredLoginInputs(fc(map[string]string{present: "a"}, true)); len(got) != 0 {
		t.Errorf("a present file was reported missing: %v", got)
	}
	got := missingRequiredLoginInputs(fc(map[string]string{present: "a", absent: "b"}, true))
	if len(got) != 1 || got[0] != absent {
		t.Errorf("missing file not reported: %v", got)
	}

	// Optional login files are the platform's own business.
	if got := missingRequiredLoginInputs(fc(map[string]string{absent: "b"}, false)); len(got) != 0 {
		t.Errorf("reported a missing input on an optional platform: %v", got)
	}

	// Kinds the preflight cannot evaluate must pass through untouched, or it would
	// block a save the copy loop would have completed.
	ambiguous := map[string]string{
		filepath.Join(dir, "*.ini"):               "glob",
		"JSON_SELECT_FIRST,::" + absent + "::a.b": "json",
		"JSON_EMPTY_VALUE::" + absent + "::a.b":   "jsonEmpty",
	}
	if got := missingRequiredLoginInputs(fc(ambiguous, true)); len(got) != 0 {
		t.Errorf("preflight judged a kind it cannot evaluate: %v", got)
	}
}
