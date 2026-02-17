package c4

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}

type secondReadErrReader struct {
	header []byte
	read   int
	err    error
}

func (r *secondReadErrReader) Read(p []byte) (int, error) {
	if r.read == 0 {
		r.read++
		n := copy(p, r.header)
		return n, nil
	}
	return 0, r.err
}

func TestRootInternalErrorAndParseBranches(t *testing.T) {
	if got := (errNil{}).Error(); got != "unexpected nil id" {
		t.Fatalf("unexpected errNil string %q", got)
	}

	if _, err := Parse(""); err == nil {
		t.Fatalf("expected parse error for empty input")
	}

	var id ID
	if err := id.UnmarshalJSON([]byte(`""`)); err != nil {
		t.Fatalf("unexpected empty-json unmarshal error: %v", err)
	}
	if !id.IsNil() {
		t.Fatalf("expected zero id after empty-json unmarshal")
	}
}

func TestReadTreeErrorAndListSizeEdgeBranches(t *testing.T) {
	readErr := errors.New("read-failed")
	if _, err := ReadTree(errReader{err: readErr}); !errors.Is(err, readErr) {
		t.Fatalf("expected reader error passthrough, got %v", err)
	}

	if got := listSize(0); got != 0 {
		t.Fatalf("expected listSize(0)=0, got %d", got)
	}
	if got := listSize(-10); got != 0 {
		t.Fatalf("expected listSize(-10)=0, got %d", got)
	}
	if got := listSize(2); got != 0 {
		t.Fatalf("expected listSize(2)=0 for invalid tree size, got %d", got)
	}
}

func TestReadTreeMidReadError(t *testing.T) {
	left := Identify(strings.NewReader("left"))
	right := Identify(strings.NewReader("right"))
	root := left.Sum(right)
	header := make([]byte, 192)
	copy(header[0:64], root[:])
	copy(header[64:128], left[:])
	copy(header[128:192], right[:])

	rr := &secondReadErrReader{
		header: header,
		err:    errors.New("mid-stream-read-error"),
	}
	_, err := ReadTree(rr)
	if err == nil || !strings.Contains(err.Error(), "mid-stream-read-error") {
		t.Fatalf("expected mid-read error propagation, got %v", err)
	}

	// Ensure header itself is valid.
	if _, err := ReadTree(bytes.NewReader(header)); err != nil {
		t.Fatalf("expected header-only tree to parse, got %v", err)
	}
}
