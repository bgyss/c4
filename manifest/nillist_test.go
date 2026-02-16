package manifest

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNilListConversions(t *testing.T) {
	if got := string(fromSlash("a/b")); got != "a\x00b" {
		t.Fatalf("unexpected fromSlash result %q", got)
	}
	if got := toSlash([]byte("a\x00b")); got != "a/b" {
		t.Fatalf("unexpected toSlash result %q", got)
	}
	if got := fromSlash(""); len(got) != 0 {
		t.Fatalf("expected empty fromSlash result")
	}
	if got := toSlash(nil); got != "" {
		t.Fatalf("expected empty toSlash result")
	}
}

func TestNilListBasics(t *testing.T) {
	l := newNilList([]string{"b/c", "a", "b/a", "b/c"})
	if got := l.StringSlice(); !reflect.DeepEqual(got, []string{"a", "b/a", "b/c"}) {
		t.Fatalf("unexpected normalized list: %v", got)
	}
	if l.Get(1) != "b/a" {
		t.Fatalf("unexpected Get value %q", l.Get(1))
	}

	l.Reverse()
	if got := l.StringSlice(); !reflect.DeepEqual(got, []string{"b/c", "b/a", "a"}) {
		t.Fatalf("unexpected reversed list: %v", got)
	}
}

func TestNilListFindEndSublistChildren(t *testing.T) {
	l := newNilList([]string{
		"a",
		"a/b",
		"a/b/c",
		"a/d",
		"a/e/f",
		"ab",
		"z",
	})

	keyA := fromSlash("a")
	if got := l.Find(keyA); got != 0 {
		t.Fatalf("unexpected Find(a) %d", got)
	}
	if got := l.Find(fromSlash("a/c")); got <= 0 {
		t.Fatalf("unexpected insertion point for a/c: %d", got)
	}
	if got := l.End(keyA); got != 6 {
		t.Fatalf("unexpected End(a) %d", got)
	}

	sub := l.Sublist(keyA).StringSlice()
	if !reflect.DeepEqual(sub, []string{"a", "a/b", "a/b/c", "a/d", "a/e/f", "ab"}) {
		t.Fatalf("unexpected Sublist(a): %v", sub)
	}
	if got := l.Sublist(fromSlash("zz")); len(got) != 0 {
		t.Fatalf("expected empty sublist for missing key")
	}

	lChildren := newNilList([]string{"a/b", "a/b/c", "a/d", "a/e/f"})
	children := lChildren.Children(fromSlash("a")).StringSlice()
	if !reflect.DeepEqual(children, []string{"a/b", "a/d", "a/e"}) {
		t.Fatalf("unexpected children: %v", children)
	}
}

func TestNilListDiff(t *testing.T) {
	a := newNilList([]string{"a", "b", "c"})
	b := newNilList([]string{"b", "d"})
	diff := Diff(a, b).StringSlice()
	if !reflect.DeepEqual(diff, []string{"a", "c"}) {
		t.Fatalf("unexpected diff: %v", diff)
	}
}

func TestNilListOrdering(t *testing.T) {
	l := nilList{fromSlash("a/b"), fromSlash("a/c")}
	if !l.Less(0, 1) {
		t.Fatalf("expected Less(0,1) to be true")
	}
	l.Swap(0, 1)
	if !bytes.Equal(l[1], fromSlash("a/b")) {
		t.Fatalf("unexpected swap behavior")
	}
}
