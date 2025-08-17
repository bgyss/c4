package store

import (
	"fmt"
	"crypto/rand"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/bgyss/c4"
)

func TestValidatingStore(t *testing.T) {
	var st Store
	ramst := NewRAM()
	st = NewValidating(ramst)

	randIndx := secureRandIntN(100)
	t.Logf("random index: %d", randIndx)
	testdata := make(map[string]c4.ID)
	for i := 0; i < 100; i++ {

		// Create arbitrary test data
		key := fmt.Sprintf("%06d", i)

		// Create c4 id of the test data
		id := c4.Identify(strings.NewReader(key))
		testdata[key] = id

		// Test Validating store `Create` method
		w, err := st.Create(id)
		if err != nil {
			t.Fatal(err)
		}
		actualKey := key
		if i == randIndx {
			key = "bad data"
		}
		// Write data to the Validating store
		_, err = w.Write([]byte(key))
		if err != nil {
			t.Fatal(err)
		}

		// Close the Validating store
		err = w.Close()
		if i == randIndx {
			if err != ErrInvalidID {
				t.Errorf("expected error on reader close.")
			}
			delete(testdata, actualKey)
			continue
		}
		if err != nil {
			t.Fatal(err)
		}

	}

	// Test Validating store `Open` method
	for k, v := range testdata {

		f, err := st.Open(v)
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
			t.Errorf("wrong data read from file, expted %q, got %q", k, string(data))
		}
		err = f.Close()
		if err != nil {
			t.Errorf("reader close with error %s", err)
		}
	}

	var badID c4.ID

	// sneek in some wrong data
	idmap := (map[c4.ID][]byte)(*ramst)
	if len(idmap) != len(testdata) {
		t.Error("test data and ram store length do not match")
	}
	for k := range idmap {
		idmap[k] = []byte("bad data")
		badID = k
		break
	}

	// Test Validating store `Open` method
	for _, v := range testdata {
		f, err := st.Open(v)
		if err != nil {
			t.Error(err)
		}

		data := make([]byte, 512)
		_, err = f.Read(data)
		if err != nil {
			t.Error(err)
		}

		_ = badID
		err = f.Close()
		if v == badID {
			if err != ErrInvalidID {
				t.Errorf("expected error on reader close.")
			}
			continue
		}
		if err != nil {
			t.Errorf("reader close with error %s", err)
		}
	}

}

func TestValidatingStoreRemove(t *testing.T) {
	ramst := NewRAM()
	st := NewValidating(ramst)
	
	// Create test data
	testData := "test data for validating remove"
	id := c4.Identify(strings.NewReader(testData))
	
	// Create a file in the store
	w, err := st.Create(id)
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
	_, err = st.Open(id)
	if err != nil {
		t.Fatal("File should exist before removal:", err)
	}
	
	// Test Remove function
	err = st.Remove(id)
	if err != nil {
		t.Fatal("Remove should succeed:", err)
	}
	
	// Verify file no longer exists
	_, err = st.Open(id)
	if err == nil {
		t.Error("File should not exist after removal")
	}
}

func TestValidatingStoreErrorCases(t *testing.T) {
	ramst := NewRAM()
	st := NewValidating(ramst)
	
	// Test opening non-existent file
	nonExistentID := c4.Identify(strings.NewReader("non-existent"))
	_, err := st.Open(nonExistentID)
	if err == nil {
		t.Error("Open should fail for non-existent file")
	}
	
	// Test creating with invalid write data (write different data than ID)
	validData := "valid data"
	validID := c4.Identify(strings.NewReader(validData))
	
	w, err := st.Create(validID)
	if err != nil {
		t.Fatal(err)
	}
	
	// Write different data (should cause validation error)
	_, err = w.Write([]byte("different data"))
	if err != nil {
		t.Fatal(err)
	}
	
	err = w.Close()
	if err != ErrInvalidID {
		t.Errorf("Expected ErrInvalidID, got: %v", err)
	}
	
	// Verify the invalid file was not stored
	_, err = st.Open(validID)
	if err == nil {
		t.Error("Invalid file should not be stored")
	}
}

// secureRandIntN returns a cryptographically secure random integer in [0, n)
func secureRandIntN(n int) int {
	if n <= 0 {
		return 0
	}
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int(binary.BigEndian.Uint64(b[:]) % uint64(n))
}
