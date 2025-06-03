package manifest_test

import (
	"github.com/Avalanche-io/c4/manifest"
	"os"
	"testing"
)

func TestParseFileModeFlags(t *testing.T) {
	tests := []struct {
		modeStr string
		flag    os.FileMode
	}{
		{"trw-r--r--", os.ModeTemporary},
		{"lrw-r--r--", os.ModeSymlink},
		{"drw-r--r--", os.ModeDevice},
		{"srw-r--r--", os.ModeSocket},
	}

	for _, tt := range tests {
		mode, err := manifest.ParseFileMode(tt.modeStr)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", tt.modeStr, err)
		}
		if mode&tt.flag == 0 {
			t.Errorf("%s: expected flag %v set", tt.modeStr, tt.flag)
		}
	}
}
