package db_test

import (
	"bytes"
	"errors"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bgyss/c4/db"
	c4 "github.com/bgyss/c4/id"
	"go.etcd.io/bbolt"
)

func mkdbWithOptions(name string, options *db.Options, t *testing.T) (*db.DB, func() error, error) {
	dir, err := os.MkdirTemp("", "c4_tests")
	if err != nil {
		return nil, nil, err
	}
	t.Logf("temp folder created at %q", dir)
	tmpdb := filepath.Join(dir, name)
	db, err := db.Open(tmpdb, options)
	if err != nil {
		return nil, nil, err
	}

	return db, func() error {
		err := db.Close()
		if err != nil {
			return err
		}
		return os.RemoveAll(dir)
	}, nil
}

func mkdb(name string, t *testing.T) (*db.DB, func() error, error) {
	return mkdbWithOptions(name, nil, t)
}

func TestKeyApi(t *testing.T) {
	db_filename := "test.db"
	db, done, err := mkdb(db_filename, t)
	if err != nil {
		t.Errorf("error opening db at %q: %q", db_filename, err)
	}
	defer func() { _ = done() }()

	t.Run("Key Set, Get, Find, Delete", func(t *testing.T) {
		id := c4.Identify(strings.NewReader("foo"))
		id2 := c4.Identify(strings.NewReader("foo-updated"))
		key := "test/key/path"
		// Set
		old_id, err := db.KeySet(key, id.Digest())
		if err != nil {
			t.Errorf("error setting key: %q", err)
		}
		if old_id != nil {
			t.Errorf("setting an unset key should return a nil digest: %q", err)
		}
		t.Logf("Set %q: %q", key, id)

		// Update existing key and expect previous value back.
		previous_id, err := db.KeySet(key, id2.Digest())
		if err != nil {
			t.Errorf("error updating key: %q", err)
		}
		if previous_id == nil || previous_id.ID().Cmp(id) != 0 {
			t.Errorf("updating key should return previous digest")
		}

		// Get
		test_digest, err := db.KeyGet(key)
		if err != nil {
			t.Errorf("error getting key: %q", err)
		}
		if id2.Cmp(test_digest.ID()) != 0 {
			t.Errorf("values don't match expected %q, got %q", id2, test_digest.ID())
		}
		t.Logf("Get %q: %q", key, test_digest.ID())

		// Find
		for _, found_key := range db.KeyFind(id2.Digest()) {
			if found_key != key {
				t.Errorf("keys do not match expecting %q, got %q", key, found_key)
			}
		}

		// Delete
		deleted_digest, err := db.KeyDelete(key)
		if err != nil {
			t.Errorf("unable to delete key %q: %q", key, err)
		}
		if deleted_digest.ID().Cmp(id2) != 0 {
			t.Errorf("delete returned incorrect value expected %s, got %s", id2, deleted_digest.ID())
		}

		// Deleting a missing key should return nil digest and no error.
		deleted_digest, err = db.KeyDelete("test/key/missing")
		if err != nil {
			t.Errorf("unable to delete missing key: %q", err)
		}
		if deleted_digest != nil {
			t.Errorf("expected nil digest deleting missing key")
		}

	})

	var prefix string
	var sorted_keys []string
	t.Run("KeyGetAll", func(t *testing.T) {
		rng := rand.New(rand.NewPCG(42, 0))

		// Create a map of random keys, and a sorted slice of those keys
		// Use smaller test size in short mode for better Windows performance
		keyCount := 1000
		if testing.Short() {
			keyCount = 100
		}
		keys := make(map[string]c4.Digest)
		sorted_keys = make([]string, keyCount)
		var key string
		alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		prefix = path.Join("test", "prefix")
		for i := range sorted_keys {

			// Every 50th key we choose a random letter to put in the path to make
			// the sorting a little more interesting, and more representative of
			// actual use.
			if i%50 == 0 {
				key = path.Join(prefix, string(alphabet[rng.Int()%len(alphabet)]))
			}

			// Setting key, and value
			k := path.Join(key, strconv.Itoa(rng.Int()))
			v := randomDigest()
			sorted_keys[i] = k
			keys[k] = v
		}
		sort.Strings(sorted_keys)

		// Set the keys in the database.
		for k, v := range keys {
			_, err := db.KeySet(k, v)
			if err != nil {
				t.Errorf("error setting key: %q, %q", k, err)
			}
		}

		// Test KeyGetAll with empty key
		var count int
		for en := range db.KeyGetAll("") {
			if en.Err() != nil {
				t.Errorf("error in KeyGetAll %q", en.Err())
			}
			v := en.Value()
			k := en.Key()

			if keys[k].ID().Cmp(v.ID()) != 0 {
				t.Errorf("values don't match for key %q: %q, %q", k, keys[k].ID(), v.ID())
			}
			en.Close()
			count++
		}
		if count != len(sorted_keys) {
			t.Errorf("wrong number of keys returned %d of %d", count, len(sorted_keys))
		}

		// We pick an arbitrary key, and trim it to create a prefix
		// Use a valid index based on the actual array size
		prefixIndex := 115
		if prefixIndex >= len(sorted_keys) {
			prefixIndex = len(sorted_keys) / 2
		}
		prefix = path.Dir(sorted_keys[prefixIndex])
		count = 0

		// Build the expected list of keys for this prefix from our sorted list
		expected_keys_for_prefix := make([]string, 0)
		for _, k := range sorted_keys {
			if len(k) > len(prefix) && k[:len(prefix)] == prefix {
				expected_keys_for_prefix = append(expected_keys_for_prefix, k)
			}
		}

		// Keys returned by KeyGetAll should match the expected keys in order
		i := 0
		for en := range db.KeyGetAll(prefix) {
			if en.Err() != nil {
				t.Errorf("error in KeyGetAll %q", en.Err())
			}
			if i >= len(expected_keys_for_prefix) {
				t.Errorf("KeyGetAll returned more keys than expected")
				en.Close()
				count++
				continue
			}
			if expected_keys_for_prefix[i] != en.Key() {
				t.Errorf("keys not equal %q, %q", expected_keys_for_prefix[i], en.Key())
			}
			en.Close()
			count++
			i++
		}
		if count != len(expected_keys_for_prefix) {
			t.Errorf("wrong number of keys returned in prefix search, got %d expected %d", count, len(expected_keys_for_prefix))
		}

	})

	t.Run("KeyDeleteAll", func(t *testing.T) {

		// Get the expected count for this prefix by counting matching keys
		expected_prefix_count := 0
		for _, k := range sorted_keys {
			if len(k) > len(prefix) && k[:len(prefix)] == prefix {
				expected_prefix_count++
			}
		}

		n, err := db.KeyDeleteAll(prefix)
		if err != nil {
			t.Errorf("unable to delete all entries with %q prefix %s", prefix, err)
		}
		if n != expected_prefix_count {
			t.Errorf("unable to delete all entries with %q prefix, expected %d, got %d", prefix, expected_prefix_count, n)
		}

		expected_remaining := len(sorted_keys) - expected_prefix_count
		n, err = db.KeyDeleteAll()
		if err != nil {
			t.Errorf("unable to delete all entries, %q", err)
		}
		if n != expected_remaining {
			t.Errorf("unable to delete all entries, expected %d, got %d", expected_remaining, n)
		}

		i := 0
		for en := range db.KeyGetAll("") {
			if en.Err() != nil {
				t.Errorf("error in KeyGetAll %q", en.Err())
			}
			i++
			en.Close()
		}
		if i != 0 {
			t.Errorf("not all entries delete %d remain", i)
		}
	})

	t.Run("KeyCAS", func(t *testing.T) {
		id_foo := c4.Identify(strings.NewReader("foo"))
		id_bar := c4.Identify(strings.NewReader("bar"))
		id_bat := c4.Identify(strings.NewReader("bat"))
		key := "test compare and swap"

		// Set initial value
		_, err := db.KeySet(key, id_foo.Digest())
		if err != nil {
			t.Errorf("error setting key: %q", err)
		}

		// Test the positive case
		if !db.KeyCAS(key, id_foo.Digest(), id_bar.Digest()) {
			t.Errorf("compare and swap operation failed on valid compare")
		}

		// Expecting fail since the key should now be set to id_bar, not id_foo
		if db.KeyCAS(key, id_foo.Digest(), id_bat.Digest()) {
			t.Errorf("compare and swap operation succeeded on invalid compare")
		}

		// Test nil handling
		if !db.KeyCAS(key, id_bar.Digest(), nil) {
			t.Errorf("compare and swap operation failed on valid compare")
		}

		// Sets key only if existing value is nil
		if !db.KeyCAS(key, nil, id_bat.Digest()) {
			t.Errorf("compare and swap operation failed on valid compare")
		}

	})

	// db.KeyClock(key) uint64

}

func TestLinkApi(t *testing.T) {
	db_filename := "test.db"
	db, done, err := mkdb(db_filename, t)
	if err != nil {
		t.Errorf("error opening db at %q: %q", db_filename, err)
	}
	defer func() { _ = done() }()

	_ = db

	t.Run("Link Set, Get, Delete", func(t *testing.T) {
		foo_id := c4.Identify(strings.NewReader("foo"))
		fooAttr_id := c4.Identify(strings.NewReader("attributes of foo"))
		fooMeta_id := c4.Identify(strings.NewReader("metadata of foo"))

		// Set
		err := db.LinkSet("attributes", foo_id.Digest(), fooAttr_id.Digest())
		if err != nil {
			t.Errorf("error setting link: %q", err)
		}
		err = db.LinkSet("metadata", foo_id.Digest(), fooMeta_id.Digest())
		if err != nil {
			t.Errorf("error setting link: %q", err)
		}

		// Get
		count := 0

		for en := range db.LinkGet("attributes", foo_id.Digest()) {
			if en.Err() != nil {
				t.Errorf("error getting link: %q", en.Err())
				en.Stop()
				en.Close()
				break
			}
			if foo_id.Cmp(en.Source().ID()) != 0 {
				t.Errorf("error getting source digest")
			}
			if fooAttr_id.Cmp(en.Target().ID()) != 0 {
				t.Errorf("error getting link")
			}
			relationships := en.Relationships()
			if len(relationships) != 1 || relationships[0] != "attributes" {
				t.Errorf("error getting relationship")
			}
			en.Close()
			count++
		}
		if count != 1 {
			t.Errorf("incorrect link count %d", count)
		}

		// Delete
		n, err := db.LinkDelete("attributes", foo_id.Digest(), fooAttr_id.Digest())
		if err != nil {
			t.Errorf("error failed to delete link %q", err)
		}
		if n != 1 {
			t.Errorf("failed to delete link")
		}
		n, err = db.LinkDelete("metadata", foo_id.Digest(), fooMeta_id.Digest())
		if err != nil {
			t.Errorf("error failed to delete link %q", err)
		}
		if n != 1 {
			t.Errorf("failed to delete metadata link")
		}

		err = db.LinkSet("attributes", foo_id.Digest())
		if err == nil {
			t.Errorf("expected missing targets error from LinkSet")
		}
		n, err = db.LinkDelete("attributes", foo_id.Digest())
		if err == nil || n != 0 {
			t.Errorf("expected missing targets error from LinkDelete")
		}

	})

	var delete_digest c4.Digest
	t.Run("LinkGetAll", func(t *testing.T) {
		sourceA := c4.Identify(strings.NewReader("sourceA")).Digest()
		sourceB := c4.Identify(strings.NewReader("sourceB")).Digest()
		delete_digest = sourceA

		targetA1 := c4.Identify(strings.NewReader("sourceA-metadata-1")).Digest()
		targetA2 := c4.Identify(strings.NewReader("sourceA-metadata-2")).Digest()
		targetA3 := c4.Identify(strings.NewReader("sourceA-parent-1")).Digest()
		targetB1 := c4.Identify(strings.NewReader("sourceB-metadata-1")).Digest()

		err := db.LinkSet("metadata", sourceA, targetA1, targetA2)
		if err != nil {
			t.Errorf("error setting link: %q", err)
		}
		err = db.LinkSet("parent", sourceA, targetA3)
		if err != nil {
			t.Errorf("error setting link: %q", err)
		}
		err = db.LinkSet("metadata", sourceB, targetB1)
		if err != nil {
			t.Errorf("error setting link: %q", err)
		}

		relationship_counts := map[string]int{
			"metadata": 0,
			"parent":   0,
		}
		count := 0
		for en := range db.LinkGetAll(sourceA) {
			if en.Err() != nil {
				t.Errorf("error in LinkGetAll %q", en.Err())
			}
			if sourceA.ID().Cmp(en.Source().ID()) != 0 {
				t.Errorf("LinkGetAll wrong source %q, got %q", sourceA.ID(), en.Source().ID())
			}
			relationship := en.Relationships()[0]
			relationship_counts[relationship]++
			en.Close()
			count++
		}
		if count != 3 {
			t.Errorf("failed to return all links expected %d, got %d", 3, count)
		}
		if relationship_counts["metadata"] != 2 || relationship_counts["parent"] != 1 {
			t.Errorf("incorrect relationship counts for sourceA: metadata=%d, parent=%d", relationship_counts["metadata"], relationship_counts["parent"])
		}

		count = 0
		for en := range db.LinkGetAll(sourceA, sourceB) {
			if en.Err() != nil {
				t.Errorf("error in LinkGetAll %q", en.Err())
			}
			source := en.Source().ID()
			if source.Cmp(sourceA.ID()) != 0 && source.Cmp(sourceB.ID()) != 0 {
				t.Errorf("unexpected source from LinkGetAll: %q", source)
			}
			en.Close()
			count++
		}
		if count != 4 {
			t.Errorf("failed to return all links expected %d, got %d", 4, count)
		}

		count = 0
		for en := range db.LinkGetAll() {
			if en.Err() != nil {
				t.Errorf("error in LinkGetAll %q", en.Err())
			}
			en.Close()
			count++
		}
		if count != 4 {
			t.Errorf("failed to return all links expected %d, got %d", 4, count)
		}

	})

	t.Run("LinkDeleteAll", func(t *testing.T) {
		// Count links for delete_digest before deletion
		expected_delete_digest_count := 0
		for en := range db.LinkGetAll(delete_digest) {
			if en.Err() != nil {
				t.Errorf("error counting links: %q", en.Err())
				en.Close()
				break
			}
			expected_delete_digest_count++
			en.Close()
		}

		// Count total links before deletion
		expected_total_count := 0
		for en := range db.LinkGetAll() {
			if en.Err() != nil {
				t.Errorf("error counting total links: %q", en.Err())
				en.Close()
				break
			}
			expected_total_count++
			en.Close()
		}

		// db.LinkDeleteAll(id.Digest) (int, error)
		n, err := db.LinkDeleteAll(delete_digest)
		if err != nil {
			t.Errorf("unable to delete all entries %s", err)
		}
		if n != expected_delete_digest_count {
			t.Errorf("unable to delete all entries, expected %d, got %d", expected_delete_digest_count, n)
		}

		expected_remaining := expected_total_count - expected_delete_digest_count
		n, err = db.LinkDeleteAll()
		if err != nil {
			t.Errorf("unable to delete all entries, %q", err)
		}
		if n != expected_remaining {
			t.Errorf("unable to delete all entries, expected %d, got %d", expected_remaining, n)
		}
		st := db.Stats()
		t.Logf("Stats Trees:%d, Keys:%d, Indexes: %d, Links:%d, TreesSize:%d(%d)\n", st.Trees, st.Keys, st.KeyIndexes, st.Links, st.TreesSize, st.TreesSize/64)

		i := 0
		for en := range db.LinkGetAll() {
			if en.Err() != nil {
				t.Errorf("error in KeyGetAll %q", en.Err())
			}
			i++
			en.Close()
		}
		if i != 0 {
			t.Errorf("not all entries delete %d remain", i)
		}
	})

}

func TestTreeApi(t *testing.T) {
	db_filename := "test.db"
	db, done, err := mkdb(db_filename, t)
	if err != nil {
		t.Errorf("error opening db at %q: %q", db_filename, err)
	}
	defer func() { _ = done() }()

	_ = db

	t.Run("Tree Set, Get, Delete", func(t *testing.T) {
		// Create a tree
		var digests c4.DigestSlice
		for i := 0; i < 100; i++ {
			digests.Insert(randomDigest())
		}
		tree := c4.NewTree(digests)
		tree_digest := tree.Compute()

		err := db.TreeSet(tree)
		if err != nil {
			t.Errorf("error setting tree %q", err)
		}

		tree2, err := db.TreeGet(tree_digest)
		if err != nil {
			t.Errorf("error getting tree %q", err)
		}

		if tree2 == nil || tree2.ID().Cmp(tree_digest.ID()) != 0 {
			id_str := "<nil>"
			if tree2 != nil {
				id_str = tree2.ID().String()
			}
			t.Errorf("error tree ids don't match expected %q, got %q", tree_digest.ID(), id_str)
		}

		st := db.Stats()
		if st.Trees != 1 || st.TreesSize/64 != 202 {
			t.Errorf("error tree has incorrect stats before delete")
		}
		t.Logf("Stats Trees:%d, Keys:%d, Indexes: %d, Links:%d, TreesSize:%d(%d)\n", st.Trees, st.Keys, st.KeyIndexes, st.Links, st.TreesSize, st.TreesSize/64)
		err = db.TreeDelete(tree_digest)
		if err != nil {
			t.Errorf("failed to delete tree")
		}
		st = db.Stats()
		if st.Trees != 0 || st.TreesSize != 0 {
			t.Errorf("error tree has incorrect stats after delete")
			t.Errorf("Stats Trees:%d, Keys:%d, Indexes: %d, Links:%d, TreesSize:%d(%d)\n", st.Trees, st.Keys, st.KeyIndexes, st.Links, st.TreesSize, st.TreesSize/64)
		}
	})

}

func TestTreeExternalStoreAndPersistedOptions(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "test.db")
	storagePath := filepath.Join(root, "store")
	err := os.MkdirAll(storagePath, 0700)
	if err != nil {
		t.Fatalf("error creating storage path: %q", err)
	}

	opts := &db.Options{
		TreeMaxSize:   1,
		ExternalStore: []string{storagePath},
	}

	c4db, err := db.Open(dbPath, opts)
	if err != nil {
		t.Fatalf("error opening db at %q: %q", dbPath, err)
	}

	var digests c4.DigestSlice
	for _, value := range []string{"alpha", "bravo", "charlie", "delta"} {
		digests.Insert(c4.Identify(strings.NewReader(value)).Digest())
	}
	tree := c4.NewTree(digests)
	treeDigest := tree.Compute()

	err = c4db.TreeSet(tree)
	if err != nil {
		t.Fatalf("error setting tree with external storage: %q", err)
	}

	matches, err := filepath.Glob(filepath.Join(storagePath, "*", "*", treeDigest.ID().String()))
	if err != nil {
		t.Fatalf("error listing stored files: %q", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one external tree file, got %d", len(matches))
	}

	tree2, err := c4db.TreeGet(treeDigest)
	if err != nil {
		t.Fatalf("error getting externally stored tree: %q", err)
	}
	if tree2 == nil || tree2.ID().Cmp(treeDigest.ID()) != 0 {
		t.Fatalf("unexpected tree returned from TreeGet")
	}

	err = c4db.Close()
	if err != nil {
		t.Fatalf("error closing db: %q", err)
	}

	c4db, err = db.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("error reopening db at %q: %q", dbPath, err)
	}
	defer func() { _ = c4db.Close() }()

	tree3, err := c4db.TreeGet(treeDigest)
	if err != nil {
		t.Fatalf("error getting externally stored tree after reopen: %q", err)
	}
	if tree3 == nil || tree3.ID().Cmp(treeDigest.ID()) != 0 {
		t.Fatalf("unexpected tree returned from TreeGet after reopen")
	}
}

func TestTreeGetPathValidation(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "test.db")
	storagePath := filepath.Join(root, "store")
	err := os.MkdirAll(storagePath, 0700)
	if err != nil {
		t.Fatalf("error creating storage path: %q", err)
	}

	c4db, err := db.Open(dbPath, &db.Options{
		TreeMaxSize:   1,
		ExternalStore: []string{storagePath},
	})
	if err != nil {
		t.Fatalf("error opening db at %q: %q", dbPath, err)
	}
	defer func() { _ = c4db.Close() }()

	var digests c4.DigestSlice
	for _, value := range []string{"one", "two", "three"} {
		digests.Insert(c4.Identify(strings.NewReader(value)).Digest())
	}
	tree := c4.NewTree(digests)
	treeDigest := tree.Compute()

	err = c4db.TreeSet(tree)
	if err != nil {
		t.Fatalf("error setting tree with external storage: %q", err)
	}

	err = c4db.Update(func(tx *bbolt.Tx) error {
		rootBucket := tx.Bucket([]byte("c4"))
		treeBucket := rootBucket.Bucket([]byte("tree"))
		pathBucket := rootBucket.Bucket([]byte("path"))
		if err := treeBucket.Put(treeDigest, treeDigest); err != nil {
			return err
		}
		return pathBucket.Put(treeDigest, []byte(filepath.Join("..", "escape")))
	})
	if err != nil {
		t.Fatalf("error injecting invalid path: %q", err)
	}

	_, err = c4db.TreeGet(treeDigest)
	if err == nil || !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("expected invalid path error, got %v", err)
	}

	outsideFile := filepath.Join(root, "outside", treeDigest.ID().String())
	err = os.MkdirAll(filepath.Dir(outsideFile), 0700)
	if err != nil {
		t.Fatalf("error creating outside dir: %q", err)
	}
	err = os.WriteFile(outsideFile, []byte("outside"), 0600)
	if err != nil {
		t.Fatalf("error writing outside file: %q", err)
	}

	err = c4db.Update(func(tx *bbolt.Tx) error {
		rootBucket := tx.Bucket([]byte("c4"))
		pathBucket := rootBucket.Bucket([]byte("path"))
		return pathBucket.Put(treeDigest, []byte(outsideFile))
	})
	if err != nil {
		t.Fatalf("error setting outside path: %q", err)
	}

	_, err = c4db.TreeGet(treeDigest)
	if err == nil || !strings.Contains(err.Error(), "path traversal attack detected") {
		t.Fatalf("expected path traversal attack error, got %v", err)
	}

	missingPath := filepath.Join(storagePath, "missing", treeDigest.ID().String())
	err = c4db.Update(func(tx *bbolt.Tx) error {
		rootBucket := tx.Bucket([]byte("c4"))
		pathBucket := rootBucket.Bucket([]byte("path"))
		return pathBucket.Put(treeDigest, []byte(missingPath))
	})
	if err != nil {
		t.Fatalf("error setting missing file path: %q", err)
	}

	_, err = c4db.TreeGet(treeDigest)
	if err == nil {
		t.Fatalf("expected missing file error")
	}
}

func TestDBTransactionWrappers(t *testing.T) {
	db_filename := "test.db"
	c4db, done, err := mkdb(db_filename, t)
	if err != nil {
		t.Fatalf("error opening db at %q: %q", db_filename, err)
	}
	defer func() { _ = done() }()

	firstDigest := randomDigest()
	err = c4db.Update(func(tx *bbolt.Tx) error {
		rootBucket := tx.Bucket([]byte("c4"))
		keyBucket := rootBucket.Bucket([]byte("key"))
		return keyBucket.Put([]byte("wrapper/key/1"), firstDigest)
	})
	if err != nil {
		t.Fatalf("db.Update failed: %q", err)
	}

	var got []byte
	err = c4db.View(func(tx *bbolt.Tx) error {
		rootBucket := tx.Bucket([]byte("c4"))
		keyBucket := rootBucket.Bucket([]byte("key"))
		value := keyBucket.Get([]byte("wrapper/key/1"))
		got = append([]byte(nil), value...)
		return nil
	})
	if err != nil {
		t.Fatalf("db.View failed: %q", err)
	}
	if !bytes.Equal(got, firstDigest) {
		t.Fatalf("db.View returned wrong value")
	}

	secondDigest := randomDigest()
	err = c4db.Batch(func(tx *bbolt.Tx) error {
		rootBucket := tx.Bucket([]byte("c4"))
		keyBucket := rootBucket.Bucket([]byte("key"))
		return keyBucket.Put([]byte("wrapper/key/2"), secondDigest)
	})
	if err != nil {
		t.Fatalf("db.Batch failed: %q", err)
	}

	gotDigest, err := c4db.KeyGet("wrapper/key/2")
	if err != nil {
		t.Fatalf("KeyGet failed after Batch write: %q", err)
	}
	if gotDigest == nil || gotDigest.ID().Cmp(secondDigest.ID()) != 0 {
		t.Fatalf("db.Batch write did not persist expected value")
	}

	sentinelErr := errors.New("sentinel")
	err = c4db.Update(func(*bbolt.Tx) error {
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("db.Update did not return callback error")
	}
	err = c4db.View(func(*bbolt.Tx) error {
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("db.View did not return callback error")
	}
	err = c4db.Batch(func(*bbolt.Tx) error {
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("db.Batch did not return callback error")
	}
}

func TestOpenErrorPaths(t *testing.T) {
	root := t.TempDir()

	// Parent directory does not exist, so os.Mkdir(path, 0700) should fail.
	missingParentPath := filepath.Join(root, "missing", "dbdir")
	c4db, err := db.Open(missingParentPath, nil)
	if err == nil || c4db != nil {
		t.Fatalf("expected error opening path with missing parent")
	}

	// Path exists as a file, so opening <file>/db should fail.
	filePath := filepath.Join(root, "not_a_dir")
	err = os.WriteFile(filePath, []byte("x"), 0600)
	if err != nil {
		t.Fatalf("error creating file path: %q", err)
	}
	c4db, err = db.Open(filePath, nil)
	if err == nil || c4db != nil {
		t.Fatalf("expected error opening file path as db directory")
	}

	// Child bucket conflict should fail CreateBucketIfNotExists during Open.
	conflictDir := filepath.Join(root, "conflict")
	err = os.MkdirAll(conflictDir, 0700)
	if err != nil {
		t.Fatalf("error creating conflict dir: %q", err)
	}
	rawDB, err := bbolt.Open(filepath.Join(conflictDir, "db"), 0700, nil)
	if err != nil {
		t.Fatalf("error opening raw bolt db: %q", err)
	}
	err = rawDB.Update(func(tx *bbolt.Tx) error {
		rootBucket, err := tx.CreateBucketIfNotExists([]byte("c4"))
		if err != nil {
			return err
		}
		return rootBucket.Put([]byte("key"), []byte("not-a-bucket"))
	})
	if err != nil {
		_ = rawDB.Close()
		t.Fatalf("error creating bucket conflict: %q", err)
	}
	err = rawDB.Close()
	if err != nil {
		t.Fatalf("error closing raw bolt db: %q", err)
	}

	c4db, err = db.Open(conflictDir, nil)
	if err == nil || c4db != nil {
		t.Fatalf("expected error opening db with bucket conflict")
	}
}

func TestClosedDBErrorPaths(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "test.db")
	c4db, err := db.Open(dbPath, nil)
	if err != nil {
		t.Fatalf("error opening db at %q: %q", dbPath, err)
	}

	err = c4db.Close()
	if err != nil {
		t.Fatalf("error closing db: %q", err)
	}

	_, err = c4db.KeySet("closed/key", randomDigest())
	if err == nil {
		t.Fatalf("expected KeySet error on closed db")
	}
	_, err = c4db.KeyGet("closed/key")
	if err == nil {
		t.Fatalf("expected KeyGet error on closed db")
	}
	_, err = c4db.KeyDelete("closed/key")
	if err == nil {
		t.Fatalf("expected KeyDelete error on closed db")
	}
	if st := c4db.Stats(); st != nil {
		t.Fatalf("expected nil stats on closed db")
	}
}

func TestKeyGetAllErrorAndStop(t *testing.T) {
	c4db, done, err := mkdb("test.db", t)
	if err != nil {
		t.Fatalf("error opening db: %q", err)
	}
	defer func() { _ = done() }()

	err = c4db.Update(func(tx *bbolt.Tx) error {
		rootBucket := tx.Bucket([]byte("c4"))
		keyBucket := rootBucket.Bucket([]byte("key"))
		if err := keyBucket.Put([]byte("bad/000"), []byte("short")); err != nil {
			return err
		}
		if err := keyBucket.Put([]byte("bad/111"), randomDigest()); err != nil {
			return err
		}
		return keyBucket.Put([]byte("bad/222"), randomDigest())
	})
	if err != nil {
		t.Fatalf("error seeding invalid key data: %q", err)
	}

	ch := c4db.KeyGetAll("bad/")
	first, ok := <-ch
	if !ok {
		t.Fatalf("expected first entry from KeyGetAll")
	}
	if first.Err() == nil {
		first.Close()
		t.Fatalf("expected first KeyGetAll entry to include wrong value size error")
	}
	first.Stop()
	first.Close()

	for en := range ch {
		en.Close()
	}
}

func TestLinkStopPaths(t *testing.T) {
	c4db, done, err := mkdb("test.db", t)
	if err != nil {
		t.Fatalf("error opening db: %q", err)
	}
	defer func() { _ = done() }()

	sourceA := c4.Identify(strings.NewReader("sourceA-stop")).Digest()
	sourceB := c4.Identify(strings.NewReader("sourceB-stop")).Digest()
	targetA1 := c4.Identify(strings.NewReader("targetA1-stop")).Digest()
	targetA2 := c4.Identify(strings.NewReader("targetA2-stop")).Digest()
	targetB1 := c4.Identify(strings.NewReader("targetB1-stop")).Digest()

	err = c4db.LinkSet("stop", sourceA, targetA1, targetA2)
	if err != nil {
		t.Fatalf("error setting links for sourceA: %q", err)
	}
	err = c4db.LinkSet("stop", sourceB, targetB1)
	if err != nil {
		t.Fatalf("error setting links for sourceB: %q", err)
	}

	linkGet := c4db.LinkGet("stop", sourceA)
	first, ok := <-linkGet
	if !ok {
		t.Fatalf("expected first entry from LinkGet")
	}
	first.Stop()
	first.Close()
	for en := range linkGet {
		en.Close()
	}

	linkGetAllAny := c4db.LinkGetAll()
	first, ok = <-linkGetAllAny
	if !ok {
		t.Fatalf("expected first entry from LinkGetAll()")
	}
	first.Stop()
	first.Close()
	for en := range linkGetAllAny {
		en.Close()
	}

	linkGetAllBySource := c4db.LinkGetAll(sourceA)
	first, ok = <-linkGetAllBySource
	if !ok {
		t.Fatalf("expected first entry from LinkGetAll(source)")
	}
	first.Stop()
	first.Close()
	for en := range linkGetAllBySource {
		en.Close()
	}
}

func TestTreeSetErrorAndTreeGetMissing(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "test.db")
	missingStorage := filepath.Join(root, "missing-storage")
	c4db, err := db.Open(dbPath, &db.Options{
		TreeMaxSize:   1,
		TreeStrategy:  db.TreeStrategyCache,
		ExternalStore: []string{missingStorage},
	})
	if err != nil {
		t.Fatalf("error opening db at %q: %q", dbPath, err)
	}
	defer func() { _ = c4db.Close() }()

	var digests c4.DigestSlice
	digests.Insert(c4.Identify(strings.NewReader("tree-error-1")).Digest())
	digests.Insert(c4.Identify(strings.NewReader("tree-error-2")).Digest())
	tree := c4.NewTree(digests)
	_ = tree.Compute()

	err = c4db.TreeSet(tree)
	if err == nil {
		t.Fatalf("expected TreeSet to fail when external storage path is missing")
	}

	missingDigest := c4.Identify(strings.NewReader("missing-tree")).Digest()
	got, err := c4db.TreeGet(missingDigest)
	if err != nil {
		t.Fatalf("unexpected TreeGet error for missing tree: %q", err)
	}
	if got != nil {
		t.Fatalf("expected nil tree for missing digest")
	}
}

func TestKeyBatchOverwrite(t *testing.T) {
	c4db, done, err := mkdb("test.db", t)
	if err != nil {
		t.Fatalf("error opening db: %q", err)
	}
	defer func() { _ = done() }()

	first := c4.Identify(strings.NewReader("batch-first")).Digest()
	second := c4.Identify(strings.NewReader("batch-second")).Digest()

	c4db.KeyBatch(func(tx *db.Tx) bool {
		tx.KeySet("batch/duplicate", first)
		tx.KeySet("batch/duplicate", second)
		return false
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		got, err := c4db.KeyGet("batch/duplicate")
		if err != nil {
			t.Fatalf("error reading batch key: %q", err)
		}
		if got != nil && got.ID().Cmp(second.ID()) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected batch key to contain second digest")
		}
		time.Sleep(10 * time.Millisecond)
	}

	deadline = time.Now().Add(2 * time.Second)
	for {
		if len(c4db.KeyFind(first)) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("old digest index entry was not deleted")
		}
		time.Sleep(10 * time.Millisecond)
	}

	deadline = time.Now().Add(2 * time.Second)
	for {
		keys := c4db.KeyFind(second)
		if len(keys) == 1 && keys[0] == "batch/duplicate" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("new digest index entry missing after overwrite")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBatching(t *testing.T) {
	db_filename := "test.db"
	c4db, done, err := mkdb(db_filename, t)
	if err != nil {
		t.Errorf("error opening db at %q: %q", db_filename, err)
	}
	defer func() { _ = done() }()

	c4db.KeyBatch(func(tx *db.Tx) bool {
		// Use smaller batch size in short mode for better Windows performance
		batchSize := 100000
		if testing.Short() {
			batchSize = 10000
		}
		for i := 0; i < batchSize; i++ {
			tx.KeySet(strconv.Itoa(i), randomDigest())
			if tx.Err() != nil {
				t.Errorf("error during batch write")
				return false
			}
		}
		return false
	})

	st := c4db.Stats()
	t.Logf("Stats Trees:%d, Keys:%d, Indexes: %d, Links:%d, TreesSize:%d(%d)\n", st.Trees, st.Keys, st.KeyIndexes, st.Links, st.TreesSize, st.TreesSize/64)

	// Check against the actual batch size we used
	expectedKeys := 100000
	if testing.Short() {
		expectedKeys = 10000
	}
	if st.Keys != expectedKeys {
		t.Errorf("error tree has incorrect stats after delete, expected %d keys, got %d", expectedKeys, st.Keys)
		t.Errorf("Stats Trees:%d, Keys:%d, Indexes: %d, Links:%d, TreesSize:%d(%d)\n", st.Trees, st.Keys, st.KeyIndexes, st.Links, st.TreesSize, st.TreesSize/64)
	}

}

// utility to create a random c4.Digest
func randomDigest() c4.Digest {
	// Create some random bytes.
	var data [8]byte
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	for i := range data {
		data[i] = byte(rng.Uint32())
	}
	return c4.Identify(bytes.NewReader(data[:])).Digest()
}
