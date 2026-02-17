package manifest

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bgyss/c4"
)

func mustID(t *testing.T, payload string) c4.ID {
	t.Helper()
	return c4.Identify(strings.NewReader(payload))
}

type basicFileInfo struct {
	name string
	size int64
	mode os.FileMode
	when time.Time
}

func (b basicFileInfo) Name() string       { return b.name }
func (b basicFileInfo) Size() int64        { return b.size }
func (b basicFileInfo) Mode() os.FileMode  { return b.mode }
func (b basicFileInfo) ModTime() time.Time { return b.when }
func (b basicFileInfo) IsDir() bool        { return b.mode.IsDir() }
func (b basicFileInfo) Sys() interface{}   { return nil }

func TestNilListMethodsCoverage(t *testing.T) {
	list := newNilList([]string{
		"a",
		"a/b",
		"a/b/c",
		"a/d",
		"z",
		"a/b", // duplicate should be removed by newNilList
	})
	if got := list.Len(); got != 5 {
		t.Fatalf("unexpected unique length %d", got)
	}

	first := list.Get(0)
	if first != "a" {
		t.Fatalf("unexpected first item %q", first)
	}

	idx := list.Find(fromSlash("a"))
	if idx != 0 {
		t.Fatalf("expected prefix index 0, got %d", idx)
	}

	sub := list.Sublist(fromSlash("a"))
	if sub.Len() != 4 {
		t.Fatalf("unexpected sublist length %d", sub.Len())
	}

	childrenList := newNilList([]string{
		"a/b",
		"a/b/c",
		"a/d",
		"z",
	})
	children := childrenList.Children(fromSlash("a"))
	gotChildren := children.StringSlice()
	if len(gotChildren) != 2 || gotChildren[0] != "a/b" || gotChildren[1] != "a/d" {
		t.Fatalf("unexpected children list: %#v", gotChildren)
	}

	// Cover branch where key itself exists and traversal stops immediately.
	if got := list.Children(fromSlash("a")).Len(); got != 0 {
		t.Fatalf("expected no children when key entry appears first, got %d", got)
	}

	// Cover branch where child extraction trims deeper descendants to direct child.
	deepChildren := newNilList([]string{"a/b/c", "a/b/d"})
	deep := deepChildren.Children(fromSlash("a"))
	if names := deep.StringSlice(); len(names) != 1 || names[0] != "a/b" {
		t.Fatalf("unexpected deep child extraction: %#v", names)
	}

	end := sub.End(append(fromSlash("a/b"), 0))
	if end == 0 {
		t.Fatalf("expected non-zero end for a/b prefix")
	}

	reversed := append(nilList{}, list...)
	reversed.Reverse()
	if reversed.Get(0) != "z" {
		t.Fatalf("expected reversed first element z, got %q", reversed.Get(0))
	}

	diff := Diff(newNilList([]string{"a", "x"}), newNilList([]string{"a", "y"}))
	gotDiff := diff.StringSlice()
	if len(gotDiff) == 0 {
		t.Fatalf("expected non-empty diff")
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for missing-key sublist")
			}
		}()
		_ = list.Sublist(fromSlash("not-found"))
	}()

	// Cover Sublist path where search index is exactly the end of the list.
	atEnd := list.Sublist(fromSlash("zzzz"))
	if atEnd.Len() != 0 {
		t.Fatalf("expected empty sublist at end, got %d", atEnd.Len())
	}
}

func TestFileInfoUnmarshalJsonBranches(t *testing.T) {
	validID := mustID(t, "id")
	validMeta := mustID(t, "meta")
	valid := `{"mode":"-rw-r--r--","mod_time":"2024-01-01T00:00:00Z","size":7,"name":"f.txt","id":"` + validID.String() + `","metadata":"` + validMeta.String() + `"}`

	var info FileInfo
	if err := info.UnmarshalJson([]byte(valid)); err != nil {
		t.Fatalf("unexpected valid unmarshal error: %v", err)
	}
	if info.Name() != "f.txt" || info.Size() != 7 {
		t.Fatalf("unexpected parsed info")
	}
	if info.ID() != validID || info.Metadata() != validMeta {
		t.Fatalf("unexpected id/metadata parse")
	}

	badMode := `{"mode":"x","mod_time":"2024-01-01T00:00:00Z","size":1,"name":"f"}`
	if err := info.UnmarshalJson([]byte(badMode)); err == nil {
		t.Fatalf("expected mode parse error")
	}

	badTime := `{"mode":"-rw-r--r--","mod_time":"bad","size":1,"name":"f"}`
	if err := info.UnmarshalJson([]byte(badTime)); err == nil {
		t.Fatalf("expected mod_time parse error")
	}

	invalidC4 := "c4" + strings.Repeat("0", 88) // invalid base58 char '0'
	badID := `{"mode":"-rw-r--r--","mod_time":"2024-01-01T00:00:00Z","size":1,"name":"f","id":"` + invalidC4 + `"}`
	if err := info.UnmarshalJson([]byte(badID)); err == nil {
		t.Fatalf("expected id parse error")
	}

	badMetadata := `{"mode":"-rw-r--r--","mod_time":"2024-01-01T00:00:00Z","size":1,"name":"f","metadata":"` + invalidC4 + `"}`
	if err := info.UnmarshalJson([]byte(badMetadata)); err == nil {
		t.Fatalf("expected metadata parse error")
	}
}

func TestParseFileInfoErrorBranches(t *testing.T) {
	validID := mustID(t, "parse-id")
	validLine := "-rw-r--r-- 10 2024-01-01T00:00:00Z file.txt " + validID.String()
	info, err := ParseFileInfo(validLine)
	if err != nil {
		t.Fatalf("unexpected valid parse error: %v", err)
	}
	if info.ID() != validID {
		t.Fatalf("parse id mismatch")
	}

	if _, err := ParseFileInfo("x 1 2024-01-01T00:00:00Z file.txt"); err == nil {
		t.Fatalf("expected mode parse error")
	}
	if _, err := ParseFileInfo("-rw-r--r-- x 2024-01-01T00:00:00Z file.txt"); err == nil {
		t.Fatalf("expected size parse error")
	}
	if _, err := ParseFileInfo("-rw-r--r-- 1 bad-time file.txt"); err == nil {
		t.Fatalf("expected time parse error")
	}

	invalidC4 := "c4" + strings.Repeat("0", 88)
	if _, err := ParseFileInfo("-rw-r--r-- 1 2024-01-01T00:00:00Z file.txt " + invalidC4); err == nil {
		t.Fatalf("expected id parse error")
	}
	if _, err := ParseFileInfo("-rw-r--r-- 1 2024-01-01T00:00:00Z file.txt " + validID.String() + " " + invalidC4); err == nil {
		t.Fatalf("expected metadata parse error")
	}
}

func TestManifestSettersAndMarshalBranches(t *testing.T) {
	m := NewManifest()
	id := mustID(t, "id")
	meta := mustID(t, "meta")
	info := MakeFileInfo(0o644, 3, time.Now().UTC(), "a.txt", id, meta)
	m.SetFileInfo("a.txt", info)

	serialized, err := m.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := string(serialized)
	if !strings.Contains(text, id.String()) || !strings.Contains(text, meta.String()) {
		t.Fatalf("expected id+metadata lines in marshal output")
	}

	// Parse a manifest without metadata to exercise the successful unmarshal path.
	mNoMeta := NewManifest()
	mNoMeta.SetFileInfo("a.txt", MakeFileInfo(0o644, 3, time.Now().UTC(), "a.txt", id, c4.ID{}))
	serializedNoMeta, err := mNoMeta.Marshal()
	if err != nil {
		t.Fatalf("marshal without metadata: %v", err)
	}

	var parsed M = make(map[string]*FileInfo)
	if err := parsed.Unmarshal(strings.NewReader(string(serializedNoMeta))); err != nil {
		t.Fatalf("unmarshal valid manifest: %v", err)
	}
	if parsed.Get("a.txt") == nil {
		t.Fatalf("expected parsed file entry")
	}

	if err := parsed.Unmarshal(strings.NewReader("-rw-r--r-- NaN 2024-01-01T00:00:00Z bad.txt\n")); err == nil {
		t.Fatalf("expected unmarshal parse error")
	}

	// Exercise SetFileInfo branch for non-*FileInfo values.
	m.SetFileInfo("from-os-info.txt", basicFileInfo{
		name: "from-os-info.txt",
		size: 11,
		mode: 0o644,
		when: time.Now().UTC(),
	})
	if m.Get("from-os-info.txt") == nil {
		t.Fatalf("expected SetFileInfo to convert os.FileInfo into *FileInfo")
	}
}

func TestManifestSetPanicBranches(t *testing.T) {
	m := NewManifest()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for SetId on missing path")
			}
		}()
		m.SetId("missing", mustID(t, "x"))
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic for SetMetadata on missing path")
			}
		}()
		m.SetMetadata("missing", mustID(t, "x"))
	}()
}

func TestParseFileModeMatrix(t *testing.T) {
	if _, err := ParseFileMode("too-short"); err == nil {
		t.Fatalf("expected too-short mode error")
	}

	tests := []struct {
		modeStr string
		expect  os.FileMode
	}{
		{"-rw-r--r--", 0o644},
		{"drwxr-xr-x", os.ModeDir | 0o755},
		{"drw-r--r--", os.ModeDevice | 0o644},
		{"Drw-r--r--", os.ModeDevice | 0o644},
		{"arw-r--r--", os.ModeAppend | 0o644},
		{"lrw-r--r--", os.ModeSymlink | 0o644},
		{"trw-r--r--", os.ModeTemporary | 0o644},
		{"prw-r--r--", os.ModeNamedPipe | 0o644},
		{"srw-r--r--", os.ModeSocket | 0o644},
		{"urw-r--r--", os.ModeSetuid | 0o644},
		{"grw-r--r--", os.ModeSetgid | 0o644},
		{"crw-r--r--", os.ModeDevice | os.ModeCharDevice | 0o644},
		{"brw-r--r--", os.ModeDevice | 0o644},
	}

	for _, tc := range tests {
		got, err := ParseFileMode(tc.modeStr)
		if err != nil {
			t.Fatalf("parse mode %q: %v", tc.modeStr, err)
		}
		if got != tc.expect {
			t.Fatalf("parse mode %q got %#o want %#o", tc.modeStr, got, tc.expect)
		}
	}
}

func TestNewFileInfoCopyAndOverrideBranches(t *testing.T) {
	id1 := mustID(t, "id1")
	id2 := mustID(t, "id2")
	meta := mustID(t, "meta")
	src := MakeFileInfo(0o644, 10, time.Now().UTC(), "f.txt", id1, meta)

	// copy branch from *FileInfo
	dst := NewFileInfo(src)
	if dst.ID() != id1 || dst.Metadata() != meta {
		t.Fatalf("expected copied id and metadata")
	}

	// explicit ids override branch
	override := NewFileInfo(src, id2, c4.ID{})
	if override.ID() != id2 {
		t.Fatalf("expected explicit id override")
	}
}
