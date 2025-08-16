package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Avalanche-io/c4"
)

func TestFolderStore(t *testing.T) {
	path := os.TempDir()

	path = filepath.Join(path, "folder_test")
	err := os.Mkdir(path, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)

	folderStore := Folder(path)

	testdata := make(map[string]c4.ID)
	for i := 0; i < 100; i++ {

		// Create arbitrary test data
		key := fmt.Sprintf("%06d", i)

		// Create c4 id of the test data
		id := c4.Identify(strings.NewReader(key))
		testdata[key] = id

		// Test Folder store `Create` method
		w, err := folderStore.Create(id)
		if err != nil {
			t.Fatal(err)
		}

		// Write data to the Folder store
		_, err = w.Write([]byte(key))
		if err != nil {
			t.Fatal(err)
		}

		// Close the Folder store
		err = w.Close()
		if err != nil {
			t.Fatal(err)
		}

	}

	// Check that files with the appropreate C4 id are indeed located in the test
	// folder.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}

	var ids []string
	for _, v := range testdata {
		ids = append(ids, v.String())
	}

	if len(names) != len(ids) {
		t.Errorf("wrong number of results got %d expected %d", len(names), len(ids))
	}

	// Test all filenames against all ids
	for _, name := range names {
		found := false
		for _, id := range ids {
			if name == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("find file name that does not match an id in the list %q", name)
		}
	}

	// Test all ids against all filenames
	for _, id := range ids {
		found := false
		for _, name := range names {
			if id == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("an id was not matched by a file %s", id)
		}
	}

	// Test Folder store `Open` method
	for k, v := range testdata {

		f, err := folderStore.Open(v)
		if err != nil {
			t.Error(err)
		}

		data := make([]byte, 512)
		n, err := f.Read(data)
		if err != nil {
			t.Error(err)
		}

		data = data[:n]
		if string(data) != k {
			t.Errorf("wrong data read from file, expted %q, go %q", k, string(data))
		}
	}

}

func TestFolderStoreRemove(t *testing.T) {
	path := os.TempDir()
	path = filepath.Join(path, "folder_remove_test")
	err := os.Mkdir(path, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)

	folderStore := Folder(path)
	
	// Create test data
	testData := "test data for removal"
	id := c4.Identify(strings.NewReader(testData))
	
	// Create a file in the store
	w, err := folderStore.Create(id)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]byte(testData))
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	
	// Verify file exists
	_, err = folderStore.Open(id)
	if err != nil {
		t.Fatal("File should exist before removal:", err)
	}
	
	// Test Remove function
	err = folderStore.Remove(id)
	if err != nil {
		t.Fatal("Remove should succeed:", err)
	}
	
	// Verify file no longer exists
	_, err = folderStore.Open(id)
	if err == nil {
		t.Error("File should not exist after removal")
	}
	
	// Test removing non-existent file
	nonExistentID := c4.Identify(strings.NewReader("non-existent"))
	err = folderStore.Remove(nonExistentID)
	if err == nil {
		t.Error("Remove should fail for non-existent file")
	}
}

func TestFolderStoreCreateExisting(t *testing.T) {
	path := os.TempDir()
	path = filepath.Join(path, "folder_create_existing_test")
	err := os.Mkdir(path, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)

	folderStore := Folder(path)
	
	// Create test data
	testData := "test data for duplicate creation"
	id := c4.Identify(strings.NewReader(testData))
	
	// Create a file in the store
	w, err := folderStore.Create(id)
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	
	// Try to create the same file again (should fail)
	_, err = folderStore.Create(id)
	if err == nil {
		t.Error("Create should fail when file already exists")
	}
	
	// Verify it's a PathError with ErrExist
	if pathErr, ok := err.(*os.PathError); ok {
		if pathErr.Err != os.ErrExist {
			t.Errorf("Expected ErrExist, got: %v", pathErr.Err)
		}
	} else {
		t.Errorf("Expected *os.PathError, got: %T", err)
	}
}
