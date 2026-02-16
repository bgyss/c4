package c4

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type shortNoErrReader struct {
	data []byte
}

func (r *shortNoErrReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	return n, nil
}

type headThenErrReader struct {
	head []byte
	done bool
}

func (r *headThenErrReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		return copy(p, r.head), nil
	}
	return 0, errors.New("read-fail")
}

func TestReadTreeCoverage(t *testing.T) {
	// Read returning an error on first read.
	if _, err := ReadTree(bytes.NewReader([]byte("tiny"))); err == nil {
		t.Fatalf("expected initial read error")
	}

	// Short read without error should trigger invalid tree length.
	if _, err := ReadTree(&shortNoErrReader{data: make([]byte, 64)}); err == nil {
		t.Fatalf("expected invalid tree error for short read")
	}

	// Invalid head content should fail root validation.
	invalid := make([]byte, 3*64)
	invalid[0] = 1
	if _, err := ReadTree(bytes.NewReader(invalid)); err == nil {
		t.Fatalf("expected invalid root error")
	}

	ids := IDs{
		Identify(strings.NewReader("a")),
		Identify(strings.NewReader("b")),
		Identify(strings.NewReader("c")),
	}
	tree := NewTree(ids)
	tree.compute()
	data := tree.Bytes()

	decoded, err := ReadTree(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadTree valid data error: %v", err)
	}
	if decoded.ID().Cmp(tree.ID()) != 0 {
		t.Fatalf("decoded tree id mismatch")
	}

	// Error while reading the tail should be returned.
	if _, err := ReadTree(&headThenErrReader{head: data[:3*64]}); err == nil || err.Error() != "read-fail" {
		t.Fatalf("expected tail read error, got %v", err)
	}
}

func TestListSizeEdgeCases(t *testing.T) {
	if got := listSize(treeSize(1)); got != 0 {
		t.Fatalf("unexpected list size for single node: %d", got)
	}
	if got := listSize(treeSize(5)); got != 5 {
		t.Fatalf("unexpected list size for five nodes: %d", got)
	}
}

func TestTreeStringFromInvalidRoot(t *testing.T) {
	ids := IDs{
		Identify(strings.NewReader("x")),
		Identify(strings.NewReader("y")),
	}
	tree := NewTree(ids)
	// Force invalid root so String() and Bytes() trigger compute() path.
	for i := 0; i < 64 && i < len(tree); i++ {
		tree[i] = 0
	}
	if tree.valid() {
		t.Fatalf("expected tree to be invalid before recompute")
	}
	if len(tree.String()) == 0 {
		t.Fatalf("expected non-empty tree string")
	}
	if _, err := io.ReadAll(bytes.NewReader(tree.Bytes())); err != nil {
		t.Fatalf("unexpected bytes read error: %v", err)
	}
}
