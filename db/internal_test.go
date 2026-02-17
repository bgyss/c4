package db

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	c4 "github.com/bgyss/c4/id"
	"go.etcd.io/bbolt"
)

func TestShuffle(t *testing.T) {
	r := rand.New(rand.NewPCG(42, 0))
	for j := 1; j < 1000; j++ {
		var list []string
		for i := 0; i < j; i++ {
			list = append(list, strconv.Itoa(i))
		}
		shuffleWithRand(list, r)
		count := 0
		for i := 0; i < j; i++ {
			if list[i] == strconv.Itoa(i) {
				count++
			}
		}
		if j > 12 && (float32(count)/float32(j)) > 0.31 {
			t.Errorf("shuffle ratio for %d: %.2f", j, (float32(count) / float32(j)))
		}
		if j == 10 {
			t.Logf("shuffle: %v", list)
		}
	}

}

func TestEntryMethods(t *testing.T) {
	stop := make(chan struct{})
	en := &entry{
		k:  []byte("source"),
		v:  []byte("target"),
		r:  []byte("relationship"),
		st: stop,
	}

	source := en.Source()
	target := en.Target()
	relationships := en.Relationships()

	en.k[0] = 'X'
	en.v[0] = 'Y'
	en.r[0] = 'Z'

	if string(source) != "source" {
		t.Fatalf("source was not copied")
	}
	if string(target) != "target" {
		t.Fatalf("target was not copied")
	}
	if len(relationships) != 1 || relationships[0] != "relationship" {
		t.Fatalf("relationship was not copied")
	}

	done := make(chan struct{})
	go func() {
		<-stop
		close(done)
	}()

	en.Stop()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("entry stop channel was not closed")
	}

	// nil stop channel branch
	enNil := &entry{}
	enNil.Stop()

	// sync.Once guarded stop branch
	stop2 := make(chan struct{})
	enOnce := &entry{
		st: stop2,
		so: new(sync.Once),
	}
	enOnce.Stop()
	enOnce.Stop() // second call should be a no-op and must not panic
}

func TestWriteFileData(t *testing.T) {
	tmp := t.TempDir()
	pathA := filepath.Join(tmp, "a")
	pathB := filepath.Join(tmp, "b")
	err := os.MkdirAll(pathA, 0700)
	if err != nil {
		t.Fatalf("error creating pathA: %q", err)
	}
	err = os.MkdirAll(pathB, 0700)
	if err != nil {
		t.Fatalf("error creating pathB: %q", err)
	}

	digest := c4.Identify(bytes.NewReader([]byte("write-file-data"))).Digest()
	data := []byte("write-file-data-payload")

	savedPath, err := write_file_data([]string{pathA, pathB}, digest, data)
	if err != nil {
		t.Fatalf("error writing file data: %q", err)
	}

	got, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("error reading written file: %q", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("written data mismatch")
	}

	againPath, err := write_file_data([]string{pathA, pathB}, digest, data)
	if err != nil {
		t.Fatalf("error checking existing file data: %q", err)
	}
	if againPath != savedPath {
		t.Fatalf("expected existing file path %q, got %q", savedPath, againPath)
	}

	_, err = write_file_data([]string{filepath.Join(tmp, "missing")}, digest, data)
	if err == nil {
		t.Fatalf("expected error for missing storage path")
	}

	// Path containing ".." anywhere in the cleaned path is skipped.
	dotPath := filepath.Join(tmp, "safe..name")
	err = os.MkdirAll(dotPath, 0700)
	if err != nil {
		t.Fatalf("error creating dotPath: %q", err)
	}
	skippedPath, err := write_file_data([]string{dotPath}, digest, data)
	if err != nil {
		t.Fatalf("unexpected error for dotPath: %q", err)
	}
	if skippedPath != "" {
		t.Fatalf("expected empty path when candidate path is skipped")
	}

	// Existing file path used as storage causes MkdirAll to fail with ENOTDIR.
	fileStorage := filepath.Join(tmp, "file-storage")
	err = os.WriteFile(fileStorage, []byte("x"), 0600)
	if err != nil {
		t.Fatalf("error creating file storage path: %q", err)
	}
	_, err = write_file_data([]string{fileStorage}, digest, data)
	if err == nil {
		t.Fatalf("expected error when storage path is a file")
	}

	// If the final filepath is already a directory, os.Create fails and
	// write_file_data eventually returns ("", nil).
	createFailBase := filepath.Join(tmp, "create-fail")
	err = os.MkdirAll(createFailBase, 0700)
	if err != nil {
		t.Fatalf("error creating createFailBase: %q", err)
	}
	filename := digest.ID().String()
	createFailTarget := filepath.Join(createFailBase, filename[0:2], filename[2:4], filename)
	err = os.MkdirAll(createFailTarget, 0700)
	if err != nil {
		t.Fatalf("error creating createFailTarget directory: %q", err)
	}
	createFailPath, err := write_file_data([]string{createFailBase}, digest, []byte("short"))
	if err != nil {
		t.Fatalf("unexpected error on create-fail path: %q", err)
	}
	if createFailPath != "" {
		t.Fatalf("expected empty path when os.Create fails for all candidates")
	}
}

func TestSecureRandAndShuffle(t *testing.T) {
	if secureRandIntN(0) != 0 {
		t.Fatalf("secureRandIntN(0) should be 0")
	}
	if secureRandIntN(-10) != 0 {
		t.Fatalf("secureRandIntN(-10) should be 0")
	}
	for i := 0; i < 100; i++ {
		v := secureRandIntN(7)
		if v < 0 || v >= 7 {
			t.Fatalf("secureRandIntN returned out-of-range value %d", v)
		}
	}

	list := []string{"a", "b", "c", "d", "e"}
	expected := append([]string(nil), list...)
	shuffle(list)
	sort.Strings(list)
	sort.Strings(expected)
	for i := range list {
		if list[i] != expected[i] {
			t.Fatalf("shuffle changed list members")
		}
	}
}

func TestTxErr(t *testing.T) {
	tx := &Tx{
		errCh: make(chan error, 1),
	}
	if tx.Err() != nil {
		t.Fatalf("expected nil error from empty channel")
	}

	expectedErr := errors.New("tx error")
	tx.errCh <- expectedErr
	if !errors.Is(tx.Err(), expectedErr) {
		t.Fatalf("expected error from channel")
	}
}

func TestOpenWindowsOptionsSeam(t *testing.T) {
	origGOOS := currentGOOS
	origContains := pathContains
	origBoltOpen := boltOpen
	defer func() {
		currentGOOS = origGOOS
		pathContains = origContains
		boltOpen = origBoltOpen
	}()

	currentGOOS = "windows"
	pathContains = func(string, string) bool { return true }

	var capturedNoSync bool
	var capturedNoGrowSync bool
	boltOpen = func(path string, mode os.FileMode, options *bbolt.Options) (*bbolt.DB, error) {
		capturedNoSync = options.NoSync
		capturedNoGrowSync = options.NoGrowSync
		return bbolt.Open(path, mode, options)
	}

	tmp := t.TempDir()
	databasePath := filepath.Join(tmp, "c4_tests_db")
	db, err := Open(databasePath, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	if !capturedNoSync || !capturedNoGrowSync {
		t.Fatalf("expected windows test options to enable NoSync/NoGrowSync")
	}
}

func TestWriteOptionsMarshalFailureSeam(t *testing.T) {
	origMarshal := jsonMarshal
	defer func() {
		jsonMarshal = origMarshal
	}()

	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, "db"), nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	jsonMarshal = func(v interface{}) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	// Should return early without panicking.
	db.write_options()
}

func TestDBErrorBranchesWithOversizedKeysAndBadTrees(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, "db"), nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	digest := c4.Identify(strings.NewReader("oversized-keys")).Digest()

	// Key too large for bbolt bucket put.
	tooLargeKey := strings.Repeat("k", 40000)
	if _, err := db.KeySet(tooLargeKey, digest); err == nil {
		t.Fatalf("expected KeySet error for oversized key")
	}

	// Key size that allows key bucket put but should fail on index bucket put
	// (digest bytes are appended to form the index key).
	indexTooLargeKey := strings.Repeat("k", 32710)
	if _, err := db.KeySet(indexTooLargeKey, digest); err == nil {
		t.Fatalf("expected KeySet index put error for oversized index key")
	}

	// Trigger KeyCAS put error branch: old_digest=nil matches missing key, then put fails.
	if replaced := db.KeyCAS(tooLargeKey, nil, digest); replaced {
		t.Fatalf("expected KeyCAS replacement to fail for oversized key")
	}

	// Trigger TreeSet marshal error branch via nil tree pointer.
	var nilTree *c4.Tree
	if err := db.TreeSet(nilTree); err == nil {
		t.Fatalf("expected TreeSet error for nil tree")
	}
}

func TestOpenReadOnlyBoltOpenSeam(t *testing.T) {
	origBoltOpen := boltOpen
	defer func() {
		boltOpen = origBoltOpen
	}()

	tmp := t.TempDir()
	dir := filepath.Join(tmp, "dbdir")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	seedFile := filepath.Join(dir, "db")
	seed, err := bbolt.Open(seedFile, 0o600, nil)
	if err != nil {
		t.Fatalf("seed open: %v", err)
	}
	_ = seed.Close()

	boltOpen = func(path string, mode os.FileMode, options *bbolt.Options) (*bbolt.DB, error) {
		ro := *options
		ro.ReadOnly = true
		return bbolt.Open(path, mode, &ro)
	}

	if _, err := Open(dir, nil); err == nil {
		t.Fatalf("expected Open failure when bolt is read-only")
	}
}

func TestKeyBatchErrorBranches(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, "db"), nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	digest := c4.Identify(strings.NewReader("batch-errors")).Digest()

	// Trigger key bucket put error branch in KeyBatch.
	db.KeyBatch(func(tx *Tx) bool {
		tx.KeySet(strings.Repeat("k", 40000), digest)
		return false
	})

	// Trigger index bucket put error branch in KeyBatch.
	db.KeyBatch(func(tx *Tx) bool {
		tx.KeySet(strings.Repeat("k", 32710), digest)
		return false
	})

	// Preload malformed oversized old digest bytes so old index delete path
	// can hit the delete error branch.
	err = db.db.Update(func(t *bbolt.Tx) error {
		b := t.Bucket(c4Bucket).Bucket(keyBucket)
		return b.Put([]byte("bad-old"), bytes.Repeat([]byte{0xAA}, 40000))
	})
	if err != nil {
		t.Fatalf("failed to seed malformed key value: %v", err)
	}

	db.KeyBatch(func(tx *Tx) bool {
		tx.KeySet("bad-old", digest)
		return false
	})

	// Allow asynchronous batch goroutine to flush.
	time.Sleep(20 * time.Millisecond)
}

func TestClosedDBUpdateErrorBranches(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, "db"), nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	source := c4.Identify(strings.NewReader("source")).Digest()
	target := c4.Identify(strings.NewReader("target")).Digest()
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	if _, err := db.KeyDeleteAll("prefix"); err == nil {
		t.Fatalf("expected KeyDeleteAll error on closed db")
	}
	if _, err := db.LinkDelete("rel", source, target); err == nil {
		t.Fatalf("expected LinkDelete error on closed db")
	}
	if _, err := db.LinkDeleteAll(source); err == nil {
		t.Fatalf("expected LinkDeleteAll error on closed db")
	}
}

func shuffleWithRand(list []string, r *rand.Rand) {
	l := len(list)
	for j, i := 0, 0; i < l; i++ {
		j = r.IntN(l)
		if j != i {
			list[i], list[j] = list[j], list[i]
		}
	}
}
