package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	flag "github.com/ogier/pflag"
)

type exitPanic struct{ code int }

func panicExit(code int) { panic(exitPanic{code: code}) }

func expectExit(t *testing.T, code int, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected exit(%d)", code)
		}
		p, ok := r.(exitPanic)
		if !ok {
			t.Fatalf("unexpected panic type: %T", r)
		}
		if p.code != code {
			t.Fatalf("unexpected exit code: got %d want %d", p.code, code)
		}
	}()
	fn()
}

func withCliStubs(t *testing.T, fn func()) {
	t.Helper()
	oldParse := parseFlags
	oldArgs := flagArgs
	oldExit := exitFunc
	oldAbs := absPath
	oldEval := evalSymlinks
	oldPipe := identifyPipeFn
	oldFile := identifyFileFn
	oldFiles := identifyFilesFn
	oldVersion := version_flag
	oldRecursive := recursive_flag
	oldIncludeMeta := include_meta
	oldDepth := depth
	t.Cleanup(func() {
		parseFlags = oldParse
		flagArgs = oldArgs
		exitFunc = oldExit
		absPath = oldAbs
		evalSymlinks = oldEval
		identifyPipeFn = oldPipe
		identifyFileFn = oldFile
		identifyFilesFn = oldFiles
		version_flag = oldVersion
		recursive_flag = oldRecursive
		include_meta = oldIncludeMeta
		depth = oldDepth
	})
	fn()
}

func TestMainDispatchAndExit(t *testing.T) {
	withCliStubs(t, func() {
		parseFlags = func() {}
		exitFunc = panicExit

		version_flag = true
		flagArgs = func() []string { return []string{"file"} }
		out := captureOutput(func() {
			expectExit(t, 0, main)
		})
		if !strings.Contains(out, "c4 version") {
			t.Fatalf("expected version output, got %q", out)
		}

		version_flag = false
		recursive_flag = false
		include_meta = false
		depth = 0

		pipeCalled := false
		identifyPipeFn = func() { pipeCalled = true }
		flagArgs = func() []string { return nil }
		expectExit(t, 0, main)
		if !pipeCalled {
			t.Fatalf("expected identify_pipe branch")
		}

		fileCalled := false
		identifyFileFn = func(string) { fileCalled = true }
		flagArgs = func() []string { return []string{"one"} }
		expectExit(t, 0, main)
		if !fileCalled {
			t.Fatalf("expected identify_file branch")
		}

		filesCalled := false
		recursive_flag = true
		identifyFilesFn = func([]string) { filesCalled = true }
		expectExit(t, 0, main)
		if !filesCalled {
			t.Fatalf("expected identify_files branch")
		}
	})
}

func TestIdentifyFileAndFilesErrorPaths(t *testing.T) {
	withCliStubs(t, func() {
		exitFunc = panicExit
		absPath = func(string) (string, error) { return "", errors.New("abs-fail") }

		expectExit(t, 1, func() { identify_file("bad") })
		expectExit(t, 1, func() { identify_files([]string{"bad"}) })
	})
}

func TestIdentifyFilesDepthReset(t *testing.T) {
	withCliStubs(t, func() {
		dir := t.TempDir()
		f := filepath.Join(dir, "f.txt")
		if err := os.WriteFile(f, []byte("data"), 0o644); err != nil {
			t.Fatalf("write file error: %v", err)
		}
		absPath = func(s string) (string, error) { return s, nil }
		depth = -1
		_ = captureOutput(func() { identify_files([]string{f}) })
		if depth != 0 {
			t.Fatalf("expected depth reset to 0, got %d", depth)
		}
	})
}

func TestWalkFilesystemErrorAndSymlinkPaths(t *testing.T) {
	withCliStubs(t, func() {
		exitFunc = panicExit
		absPath = func(string) (string, error) { return "", errors.New("abs-fail") }
		expectExit(t, 1, func() { walkFilesystem(0, "x", "") })

		absPath = func(s string) (string, error) { return s, nil }
		expectExit(t, 1, func() { walkFilesystem(0, filepath.Join(t.TempDir(), "missing"), "") })

		dir := t.TempDir()
		link := filepath.Join(dir, "broken-link")
		if err := os.Symlink(filepath.Join(dir, "missing-target"), link); err != nil {
			t.Fatalf("symlink create error: %v", err)
		}
		links_flag = true
		evalSymlinks = func(string) (string, error) { return "", errors.New("link-fail") }
		id := walkFilesystem(-1, link, "")
		if id == nil {
			t.Fatalf("expected null id for broken symlink path")
		}
	})
}

func TestFileIDErrorPath(t *testing.T) {
	withCliStubs(t, func() {
		exitFunc = panicExit
		expectExit(t, 1, func() {
			_ = fileID(filepath.Join(t.TempDir(), "missing"))
		})
	})
}

func TestIdentifyPipeUsageBranch(t *testing.T) {
	devNull, err := os.Open("/dev/null")
	if err != nil {
		t.Skip("unable to open /dev/null")
	}
	defer func() { _ = devNull.Close() }()

	oldStdin := os.Stdin
	oldUsage := flag.Usage
	defer func() {
		os.Stdin = oldStdin
		flag.Usage = oldUsage
	}()

	os.Stdin = devNull
	usageCalled := false
	flag.Usage = func() { usageCalled = true }
	identify_pipe()
	if !usageCalled {
		t.Fatalf("expected flag usage for char-device stdin")
	}
}

func TestOutputPathFormattingBranch(t *testing.T) {
	include_meta = false
	absolute_flag = false
	formatting_string = "path"
	item := map[string]interface{}{
		"c4id":   "cid",
		"folder": false,
		"link":   false,
		"bytes":  int64(1),
	}
	out := captureOutput(func() { output("/tmp/path-file", item) })
	if !strings.Contains(out, ":  cid") {
		t.Fatalf("expected path-format output, got %q", out)
	}
}
