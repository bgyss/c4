package manifest_test

import (
	"os"
	"testing"

	"github.com/Avalanche-io/c4/manifest"
)

// helper to test single mode string
func checkMode(t *testing.T, modeStr string, expect os.FileMode) {
	t.Helper()
	m, err := manifest.ParseFileMode(modeStr)
	if err != nil {
		t.Fatalf("ParseFileMode(%q) returned error: %v", modeStr, err)
	}
	if m != expect {
		t.Errorf("ParseFileMode(%q) => %#o, want %#o", modeStr, m, expect)
	}
}

func TestParseFileMode_Common(t *testing.T) {
	checkMode(t, "-rw-r--r--", 0644)
	checkMode(t, "drwxr-xr-x", os.ModeDir|0755)
}

func TestParseFileMode_Special(t *testing.T) {
	checkMode(t, "trw-r--r--", os.ModeTemporary|0644)
	checkMode(t, "lrw-r--r--", os.ModeSymlink|0644)
	checkMode(t, "drw-r--r--", os.ModeDevice|0644)
	checkMode(t, "srw-r--r--", os.ModeSocket|0644)
}
