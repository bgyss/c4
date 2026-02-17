package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	flag "github.com/ogier/pflag"
)

type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string       { return "fake" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() interface{}   { return nil }

func resetCmdState(t *testing.T) {
	t.Helper()

	origExit := exitFn
	origAbs := absPath
	origPipe := identifyPipeFn
	origFile := identifyFileFn
	origFiles := identifyFilesFn
	origStdinStat := stdinStat
	origLstat := lstatFile
	origReadDir := readDir
	origEvalSymlinks := evalSymlinks
	origOpen := openFile
	origIsTerminal := isTerminal
	origArgs := os.Args
	origCmd := flag.CommandLine
	origUsage := flag.Usage

	recursive_flag = false
	version_flag = false
	links_flag = false
	depth = 0
	include_meta = false
	absolute_flag = false
	formatting_string = "id"

	fs := flag.NewFlagSet("c4-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flag.CommandLine = fs
	configureFlags(flag.CommandLine)
	flag.Usage = fs.Usage

	t.Cleanup(func() {
		exitFn = origExit
		absPath = origAbs
		identifyPipeFn = origPipe
		identifyFileFn = origFile
		identifyFilesFn = origFiles
		stdinStat = origStdinStat
		lstatFile = origLstat
		readDir = origReadDir
		evalSymlinks = origEvalSymlinks
		openFile = origOpen
		isTerminal = origIsTerminal
		os.Args = origArgs
		flag.CommandLine = origCmd
		flag.Usage = origUsage
	})
}

func TestRunVersionBranch(t *testing.T) {
	resetCmdState(t)
	os.Args = []string{"c4", "--version"}
	out := captureOutput(func() {
		if code := run(); code != 0 {
			t.Fatalf("expected 0 exit code, got %d", code)
		}
	})
	if !strings.Contains(out, "c4 version "+version_number) {
		t.Fatalf("expected version output, got %q", out)
	}
}

func TestRunDispatchBranches(t *testing.T) {
	t.Run("pipe", func(t *testing.T) {
		resetCmdState(t)
		os.Args = []string{"c4"}
		called := false
		identifyPipeFn = func() {
			called = true
		}
		if code := run(); code != 0 {
			t.Fatalf("expected 0 exit code, got %d", code)
		}
		if !called {
			t.Fatalf("identifyPipeFn was not called")
		}
	})

	t.Run("single-file", func(t *testing.T) {
		resetCmdState(t)
		os.Args = []string{"c4", "a.txt"}
		var arg string
		identifyFileFn = func(filename string) int {
			arg = filename
			return 7
		}
		if code := run(); code != 7 {
			t.Fatalf("expected 7 exit code, got %d", code)
		}
		if arg != "a.txt" {
			t.Fatalf("expected file arg %q, got %q", "a.txt", arg)
		}
	})

	t.Run("multi-file", func(t *testing.T) {
		resetCmdState(t)
		os.Args = []string{"c4", "a.txt", "b.txt"}
		called := false
		identifyFilesFn = func(fileList []string) int {
			called = true
			if len(fileList) != 2 {
				t.Fatalf("unexpected file list length %d", len(fileList))
			}
			return 3
		}
		if code := run(); code != 3 {
			t.Fatalf("expected 3 exit code, got %d", code)
		}
		if !called {
			t.Fatalf("identifyFilesFn was not called")
		}
	})

	t.Run("recursive-single-file-goes-multi", func(t *testing.T) {
		resetCmdState(t)
		os.Args = []string{"c4", "-R", "a.txt"}
		called := false
		identifyFilesFn = func(fileList []string) int {
			called = true
			return 0
		}
		if code := run(); code != 0 {
			t.Fatalf("expected 0 exit code, got %d", code)
		}
		if !called {
			t.Fatalf("expected identifyFilesFn to be called for -R")
		}
	})
}

func TestRunUsageWhenStdinIsTTY(t *testing.T) {
	resetCmdState(t)
	os.Args = []string{"c4"}
	stdinStat = func() (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeCharDevice}, nil
	}
	usedUsage := false
	flag.Usage = func() {
		usedUsage = true
	}
	if code := run(); code != 0 {
		t.Fatalf("expected 0 exit code, got %d", code)
	}
	if !usedUsage {
		t.Fatalf("expected usage to be called")
	}
}

func TestMainCallsExitWithRunCode(t *testing.T) {
	resetCmdState(t)
	os.Args = []string{"c4", "--version"}
	var got int
	exitFn = func(code int) {
		got = code
		panic("exit-called")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic from exit stub")
		}
		if got != 0 {
			t.Fatalf("expected exit code 0, got %d", got)
		}
	}()
	main()
}

func TestIdentifyFileAndFilesAbsFailure(t *testing.T) {
	resetCmdState(t)
	absPath = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	if code := identify_file("x"); code != 1 {
		t.Fatalf("expected identify_file to return 1, got %d", code)
	}
	if code := identify_files([]string{"x"}); code != 1 {
		t.Fatalf("expected identify_files to return 1, got %d", code)
	}
}

func TestIdentifyFilesClampsNegativeDepth(t *testing.T) {
	resetCmdState(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("file"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	depth = -1
	if code := identify_files([]string{file}); code != 0 {
		t.Fatalf("expected success exit code, got %d", code)
	}
	if depth != 0 {
		t.Fatalf("expected depth to be clamped to 0, got %d", depth)
	}
}

func TestIdentifyPipeUsageOnStatError(t *testing.T) {
	resetCmdState(t)
	stdinStat = func() (os.FileInfo, error) {
		return nil, errors.New("stat failed")
	}
	called := false
	flag.Usage = func() {
		called = true
	}
	identify_pipe()
	if !called {
		t.Fatalf("expected usage to be called")
	}
}

func TestNewItemAndWalkFilesystemErrorPaths(t *testing.T) {
	t.Run("newItem-lstat-error", func(t *testing.T) {
		resetCmdState(t)
		lstatFile = func(string) (os.FileInfo, error) {
			return nil, errors.New("lstat fail")
		}
		exitCalled := false
		exitFn = func(int) {
			exitCalled = true
		}
		item := newItem("/bad/path")
		if !exitCalled {
			t.Fatalf("expected exit to be called")
		}
		if len(item) != 0 {
			t.Fatalf("expected empty item map on error")
		}
	})

	t.Run("walkFilesystem-abs-error", func(t *testing.T) {
		resetCmdState(t)
		absPath = func(string) (string, error) {
			return "", errors.New("abs fail")
		}
		exitCalled := false
		exitFn = func(int) {
			exitCalled = true
		}
		got := walkFilesystem(-1, "x", "")
		if !exitCalled {
			t.Fatalf("expected exit to be called")
		}
		if got == nil {
			t.Fatalf("expected fallback nil-id pointer")
		}
	})

	t.Run("walkFilesystem-readDir-error", func(t *testing.T) {
		resetCmdState(t)
		dir := t.TempDir()
		readDir = func(string) ([]os.DirEntry, error) {
			return nil, errors.New("readdir fail")
		}
		exitCalled := false
		exitFn = func(int) {
			exitCalled = true
		}
		got := walkFilesystem(-1, dir, "")
		if !exitCalled {
			t.Fatalf("expected exit to be called")
		}
		if got == nil {
			t.Fatalf("expected fallback nil-id pointer")
		}
	})
}

func TestWalkFilesystemLinkResolutionErrorAndSocket(t *testing.T) {
	t.Run("symlink-follow-failure", func(t *testing.T) {
		resetCmdState(t)
		dir := t.TempDir()
		target := filepath.Join(dir, "target.txt")
		if err := os.WriteFile(target, []byte("target"), 0o644); err != nil {
			t.Fatalf("write target: %v", err)
		}
		link := filepath.Join(dir, "link.txt")
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("create symlink: %v", err)
		}
		links_flag = true
		evalSymlinks = func(string) (string, error) {
			return "", errors.New("bad link")
		}
		got := walkFilesystem(-1, link, "")
		if got == nil || got.String() != nullId().String() {
			t.Fatalf("expected null id for bad symlink follow, got %v", got)
		}
	})

	t.Run("socket-file", func(t *testing.T) {
		resetCmdState(t)
		lstatFile = func(string) (os.FileInfo, error) {
			return fakeFileInfo{mode: os.ModeSocket}, nil
		}
		got := walkFilesystem(-1, "/tmp/socket", "")
		if got == nil || got.String() != nullId().String() {
			t.Fatalf("expected null id for socket, got %v", got)
		}
	})
}

func TestFileIDOpenFailure(t *testing.T) {
	resetCmdState(t)
	openFile = func(string) (*os.File, error) {
		return nil, errors.New("open failed")
	}
	exitCalled := false
	exitFn = func(int) {
		exitCalled = true
	}
	id := fileID("/bad/path")
	if !exitCalled {
		t.Fatalf("expected exit to be called")
	}
	if id == nil {
		t.Fatalf("expected fallback nil-id pointer")
	}
}

func TestOutputPathWithoutMetadata(t *testing.T) {
	resetCmdState(t)
	include_meta = false
	formatting_string = "path"
	item := map[string]interface{}{
		"c4id":   "abc123",
		"folder": false,
		"link":   false,
		"bytes":  int64(1),
	}
	out := captureOutput(func() {
		output("/tmp/example", item)
	})
	if !strings.Contains(out, "example:  abc123") {
		t.Fatalf("unexpected path formatting output: %q", out)
	}
}

func TestPrintIDTerminalBranch(t *testing.T) {
	resetCmdState(t)
	id := encode(strings.NewReader("terminal-branch"))
	isTerminal = func(int) bool { return true }
	out := captureOutput(func() {
		printID(id)
	})
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("expected newline for terminal output, got %q", out)
	}
}
