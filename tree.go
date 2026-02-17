package c4

import (
	"bytes"
	"crypto/sha512"
	"io"
	"strings"
)

// `Tree` implements an ID tree as used for calculating IDs of non-contiguous
// sets of data. A C4 ID Tree is a type of merkle tree except that the list
// of IDs is sorted. According to the standard this is done to insure that
// two identical lists of IDs always resolve to the same ID.
type Tree []byte

// NewTree creates a new Tree from a DigestSlice, and copies the digests into
// the tree.  However, it does not compute the tree.
func NewTree(s []ID) Tree {
	size := 1
	for l := len(s); l > 1; l = (l + 1) / 2 {
		size += l
	}
	data := make([]byte, size*64)

	offset := len(data) - len(s)*64
	for i, id := range s {
		copy(data[offset+i*64:], id[:])
	}
	return Tree(data)
}

func ReadTree(r io.Reader) (Tree, error) {
	tree := make(Tree, 3*64)

	// If the first 192 bytes are not 3 valid digests this is not a tree.
	n, err := r.Read(tree)
	if err != nil {
		return nil, err
	}
	if n != len(tree) {
		return nil, errInvalidTree{}
	}
	head := make([]ID, 3)
	for i := range head {
		copy(head[i][:], tree[i*64:])
	}
	root := head[1].Sum(head[2])
	if root.Cmp(head[0]) != 0 {
		return nil, errInvalidTree{}
	}

	buffer := make([]byte, 4096)
	for err != io.EOF {
		n, err = r.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		tree = append(tree, buffer[:n]...)
	}

	// tree.rows = buildRows(tree)
	return tree, nil
}

// Compute resolves all Digests in the tree, and returns the root Digest
func (t Tree) compute() (id ID) {
	h := sha512.New()
	rows := buildRows(t)

	for i := len(rows) - 2; i >= 0; i-- {
		l := len(rows[i+1])
		for j := 0; j < l; j += 64 * 2 {
			jj := j / 2
			if j+64 >= l {
				copy(rows[i][jj:], rows[i+1][j:j+64])
			} else {
				h.Reset()
				if bytes.Compare(rows[i+1][j:j+64], rows[i+1][j+64:j+64*2]) == 1 {
					h.Write(rows[i+1][j+64 : j+64*2])
					h.Write(rows[i+1][j : j+64])
				} else {
					h.Write(rows[i+1][j : j+64*2])
				}
				copy(rows[i][jj:], h.Sum(nil)[0:64])
			}
		}
	}
	copy(id[:], t)
	return id
}

func (t Tree) String() string {
	var b strings.Builder
	var id ID
	if !t.valid() {
		t.compute()
	}

	for i := 0; i < len(t); i += 64 {
		copy(id[:], t[i:])
		b.WriteString(id.String())
	}
	return b.String()
}

func (t Tree) valid() bool {
	for _, b := range t[:64] {
		if b != 0 {
			return true
		}
	}
	return false
}

// Bytes returns the tree as a slice of bytes.
func (t Tree) Bytes() []byte {
	if !t.valid() {
		t.compute()
	}
	return t
}

// Number of IDs in the list (i.e. the length of the bottom row of the tree).
func (t Tree) Len() int {
	return listSize(len(t) / 64)
}

// The ID of the list (i.e. the level 0 of the tree).
func (t Tree) ID() (id ID) {
	if !t.valid() {
		return t.compute()
	}
	copy(id[:], t)
	return id
}

func buildRows(data []byte) [][]byte {
	length := len(data) / 64
	w := listSize(length)
	s := length - w

	r := 1
	for l := w; l > 1; l = (l + 1) / 2 {
		r++
	}
	rows := make([][]byte, r)

	i := len(rows) - 1
	for range rows {
		row := data[s*64 : (s+w)*64]
		rows[i] = row
		w = (w + 1) / 2
		s -= w
		i--
	}

	return rows
}

// listSize computes the length of the list represented by a
// tree given `total` number of branchs in the tree.
func listSize(total int) int {
	if total <= 0 {
		return 0
	}
	min := 1
	max := (total + 1) / 2
	if max < min {
		max = min
	}
	for min <= max {
		length := (min + max) / 2
		t := treeSize(length)
		if t == total {
			return length
		}
		if t < total {
			min = length + 1
			continue
		}
		max = length - 1
	}
	return 0
}

// treeSize computes the total number of branchs required to represent
// a list of length `l` elements.
func treeSize(l int) int {
	// Account for the root branch
	total := 1
	for ; l > 1; l = (l + 1) / 2 {
		total = total + l
	}
	return total
}

// Left
// r = row + 1
// i = i * 2

// Right
// r = row + 1
// i = (i+1)*2
