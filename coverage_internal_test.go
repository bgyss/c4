package c4

import (
	"strings"
	"testing"
)

func TestInternalErrorsAndIDsID(t *testing.T) {
	if (errNil{}).Error() != "unexpected nil id" {
		t.Fatalf("unexpected errNil string")
	}
	if (errInvalidTree{}).Error() != "invalid tree data" {
		t.Fatalf("unexpected errInvalidTree string")
	}

	id1 := Identify(strings.NewReader("id-1"))
	id2 := Identify(strings.NewReader("id-2"))
	id3 := Identify(strings.NewReader("id-1")) // duplicate on purpose

	ids := IDs{id2, id1, id3}
	root := ids.ID()
	if root.IsNil() {
		t.Fatalf("expected non-nil root id")
	}
}
