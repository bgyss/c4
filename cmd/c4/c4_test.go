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
	_ = w.Close()
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
	_ = os.WriteFile(path, []byte("bar"), 0644)
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
	_ = os.WriteFile(path, []byte("abc"), 0644)
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
	_ = os.Symlink(path, link)
	litem := newItem(link)
	if !litem["link"].(bool) {
		t.Fatal("expected link true")
	}
}

func TestOutputFormats(t *testing.T) {
	id := "testid"
	item := map[string]interface{}{"c4id": id, "folder": false, "link": false, "bytes": int64(3)}

	// Test with absolute flag = true (should output full path)
	include_meta = false
	absolute_flag = true
	formatting_string = "id"
	out := captureOutput(func() { output("/tmp/test", item) })
	if !strings.Contains(out, id+":")  && !strings.Contains(out, "/tmp/test") {
		t.Fatalf("absolute output missing expected content: %q", out)
	}

	// Test with metadata output
	include_meta = true
	formatting_string = "path"
	out = captureOutput(func() { output("/tmp/test", item) })
	if !strings.Contains(out, "\n  bytes:  3\n") {
		t.Fatalf("metadata output missing: %q", out)
	}
	
	// Test formatting_string = "id" with metadata
	formatting_string = "id"
	out = captureOutput(func() { output("/tmp/test", item) })
	if !strings.Contains(out, id+":") {
		t.Fatalf("id format metadata output missing: %q", out)
	}
}

func TestIdentifyFileAndFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(f1, []byte("a"), 0644)
	f2 := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(f2, []byte("b"), 0644)

	wd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(wd) }()

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
	_, _ = io.WriteString(w, "pipe")
	_ = w.Close()
	old := os.Stdin
	os.Stdin = r
	out := captureOutput(func() { identify_pipe() })
	os.Stdin = old
	id := c4.Identify(strings.NewReader("pipe"))
	if strings.TrimSpace(out) != id.String() {
		t.Fatalf("identify_pipe output %q", out)
	}
}

func TestMetadataOutput(t *testing.T) {
	dir := t.TempDir()
	id := "c41234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	
	wd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(wd) }()
	
	// Test regular file metadata output with path formatting
	item := map[string]interface{}{
		"c4id":   id,
		"folder": false,
		"link":   false,
		"bytes":  int64(100),
	}
	
	formatting_string = "path"
	out := captureOutput(func() { metadata_output(item, "test.txt", "test.txt", dir) })
	if !strings.Contains(out, "\"test.txt\":") {
		t.Fatalf("metadata_output missing path format: %q", out)
	}
	if !strings.Contains(out, "  folder:  false") {
		t.Fatalf("metadata_output missing folder info: %q", out)
	}
	if !strings.Contains(out, "  link:  false") {
		t.Fatalf("metadata_output missing link info: %q", out)
	}
	if !strings.Contains(out, "  bytes:  100") {
		t.Fatalf("metadata_output missing bytes info: %q", out)
	}
	
	// Test folder metadata output with id formatting
	folderItem := map[string]interface{}{
		"c4id":   id,
		"folder": true,
		"link":   false,
		"bytes":  int64(0),
	}
	
	formatting_string = "id"
	out = captureOutput(func() { metadata_output(folderItem, "testdir", "testdir", dir) })
	if !strings.Contains(out, id+":") {
		t.Fatalf("metadata_output missing id format: %q", out)
	}
	if !strings.Contains(out, "  folder:  true") {
		t.Fatalf("metadata_output missing folder true: %q", out)
	}
	
	// Test symlink metadata output
	linkItem := map[string]interface{}{
		"c4id":   id,
		"folder": false,
		"link":   "/some/target/path",
		"bytes":  int64(10),
	}
	
	absolute_flag = false
	out = captureOutput(func() { metadata_output(linkItem, "link.txt", "link.txt", dir) })
	if !strings.Contains(out, "  link:  ") {
		t.Fatalf("metadata_output missing link path: %q", out)
	}
	
	// Test symlink with absolute flag
	absolute_flag = true
	out = captureOutput(func() { metadata_output(linkItem, "link.txt", "link.txt", dir) })
	if !strings.Contains(out, "  link:  \"/some/target/path\"") {
		t.Fatalf("metadata_output missing absolute link path: %q", out)
	}
}

func TestWalkFilesystem(t *testing.T) {
	dir := t.TempDir()
	
	// Create test files and directories
	file1 := filepath.Join(dir, "file1.txt")
	_ = os.WriteFile(file1, []byte("content1"), 0644)
	
	subdir := filepath.Join(dir, "subdir")
	_ = os.Mkdir(subdir, 0755)
	
	file2 := filepath.Join(subdir, "file2.txt") 
	_ = os.WriteFile(file2, []byte("content2"), 0644)
	
	// Create a symlink
	link1 := filepath.Join(dir, "link1")
	_ = os.Symlink(file1, link1)
	
	wd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(wd) }()
	
	// Test regular file walking
	recursive_flag = false
	depth = -1
	links_flag = false
	
	id1 := walkFilesystem(-1, file1, "")
	expectedID1 := c4.Identify(strings.NewReader("content1"))
	if id1.String() != expectedID1.String() {
		t.Fatalf("walkFilesystem file ID mismatch: %s != %s", id1, expectedID1)
	}
	
	// Test directory walking
	out := captureOutput(func() {
		recursive_flag = true
		depth = 1
		walkFilesystem(1, subdir, "")
	})
	if !strings.Contains(out, "file2.txt") {
		t.Fatalf("walkFilesystem directory traversal missing file2.txt: %q", out)
	}
	
	// Test symlink handling with links_flag = false
	links_flag = false
	out = captureOutput(func() {
		recursive_flag = false
		depth = 0
		walkFilesystem(0, link1, "")
	})
	if !strings.Contains(out, "link") {
		t.Fatalf("walkFilesystem symlink handling failed: %q", out)
	}
	
	// Test symlink handling with links_flag = true
	links_flag = true
	id_link := walkFilesystem(-1, link1, "")
	// The symlink should resolve to the same content as the original file
	if id_link.String() != expectedID1.String() {
		t.Fatalf("walkFilesystem symlink resolution ID mismatch: %s != %s", id_link, expectedID1)
	}
}
