package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	c4 "github.com/Avalanche-io/c4/id"
)

func captureOutput(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestVersionString(t *testing.T) {
	s := versionString()
	if !strings.HasPrefix(s, "c4 version "+version_number) {
		t.Fatalf("unexpected version string %q", s)
	}
}

func TestEncodeAndNullID(t *testing.T) {
	data := "foo"
	id := encode(strings.NewReader(data))
	expected := c4.Identify(strings.NewReader(data))
	if id.String() != expected.String() {
		t.Fatalf("encode mismatch: %s != %s", id, expected)
	}

	n := nullId()
	empty := c4.Identify(strings.NewReader(""))
	if n.String() != empty.String() {
		t.Fatalf("nullId mismatch: %s != %s", n, empty)
	}
}

func TestFileID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	os.WriteFile(path, []byte("bar"), 0644)
	id := fileID(path)
	expected := c4.Identify(strings.NewReader("bar"))
	if id.String() != expected.String() {
		t.Fatalf("fileID mismatch: %s != %s", id, expected)
	}
}

func TestPrintID(t *testing.T) {
	id := c4.Identify(strings.NewReader("baz"))
	out := captureOutput(func() { printID(id) })
	if strings.TrimSpace(out) != id.String() {
		t.Fatalf("printID output %q", out)
	}
}

func TestNewItem(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	os.WriteFile(path, []byte("abc"), 0644)
	item := newItem(path)
	if item["folder"].(bool) {
		t.Fatal("expected file not folder")
	}
	if item["link"].(bool) {
		t.Fatal("expected not link")
	}
	if item["bytes"].(int64) != int64(len("abc")) {
		t.Fatalf("unexpected size %v", item["bytes"])
	}

	ditem := newItem(dir)
	if !ditem["folder"].(bool) {
		t.Fatal("expected folder true")
	}

	link := filepath.Join(dir, "lnk")
	os.Symlink(path, link)
	litem := newItem(link)
	if !litem["link"].(bool) {
		t.Fatal("expected link true")
	}
}

func TestOutputFormats(t *testing.T) {
	dir := t.TempDir()
	id := "testid"
	item := map[string]interface{}{"c4id": id, "folder": false, "link": false, "bytes": int64(3)}

	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	include_meta = false
	absolute_flag = false
	formatting_string = "id"
	out := captureOutput(func() { output(filepath.Join(dir, "file"), item) })
	if strings.TrimSpace(out) != id+":  file" {
		t.Fatalf("unexpected output %q", out)
	}

	include_meta = true
	formatting_string = "path"
	out = captureOutput(func() { output(filepath.Join(dir, "file"), item) })
	if !strings.Contains(out, "\n  bytes:  3\n") {
		t.Fatalf("metadata output missing: %q", out)
	}
}

func TestIdentifyFileAndFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	os.WriteFile(f1, []byte("a"), 0644)
	f2 := filepath.Join(dir, "b.txt")
	os.WriteFile(f2, []byte("b"), 0644)

	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)

	out1 := captureOutput(func() { identify_file("a.txt") })
	id1 := c4.Identify(strings.NewReader("a"))
	if strings.TrimSpace(out1) != id1.String() {
		t.Fatalf("identify_file wrong output %q", out1)
	}

	depth = 0
	absolute_flag = false
	include_meta = false
	formatting_string = "id"
	out := captureOutput(func() { identify_files([]string{"a.txt", "b.txt"}) })
	id2 := c4.Identify(strings.NewReader("b"))
	lines := strings.Split(strings.TrimSpace(out), "\n")
	expect1 := id1.String() + ":  a.txt"
	expect2 := id2.String() + ":  b.txt"
	if lines[0] != expect1 || lines[1] != expect2 {
		t.Fatalf("identify_files output unexpected: %q", out)
	}
}

func TestIdentifyPipe(t *testing.T) {
	r, w, _ := os.Pipe()
	io.WriteString(w, "pipe")
	w.Close()
	old := os.Stdin
	os.Stdin = r
	out := captureOutput(func() { identify_pipe() })
	os.Stdin = old
	id := c4.Identify(strings.NewReader("pipe"))
	if strings.TrimSpace(out) != id.String() {
		t.Fatalf("identify_pipe output %q", out)
	}
}
