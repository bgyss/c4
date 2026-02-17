package main

import (
	"io"
	"os"
	"strings"
	"testing"

	flag "github.com/ogier/pflag"
)

func TestConfigureFlagsUsageOutput(t *testing.T) {
	fs := flag.NewFlagSet("c4-flags-test", flag.ContinueOnError)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	fs.SetOutput(w)

	configureFlags(fs)
	fs.Usage()
	_ = w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read usage output: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "Usage: c4 [flags] [file]") {
		t.Fatalf("usage output missing command usage text: %q", out)
	}
	if !strings.Contains(out, "--recursive") {
		t.Fatalf("usage output missing flag defaults: %q", out)
	}
}
