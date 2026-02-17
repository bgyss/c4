package id

import (
	"bytes"
	"strings"
	"testing"
)

func TestSliceInsertAndIndexBranches(t *testing.T) {
	var s Slice
	if got := s.Index(nil); got != -1 {
		t.Fatalf("expected nil index -1, got %d", got)
	}

	s.Insert(nil)
	if len(s) != 0 {
		t.Fatalf("nil insert should not change slice")
	}

	idA := Identify(strings.NewReader("a"))
	idB := Identify(strings.NewReader("b"))
	s.Insert(idA)
	s.Insert(idA) // duplicate
	if len(s) != 1 {
		t.Fatalf("duplicate insert should not change length")
	}

	s.Insert(idB)
	if len(s) != 2 {
		t.Fatalf("expected second unique insert")
	}
}

func TestDigestSumEqualBranch(t *testing.T) {
	d := Identify(strings.NewReader("same")).Digest()
	if got := d.Sum(d); !bytes.Equal(got, d) {
		t.Fatalf("sum of equal digests should return same digest")
	}
}

func TestDigestSliceEdgeBranches(t *testing.T) {
	var ds DigestSlice
	if idx := ds.Insert(nil); idx != -1 {
		t.Fatalf("expected nil insert index -1, got %d", idx)
	}

	d := Identify(strings.NewReader("a")).Digest()
	if idx := ds.Insert(d); idx != 0 {
		t.Fatalf("unexpected first insert index %d", idx)
	}
	if idx := ds.Insert(d); idx != -1 {
		t.Fatalf("duplicate insert should return -1 for index 0 duplicate, got %d", idx)
	}

	if _, err := ds.Read(make([]byte, 1)); err == nil {
		t.Fatalf("expected read buffer-too-small error")
	}
	if _, err := ds.Write([]byte{0x01, 0x02, 0x03}); err == nil {
		t.Fatalf("expected write alignment error")
	}
}

func TestTreeUtilsBitCoverage(t *testing.T) {
	if got := log2(-1); got != 0 {
		t.Fatalf("expected log2(-1)=0, got %d", got)
	}
	if got := log2(0); got != 0 {
		t.Fatalf("expected log2(0)=0, got %d", got)
	}
	if got := log2(1); got != 0 {
		t.Fatalf("expected log2(1)=0, got %d", got)
	}
	if got := log2(1024); got != 10 {
		t.Fatalf("expected log2(1024)=10, got %d", got)
	}

	tests := []struct {
		in  uint64
		exp int
	}{
		{0, 0},
		{1, 1},
		{1 << 8, 9},
		{1 << 16, 17},
		{1 << 32, 33},
	}
	for _, tc := range tests {
		if got := bitsLen64(tc.in); got != tc.exp {
			t.Fatalf("bitsLen64(%d)=%d want %d", tc.in, got, tc.exp)
		}
	}
}
