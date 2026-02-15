package db

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	c4 "github.com/bgyss/c4/id"
	"go.etcd.io/bbolt"
)

func openTestDB(t *testing.T, opts *Options) (*DB, string) {
	t.Helper()
	base := t.TempDir()
	dbPath := filepath.Join(base, "dbdata")
	c4db, err := Open(dbPath, opts)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	t.Cleanup(func() {
		_ = c4db.Close()
	})
	return c4db, dbPath
}

func testTree(t *testing.T) (*c4.Tree, c4.Digest, []byte) {
	t.Helper()
	var digests c4.DigestSlice
	digests.Insert(c4.Identify(strings.NewReader("one")).Digest())
	digests.Insert(c4.Identify(strings.NewReader("two")).Digest())
	digests.Insert(c4.Identify(strings.NewReader("three")).Digest())
	tree := c4.NewTree(digests)
	digest := tree.Compute()
	data, err := tree.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary error: %v", err)
	}
	return tree, digest, data
}

func TestEntryAccessorsAndStop(t *testing.T) {
	src := c4.Identify(strings.NewReader("src")).Digest()
	dst := c4.Identify(strings.NewReader("dst")).Digest()
	e := &entry{
		k:  append([]byte(nil), src...),
		v:  append([]byte(nil), dst...),
		r:  []byte("rel"),
		st: make(chan struct{}),
		e:  errors.New("x"),
	}

	if e.Key() != string(src) {
		t.Fatalf("unexpected key value")
	}
	if e.Value().ID().Cmp(dst.ID()) != 0 {
		t.Fatalf("unexpected Value digest")
	}
	if e.Source().ID().Cmp(src.ID()) != 0 {
		t.Fatalf("unexpected Source digest")
	}
	if e.Target().ID().Cmp(dst.ID()) != 0 {
		t.Fatalf("unexpected Target digest")
	}
	if rel := e.Relationships(); len(rel) != 1 || rel[0] != "rel" {
		t.Fatalf("unexpected relationships: %v", rel)
	}
	if e.Err() == nil {
		t.Fatalf("expected non-nil error")
	}

	sourceCopy := e.Source()
	sourceCopy[0] ^= 0xff
	if bytes.Equal(sourceCopy, e.k) {
		t.Fatalf("Source should return a copy")
	}

	e.Stop()
	select {
	case <-e.st:
	default:
		t.Fatalf("expected stop channel to be closed")
	}
}

func TestDBManagedTxWrappers(t *testing.T) {
	c4db, _ := openTestDB(t, nil)

	if err := c4db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(c4Bucket).Bucket(optionsBucket).Put([]byte("k"), []byte("v"))
	}); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if err := c4db.View(func(tx *bbolt.Tx) error {
		got := tx.Bucket(c4Bucket).Bucket(optionsBucket).Get([]byte("k"))
		if string(got) != "v" {
			t.Fatalf("unexpected View value %q", string(got))
		}
		return nil
	}); err != nil {
		t.Fatalf("View error: %v", err)
	}

	if err := c4db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(c4Bucket).Bucket(optionsBucket).Put([]byte("k2"), []byte("v2"))
	}); err != nil {
		t.Fatalf("Batch error: %v", err)
	}
}

func TestWriteFileData(t *testing.T) {
	digest := c4.Identify(strings.NewReader("write-file-data")).Digest()
	data := []byte("payload")

	if _, err := write_file_data([]string{filepath.Join(t.TempDir(), "missing")}, digest, data); err == nil {
		t.Fatalf("expected error for missing storage path")
	}

	storageA := t.TempDir()
	storageB := t.TempDir()
	path, err := write_file_data([]string{storageA, storageB}, digest, data)
	if err != nil {
		t.Fatalf("write_file_data error: %v", err)
	}
	if path == "" {
		t.Fatalf("expected non-empty output path")
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if !bytes.Equal(written, data) {
		t.Fatalf("written data mismatch")
	}

	filename := digest.ID().String()
	preexistingDir := filepath.Join(storageA, filename[:2], filename[2:4])
	if err := os.MkdirAll(preexistingDir, 0o700); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	preexistingFile := filepath.Join(preexistingDir, filename)
	if err := os.WriteFile(preexistingFile, data, 0o600); err != nil {
		t.Fatalf("write preexisting file error: %v", err)
	}
	found, err := write_file_data([]string{storageA, storageB}, digest, data)
	if err != nil {
		t.Fatalf("write_file_data existing error: %v", err)
	}
	if found != preexistingFile {
		t.Fatalf("expected existing path %q, got %q", preexistingFile, found)
	}
}

func TestSecureRandIntNAndShuffle(t *testing.T) {
	if secureRandIntN(0) != 0 {
		t.Fatalf("expected 0 for non-positive n")
	}
	for i := 0; i < 100; i++ {
		v := secureRandIntN(7)
		if v < 0 || v >= 7 {
			t.Fatalf("out of range secureRandIntN value %d", v)
		}
	}

	values := []string{"a", "b", "c", "d", "e"}
	original := append([]string(nil), values...)
	shuffle(values)
	if len(values) != len(original) {
		t.Fatalf("shuffle changed length")
	}
	seen := make(map[string]int, len(values))
	for _, v := range values {
		seen[v]++
	}
	for _, v := range original {
		if seen[v] != 1 {
			t.Fatalf("shuffle lost or duplicated value %q", v)
		}
	}
}

func TestTreeGetExternalPathAndValidation(t *testing.T) {
	c4db, dbPath := openTestDB(t, nil)
	c4db.treeMaxSize = 1
	_, digest, data := testTree(t)

	// TreeGet reads external paths from treeBucket indirection:
	// digest -> pointerDigest (64 bytes), pointerDigest -> file path.
	pointer := c4.Identify(strings.NewReader("pointer")).Digest()
	validPath := filepath.Join(dbPath, "tree.bin")
	if err := os.WriteFile(validPath, data, 0o600); err != nil {
		t.Fatalf("failed to write test tree: %v", err)
	}

	if err := c4db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(c4Bucket).Bucket(treeBucket)
		if err := b.Put(digest, pointer); err != nil {
			return err
		}
		return b.Put(pointer, []byte(validPath))
	}); err != nil {
		t.Fatalf("failed to seed tree indirection: %v", err)
	}

	got, err := c4db.TreeGet(digest)
	if err != nil {
		t.Fatalf("TreeGet valid external path error: %v", err)
	}
	if got == nil || got.ID().Cmp(digest.ID()) != 0 {
		t.Fatalf("TreeGet returned wrong tree")
	}

	badPointer := c4.Identify(strings.NewReader("bad-pointer")).Digest()
	if err := c4db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(c4Bucket).Bucket(treeBucket)
		if err := b.Put(digest, badPointer); err != nil {
			return err
		}
		return b.Put(badPointer, []byte("../outside.bin"))
	}); err != nil {
		t.Fatalf("failed to seed traversal path: %v", err)
	}
	if _, err := c4db.TreeGet(digest); err == nil || !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("expected invalid path error, got %v", err)
	}

	outsidePath := filepath.Join(t.TempDir(), "outside.bin")
	if err := os.WriteFile(outsidePath, data, 0o600); err != nil {
		t.Fatalf("failed to write outside tree: %v", err)
	}
	if err := c4db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(c4Bucket).Bucket(treeBucket)
		if err := b.Put(digest, badPointer); err != nil {
			return err
		}
		return b.Put(badPointer, []byte(outsidePath))
	}); err != nil {
		t.Fatalf("failed to seed outside path: %v", err)
	}
	if _, err := c4db.TreeGet(digest); err == nil || !strings.Contains(err.Error(), "path traversal attack detected") {
		t.Fatalf("expected traversal detection error, got %v", err)
	}
}

func TestLinkGetAllBranches(t *testing.T) {
	c4db, _ := openTestDB(t, nil)
	src1 := c4.Identify(strings.NewReader("src1")).Digest()
	src2 := c4.Identify(strings.NewReader("src2")).Digest()
	tgt1 := c4.Identify(strings.NewReader("tgt1")).Digest()
	tgt2 := c4.Identify(strings.NewReader("tgt2")).Digest()

	if err := c4db.LinkSet("rel-a", src1, tgt1, tgt2); err != nil {
		t.Fatalf("LinkSet src1 error: %v", err)
	}
	if err := c4db.LinkSet("rel-b", src2, tgt1); err != nil {
		t.Fatalf("LinkSet src2 error: %v", err)
	}

	allCount := 0
	for e := range c4db.LinkGetAll() {
		if e.Err() != nil {
			t.Fatalf("LinkGetAll unexpected error: %v", e.Err())
		}
		if len(e.Relationships()) != 1 {
			t.Fatalf("expected exactly one relationship")
		}
		if e.Source() == nil || e.Target() == nil {
			t.Fatalf("expected source and target")
		}
		e.Close()
		allCount++
	}
	if allCount != 3 {
		t.Fatalf("expected 3 total links, got %d", allCount)
	}

	sourceCount := 0
	for e := range c4db.LinkGetAll(src1) {
		if e.Err() != nil {
			t.Fatalf("LinkGetAll(src1) unexpected error: %v", e.Err())
		}
		if e.Source().ID().Cmp(src1.ID()) != 0 {
			t.Fatalf("unexpected source in filtered results")
		}
		e.Close()
		sourceCount++
	}
	if sourceCount != 2 {
		t.Fatalf("expected 2 links for src1, got %d", sourceCount)
	}
}

func TestKeySetReplacementAndTxErr(t *testing.T) {
	c4db, _ := openTestDB(t, nil)
	key := "replace/me"
	first := c4.Identify(strings.NewReader("first")).Digest()
	second := c4.Identify(strings.NewReader("second")).Digest()

	prev, err := c4db.KeySet(key, first)
	if err != nil {
		t.Fatalf("KeySet first error: %v", err)
	}
	if prev != nil {
		t.Fatalf("expected nil previous value on first set")
	}

	prev, err = c4db.KeySet(key, second)
	if err != nil {
		t.Fatalf("KeySet second error: %v", err)
	}
	if prev == nil || prev.ID().Cmp(first.ID()) != 0 {
		t.Fatalf("expected previous value to match first digest")
	}

	if keys := c4db.KeyFind(first); len(keys) != 0 {
		t.Fatalf("expected old digest index entry to be removed, got %v", keys)
	}
	if keys := c4db.KeyFind(second); len(keys) != 1 || keys[0] != key {
		t.Fatalf("expected key under new digest index, got %v", keys)
	}

	tx := &Tx{errCh: make(chan error, 1)}
	if got := tx.Err(); got != nil {
		t.Fatalf("expected nil tx error, got %v", got)
	}
	expected := errors.New("batch error")
	tx.errCh <- expected
	if got := tx.Err(); got == nil || got.Error() != expected.Error() {
		t.Fatalf("unexpected tx error %v", got)
	}
}

func TestLinkDeleteAllBranches(t *testing.T) {
	c4db, _ := openTestDB(t, nil)
	srcA := c4.Identify(strings.NewReader("src-a")).Digest()
	srcB := c4.Identify(strings.NewReader("src-b")).Digest()
	target := c4.Identify(strings.NewReader("target")).Digest()

	if err := c4db.LinkSet("rel", srcA, target); err != nil {
		t.Fatalf("LinkSet srcA error: %v", err)
	}
	if err := c4db.LinkSet("rel", srcB, target); err != nil {
		t.Fatalf("LinkSet srcB error: %v", err)
	}

	n, err := c4db.LinkDeleteAll(srcA)
	if err != nil {
		t.Fatalf("LinkDeleteAll(srcA) error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 deleted for srcA, got %d", n)
	}

	n, err = c4db.LinkDeleteAll()
	if err != nil {
		t.Fatalf("LinkDeleteAll() error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 deleted in full cleanup, got %d", n)
	}
}

func TestTreeSetBranches(t *testing.T) {
	c4db, dbPath := openTestDB(t, nil)
	tree, digest, _ := testTree(t)

	if err := c4db.TreeSet(nil); err == nil {
		t.Fatalf("expected TreeSet(nil) error")
	}

	c4db.treeMaxSize = 0
	if err := c4db.TreeSet(tree); err != nil {
		t.Fatalf("TreeSet inline error: %v", err)
	}
	inlineTree, err := c4db.TreeGet(digest)
	if err != nil {
		t.Fatalf("TreeGet inline error: %v", err)
	}
	if inlineTree == nil || inlineTree.ID().Cmp(digest.ID()) != 0 {
		t.Fatalf("inline tree retrieval mismatch")
	}

	c4db.treeMaxSize = 1
	c4db.storage = []string{dbPath}
	if err := c4db.TreeSet(tree); err != nil {
		t.Fatalf("TreeSet external error: %v", err)
	}

	if err := c4db.View(func(tx *bbolt.Tx) error {
		tb := tx.Bucket(c4Bucket).Bucket(treeBucket)
		pb := tx.Bucket(c4Bucket).Bucket(pathBucket)
		treeValue := tb.Get(digest)
		if len(treeValue) != 64 {
			t.Fatalf("expected tree bucket indirection digest, got %d bytes", len(treeValue))
		}
		pathValue := pb.Get(digest)
		if len(pathValue) == 0 {
			t.Fatalf("expected external path value in path bucket")
		}
		return nil
	}); err != nil {
		t.Fatalf("view error: %v", err)
	}
}

func TestOpenWithSavedOptions(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "dbdata")

	created, err := Open(dbPath, nil)
	if err != nil {
		t.Fatalf("initial Open error: %v", err)
	}
	if err := created.Close(); err != nil {
		t.Fatalf("initial Close error: %v", err)
	}

	ext := t.TempDir()
	reopened, err := Open(dbPath, &Options{
		TreeMaxSize:   77,
		TreeStrategy:  TreeStrategyCache,
		ExternalStore: []string{ext},
	})
	if err != nil {
		t.Fatalf("reopen with options error: %v", err)
	}
	defer func() {
		_ = reopened.Close()
	}()

	if reopened.treeMaxSize != 77 {
		t.Fatalf("expected TreeMaxSize 77, got %d", reopened.treeMaxSize)
	}
	if reopened.treeStrategy != TreeStrategyCache {
		t.Fatalf("expected TreeStrategyCache, got %d", reopened.treeStrategy)
	}
	if len(reopened.storage) == 0 {
		t.Fatalf("expected at least one storage path")
	}
}

func TestDBBranchErrorCases(t *testing.T) {
	c4db, _ := openTestDB(t, nil)
	src := c4.Identify(strings.NewReader("src-err")).Digest()
	dst := c4.Identify(strings.NewReader("dst-err")).Digest()

	if err := c4db.LinkSet("rel", src); err == nil {
		t.Fatalf("expected LinkSet missing-targets error")
	}
	if n, err := c4db.LinkDelete("rel", src); err == nil || n != 0 {
		t.Fatalf("expected LinkDelete missing-targets error")
	}

	if got, err := c4db.KeyGet("missing"); err != nil || got != nil {
		t.Fatalf("expected nil KeyGet for missing key, got %v err=%v", got, err)
	}
	if got, err := c4db.KeyDelete("missing"); err != nil || got != nil {
		t.Fatalf("expected nil KeyDelete for missing key, got %v err=%v", got, err)
	}

	if tree, err := c4db.TreeGet(src); err != nil || tree != nil {
		t.Fatalf("expected nil tree for missing digest, got tree=%v err=%v", tree, err)
	}

	if err := c4db.LinkSet("rel", src, dst); err != nil {
		t.Fatalf("LinkSet error: %v", err)
	}
	gotCount := 0
	for e := range c4db.LinkGet("other-rel", src) {
		if e.Err() != nil {
			t.Fatalf("unexpected LinkGet entry error: %v", e.Err())
		}
		gotCount++
		e.Close()
	}
	if gotCount != 0 {
		t.Fatalf("expected no results for unmatched relationship, got %d", gotCount)
	}
}

func TestKeyGetAllWrongValueSize(t *testing.T) {
	c4db, _ := openTestDB(t, nil)
	if err := c4db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(c4Bucket).Bucket(keyBucket).Put([]byte("bad/key"), []byte("short"))
	}); err != nil {
		t.Fatalf("failed to seed malformed key entry: %v", err)
	}

	entries := 0
	for e := range c4db.KeyGetAll("bad/") {
		if e.Err() == nil {
			t.Fatalf("expected wrong value size error")
		}
		e.Close()
		entries++
	}
	if entries != 1 {
		t.Fatalf("expected exactly one malformed entry, got %d", entries)
	}
}

func TestWriteFileDataTraversalSkip(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error: %v", err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir error: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	digest := c4.Identify(strings.NewReader("traversal-skip")).Digest()
	path, err := write_file_data([]string{".."}, digest, []byte("payload"))
	if err != nil {
		t.Fatalf("write_file_data traversal skip error: %v", err)
	}
	if path != "" {
		t.Fatalf("expected empty path when all candidates are skipped, got %q", path)
	}
}
