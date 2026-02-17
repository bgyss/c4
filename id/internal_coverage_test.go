package id

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestDigestReadWriteBranches(t *testing.T) {
	src := NewDigest(bytes.Repeat([]byte{0xAB}, 64))
	out := make([]byte, 64)
	n, err := src.Read(out)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if n != 64 || !bytes.Equal(out, src) {
		t.Fatalf("digest read mismatch")
	}

	short := make([]byte, 63)
	n, err = src.Read(short)
	if err != io.EOF || n != 0 {
		t.Fatalf("expected short read io.EOF, got n=%d err=%v", n, err)
	}

	dst := Digest(make([]byte, 64))
	n, err = dst.Write(bytes.Repeat([]byte{0xCD}, 64))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != 64 || !bytes.Equal(dst, bytes.Repeat([]byte{0xCD}, 64)) {
		t.Fatalf("digest write mismatch")
	}

	n, err = dst.Write(make([]byte, 10))
	if err != io.EOF || n != 0 {
		t.Fatalf("expected short write io.EOF, got n=%d err=%v", n, err)
	}
}

func TestIDEncodingBranches(t *testing.T) {
	var nilID *ID
	if _, err := nilID.MarshalBinary(); err == nil {
		t.Fatalf("expected nil marshal binary error")
	}
	if _, err := nilID.MarshalText(); err == nil {
		t.Fatalf("expected nil marshal text error")
	}

	id := Identify(strings.NewReader("encoding-branches"))
	data, err := id.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal binary: %v", err)
	}
	if len(data) != 64 {
		t.Fatalf("unexpected binary size %d", len(data))
	}

	var decoded ID
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatalf("unmarshal binary: %v", err)
	}
	if decoded.Cmp(id) != 0 {
		t.Fatalf("binary roundtrip mismatch")
	}

	if err := decoded.UnmarshalBinary(make([]byte, 65)); err == nil {
		t.Fatalf("expected unmarshal binary error for len > 64")
	}

	text, err := id.MarshalText()
	if err != nil {
		t.Fatalf("marshal text: %v", err)
	}
	var decodedText ID
	if err := decodedText.UnmarshalText(text); err != nil {
		t.Fatalf("unmarshal text: %v", err)
	}
	if decodedText.Cmp(id) != 0 {
		t.Fatalf("text roundtrip mismatch")
	}

	if err := decodedText.UnmarshalText([]byte("bad")); err == nil {
		t.Fatalf("expected parse error for bad text")
	}
}

func TestIDGobEncodeDecode(t *testing.T) {
	id := Identify(strings.NewReader("gob-branches"))
	data, err := id.GobEncode()
	if err != nil {
		t.Fatalf("gob encode: %v", err)
	}
	var decoded ID
	if err := decoded.GobDecode(data); err != nil {
		t.Fatalf("gob decode: %v", err)
	}
	if decoded.Cmp(id) != 0 {
		t.Fatalf("gob roundtrip mismatch")
	}

	if err := decoded.GobDecode([]byte{0x00, 0x01}); err == nil {
		t.Fatalf("expected gob decode error for invalid payload")
	}
}

func TestSliceIDBranches(t *testing.T) {
	var empty Slice
	id := empty.ID()
	if id == nil || id.Cmp(VOID_ID) != 0 {
		t.Fatalf("empty slice should return zero/void id")
	}

	withNil := Slice{Identify(strings.NewReader("a")), nil}
	if withNil.ID() != nil {
		t.Fatalf("slice containing nil entry should return nil")
	}

	var ids Slice
	ids.Insert(Identify(strings.NewReader("alpha")))
	ids.Insert(Identify(strings.NewReader("beta")))
	if ids.ID() == nil {
		t.Fatalf("expected non-nil id for populated slice")
	}
}

func TestTreeAndNodeMethods(t *testing.T) {
	var nilTree *Tree
	if _, err := nilTree.MarshalBinary(); err == nil {
		t.Fatalf("expected nil tree marshal error")
	}

	list := DigestSlice{
		Identify(strings.NewReader("a")).Digest(),
		Identify(strings.NewReader("b")).Digest(),
		Identify(strings.NewReader("c")).Digest(),
	}
	tree := NewTree(list)
	if _, err := tree.MarshalBinary(); err == nil {
		t.Fatalf("expected marshal error before compute")
	}

	root := tree.Compute()
	if root == nil {
		t.Fatalf("expected non-nil root digest")
	}

	data, err := tree.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal tree: %v", err)
	}

	var t2 Tree
	if err := t2.UnmarshalBinary(data[:100]); err == nil {
		t.Fatalf("expected invalid-tree error on short input")
	}
	corrupt := append([]byte{}, data...)
	corrupt[0] ^= 0xFF
	if err := t2.UnmarshalBinary(corrupt); err == nil {
		t.Fatalf("expected invalid-tree error on bad root")
	}
	if err := t2.UnmarshalBinary(data); err != nil {
		t.Fatalf("unmarshal tree: %v", err)
	}
	if text, err := t2.MarshalText(); err != nil || len(text) == 0 {
		t.Fatalf("marshal text failed: %v", err)
	}

	if t2.IDcount() != 3 {
		t.Fatalf("unexpected id count %d", t2.IDcount())
	}
	if t2.NodeCount() != t2.Length() {
		t.Fatalf("node/length mismatch")
	}
	if t2.Size() != len(data) {
		t.Fatalf("unexpected tree size")
	}
	if t2.String() == "" {
		t.Fatalf("expected non-empty tree string")
	}
	if t2.Digest().ID().Cmp(t2.ID()) != 0 {
		t.Fatalf("tree digest/id mismatch")
	}
	if len(t2.At(0, 0)) != 64 {
		t.Fatalf("unexpected digest size at root")
	}

	rootNode := t2.Node(0)
	if len(rootNode.Label()) != 64 {
		t.Fatalf("unexpected root node label size")
	}
	if len(rootNode.Left()) != 64 || len(rootNode.Right()) != 64 {
		t.Fatalf("expected root children digests")
	}

	leaf := Node{t: &t2, row: uint64(t2.RowCount() - 1), i: 0}
	if leaf.Left() != nil || leaf.Right() != nil {
		t.Fatalf("leaf node should not have children")
	}

	if p := (Node{t: &t2, i: 2}).Parent(); p.i != 0 {
		t.Fatalf("expected root parent for low indexes")
	}
	if p := (Node{t: &t2, i: 5}).Parent(); p.i != 2 {
		t.Fatalf("unexpected parent index %d", p.i)
	}
}

func TestInternalErrorStrings(t *testing.T) {
	if got := (errNil{}).Error(); got != "unexpected nil id" {
		t.Fatalf("unexpected errNil string %q", got)
	}
	if got := (errInvalidTree{}).Error(); got != "invalid tree data" {
		t.Fatalf("unexpected errInvalidTree string %q", got)
	}
}
