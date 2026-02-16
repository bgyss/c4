package id

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func makeTreeFromStrings(values ...string) *Tree {
	var digests DigestSlice
	for _, v := range values {
		digests.Insert(Identify(strings.NewReader(v)).Digest())
	}
	tree := NewTree(digests)
	tree.Compute()
	return tree
}

func TestIDBinaryAndTextEncoding(t *testing.T) {
	var nilID *ID
	if _, err := nilID.MarshalBinary(); err == nil {
		t.Fatalf("expected MarshalBinary nil receiver error")
	}
	if _, err := nilID.MarshalText(); err == nil {
		t.Fatalf("expected MarshalText nil receiver error")
	}

	src := Identify(strings.NewReader("binary-text-round-trip"))
	data, err := src.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary error: %v", err)
	}
	var fromBin ID
	if err := (&fromBin).UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary error: %v", err)
	}
	if fromBin.Cmp(src) != 0 {
		t.Fatalf("binary round-trip mismatch")
	}
	if err := (&fromBin).UnmarshalBinary(make([]byte, 65)); err == nil {
		t.Fatalf("expected UnmarshalBinary error for oversized digest")
	}

	text, err := src.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error: %v", err)
	}
	var fromText ID
	if err := (&fromText).UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText error: %v", err)
	}
	if fromText.Cmp(src) != 0 {
		t.Fatalf("text round-trip mismatch")
	}
	if err := (&fromText).UnmarshalText([]byte("not-an-id")); err == nil {
		t.Fatalf("expected UnmarshalText parse error")
	}
}

func TestIDGobEncodeDecode(t *testing.T) {
	src := Identify(strings.NewReader("gob-round-trip"))
	data, err := src.GobEncode()
	if err != nil {
		t.Fatalf("GobEncode error: %v", err)
	}
	var dst ID
	if err := (&dst).GobDecode(data); err != nil {
		t.Fatalf("GobDecode error: %v", err)
	}
	if dst.Cmp(src) != 0 {
		t.Fatalf("gob round-trip mismatch")
	}
}

func TestDigestReadWrite(t *testing.T) {
	d := Identify(strings.NewReader("digest-io")).Digest()

	short := make([]byte, 63)
	if n, err := d.Read(short); n != 0 || err != io.EOF {
		t.Fatalf("expected short Read EOF, got n=%d err=%v", n, err)
	}

	buf := make([]byte, 64)
	n, err := d.Read(buf)
	if err != nil {
		t.Fatalf("unexpected Read error: %v", err)
	}
	if n != 64 || !bytes.Equal(buf, d) {
		t.Fatalf("unexpected Read result")
	}

	target := NewDigest(make([]byte, 64))
	if n, err := target.Write(short); n != 0 || err != io.EOF {
		t.Fatalf("expected short Write EOF, got n=%d err=%v", n, err)
	}
	if n, err := target.Write(buf); err != nil || n != 64 {
		t.Fatalf("unexpected Write result n=%d err=%v", n, err)
	}
	if !bytes.Equal(target, d) {
		t.Fatalf("digest write mismatch")
	}

	if same := d.Sum(d); !bytes.Equal(same, d) {
		t.Fatalf("expected Sum(d,d) to return original digest")
	}
}

func TestTreeMarshalAndNodeHelpers(t *testing.T) {
	var nilTree *Tree
	if _, err := nilTree.MarshalBinary(); err == nil {
		t.Fatalf("expected nil tree MarshalBinary error")
	}

	rows, data := allocateTree(1)
	voidTree := &Tree{rows: rows, data: data}
	if _, err := voidTree.MarshalBinary(); err == nil {
		t.Fatalf("expected void tree MarshalBinary error")
	}

	tree := makeTreeFromStrings("a", "b", "c", "d")
	if tree.IDcount() != 4 {
		t.Fatalf("unexpected IDcount %d", tree.IDcount())
	}
	if tree.NodeCount() != tree.Length() {
		t.Fatalf("unexpected node count/length mismatch")
	}
	if tree.Size() != len(tree.data) {
		t.Fatalf("unexpected tree size")
	}
	if tree.Digest().ID().Cmp(tree.ID()) != 0 {
		t.Fatalf("digest/id mismatch")
	}
	if tree.At(0, 0).ID().Cmp(tree.ID()) != 0 {
		t.Fatalf("At returned wrong root digest")
	}

	text, err := tree.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText error: %v", err)
	}
	if string(text) != tree.String() {
		t.Fatalf("tree text mismatch")
	}

	bin, err := tree.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary error: %v", err)
	}
	var decoded Tree
	if err := (&decoded).UnmarshalBinary(bin); err != nil {
		t.Fatalf("UnmarshalBinary error: %v", err)
	}
	if decoded.ID().Cmp(tree.ID()) != 0 {
		t.Fatalf("decoded tree id mismatch")
	}

	node := tree.Node(0)
	if node.Parent().i != 0 {
		t.Fatalf("root parent should be root")
	}
	if node.Label().ID().Cmp(tree.ID()) != 0 {
		t.Fatalf("node label mismatch")
	}
	if node.Left() == nil || node.Right() == nil {
		t.Fatalf("expected root children")
	}
	if tree.Node(3).Parent().i != 1 {
		t.Fatalf("unexpected parent index")
	}

	single := makeTreeFromStrings("single")
	if single.Node(0).Left() != nil || single.Node(0).Right() != nil {
		t.Fatalf("single-node tree should have no children")
	}
}

func TestTreeUnmarshalInvalidData(t *testing.T) {
	var tree Tree
	if err := (&tree).UnmarshalBinary([]byte("short")); err == nil {
		t.Fatalf("expected short tree data error")
	}

	invalid := make([]byte, 64*3)
	copy(invalid[0:64], Identify(strings.NewReader("root")).Digest())
	copy(invalid[64:128], Identify(strings.NewReader("left")).Digest())
	copy(invalid[128:192], Identify(strings.NewReader("right")).Digest())
	if err := (&tree).UnmarshalBinary(invalid); err == nil {
		t.Fatalf("expected invalid tree root error")
	}
}

func TestSliceID(t *testing.T) {
	idA := Identify(strings.NewReader("slice-a"))
	idB := Identify(strings.NewReader("slice-b"))

	s := Slice{idA, idB}
	got := s.ID()
	if got == nil {
		t.Fatalf("expected non-nil slice id")
	}

	s = append(s, nil)
	if s.ID() != nil {
		t.Fatalf("expected nil slice id for nil entry")
	}
}

func TestInternalErrorsAndLog2(t *testing.T) {
	if (errNil{}).Error() != "unexpected nil id" {
		t.Fatalf("unexpected errNil value")
	}
	if (errInvalidTree{}).Error() != "invalid tree data" {
		t.Fatalf("unexpected errInvalidTree value")
	}
	if log2(0) != 0 {
		t.Fatalf("expected log2(0) == 0")
	}
	if log2(8) != 3 {
		t.Fatalf("expected log2(8) == 3")
	}
	if bitsLen64(1) != 1 {
		t.Fatalf("expected bitsLen64(1) == 1")
	}
	if bitsLen64(1<<40) != 41 {
		t.Fatalf("expected bitsLen64(1<<40) == 41")
	}
}

func TestDigestSliceBranches(t *testing.T) {
	var ds DigestSlice
	if ds.Insert(nil) != -1 {
		t.Fatalf("expected nil insert index -1")
	}
	if ds.Digest() != nil {
		t.Fatalf("expected nil digest for empty slice")
	}

	d := Identify(strings.NewReader("digest-slice")).Digest()
	if idx := ds.Insert(d); idx != 0 {
		t.Fatalf("expected insert index 0, got %d", idx)
	}
	if idx := ds.Insert(d); idx != -1 {
		t.Fatalf("expected duplicate insert index -1, got %d", idx)
	}

	if _, err := ds.Read(make([]byte, 1)); err == nil {
		t.Fatalf("expected Read buffer-too-small error")
	}
	if _, err := ds.Write([]byte{1, 2, 3}); err == nil {
		t.Fatalf("expected Write divisible-by-64 error")
	}

	odd := DigestSlice{
		Identify(strings.NewReader("a")).Digest(),
		Identify(strings.NewReader("b")).Digest(),
		Identify(strings.NewReader("c")).Digest(),
	}
	if odd.Digest() == nil {
		t.Fatalf("expected digest for odd-length digest slice")
	}
}

func TestSliceInsertIndexBranches(t *testing.T) {
	var s Slice
	if s.Index(nil) != -1 {
		t.Fatalf("expected index -1 for nil id")
	}
	s.Insert(nil)
	if len(s) != 0 {
		t.Fatalf("expected nil insert to be ignored")
	}

	id := Identify(strings.NewReader("slice-dup"))
	s.Insert(id)
	s.Insert(id)
	if len(s) != 1 {
		t.Fatalf("expected duplicate insert to be ignored")
	}
}
