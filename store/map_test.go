package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bgyss/c4"
)

func TestMapStore(t *testing.T) {
	tmp, done, err := MkTmp("TestMapStore")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	m := make(map[c4.ID]string)
	ms := NewMap(m)
	var ids []c4.ID
	for i := 100; i > 0; i-- {
		data := fmt.Sprintf("%04d", i)
		filename := filepath.Join(tmp, data)
		f, err := os.Create(filename)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.WriteString(data)
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
		id := c4.Identify(bytes.NewReader([]byte(data)))
		ids = append(ids, id)
		ms.LoadOrStore(id, filename)
	}

	ms.Range(func(id c4.ID, path string) bool {
		if m[id] != path {
			t.Error("wrong map content")
		}
		return true
	})

	if len(m) != len(ids) {
		t.Errorf("counts don't match %d %d", len(m), len(ids))
	}

	// Test all filenames against all ids
	for i, id := range ids {
		f, err := ms.Open(id)
		if err != nil {
			t.Fatal(err)
		}
		testid := c4.Identify(f)
		_ = f.Close()
		if testid != id {
			t.Fatalf("wrong id content %d", i)
		}

	}
}

func TestMapStoreOperations(t *testing.T) {
	tmp, done, err := MkTmp("TestMapStoreOperations")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	
	m := make(map[c4.ID]string)
	ms := NewMap(m)
	
	// Create test file
	testFile := filepath.Join(tmp, "test.txt")
	testData := "test data for map store"
	id := c4.Identify(bytes.NewReader([]byte(testData)))
	
	// Create the actual file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(testData)
	_ = f.Close()
	
	// Test LoadOrStore with new ID
	actual, loaded := ms.LoadOrStore(id, testFile)
	if loaded {
		t.Errorf("LoadOrStore should return loaded=false for new ID")
	}
	if actual != testFile {
		t.Errorf("LoadOrStore returned wrong path: %s", actual)
	}
	
	// Test LoadOrStore with existing ID
	actual2, loaded2 := ms.LoadOrStore(id, "different/path")
	if !loaded2 {
		t.Errorf("LoadOrStore should return loaded=true for existing ID")
	}
	if actual2 != testFile {
		t.Errorf("LoadOrStore should return original path: %s", actual2)
	}
	
	// Test Load function
	path := ms.Load(id)
	if path != testFile {
		t.Errorf("Load returned wrong path: %s", path)
	}
	
	// Test Load with non-existent ID
	nonExistentID := c4.Identify(bytes.NewReader([]byte("non-existent")))
	emptyPath := ms.Load(nonExistentID)
	if emptyPath != "" {
		t.Errorf("Load should return empty string for non-existent ID: %s", emptyPath)
	}
	
	// Test Create function
	newFile := filepath.Join(tmp, "new.txt")
	ms.LoadOrStore(nonExistentID, newFile)
	writer, err := ms.Create(nonExistentID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = writer.Write([]byte("new test data"))
	if err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	
	// Verify the created file
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Errorf("Create should have created file: %s", newFile)
	}
	
	// Test Remove function
	err = ms.Remove(nonExistentID)
	if err != nil {
		t.Fatal(err)
	}
	
	// Verify file was removed
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Errorf("Remove should have deleted file: %s", newFile)
	}
	
	// Test Delete function (removes from map)
	if _, exists := m[id]; !exists {
		t.Fatal("ID should exist in map before Delete")
	}
	ms.Delete(id)
	if _, exists := m[id]; exists {
		t.Errorf("Delete should have removed ID from map")
	}
	
	// Test Range function with early termination
	ms.LoadOrStore(id, testFile)
	ms.LoadOrStore(nonExistentID, "another/path")
	
	count := 0
	ms.Range(func(rangeID c4.ID, path string) bool {
		count++
		return count < 1 // Stop after first iteration
	})
	if count != 1 {
		t.Errorf("Range should have stopped early, got count: %d", count)
	}
}

func MkTmp(name string) (string, func(), error) {
	path := os.TempDir()
	path = filepath.Join(path, name)
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return "", func() {}, err
	}

	return path, func() {
		_ = os.RemoveAll(path)
	}, nil
}
