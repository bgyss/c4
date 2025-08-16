package store

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Avalanche-io/c4"
)

func TestLoggerStore(t *testing.T) {
	path := os.TempDir()
	path = filepath.Join(path, "logger_test")
	err := os.Mkdir(path, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)
	buff := new(bytes.Buffer)
	logger := NewLogger(Folder(path), buff, 0)
	var st Store
	st = logger
	// Create arbitrary test data
	testdata := "foo"
	id := c4.Identify(strings.NewReader(testdata))

	// Test Logger Create
	w, err := st.Create(id)
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 512)

	n, _ := buff.Read(data)
	if string(data[:n]) != fmt.Sprintf("%s Create\n", id) {
		t.Errorf("log output for Create does not match expected")
	}
	// Test Logger io.WriteCloser Write
	_, err = w.Write([]byte(testdata))
	if err != nil {
		t.Fatal(err)
	}

	n, _ = buff.Read(data)
	if string(data[:n]) != fmt.Sprintf("%s Write %d\n", id, len(testdata)) {
		t.Errorf("log output for Write does not match expected")
	}
	// Test Logger io.WriteCloser Close
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}

	n, _ = buff.Read(data)
	if string(data[:n]) != fmt.Sprintf("%s Close\n", id) {
		t.Errorf("log output for Close does not match expected")
	}

	// Test Logger Open
	f, err := st.Open(id)
	if err != nil {
		t.Error(err)
	}

	n, _ = buff.Read(data)
	if string(data[:n]) != fmt.Sprintf("%s Open\n", id) {
		t.Errorf("log output for Open does not match expected")
	}

	data2 := make([]byte, 512)
	n, err = f.Read(data2)
	if err != nil {
		t.Error(err)
	}
	data2 = data2[:n]
	if string(data2) != testdata {
		t.Errorf("wrong data read from file, expted %q, go %q", testdata, string(data2))
	}

	n, _ = buff.Read(data)
	if string(data[:n]) != fmt.Sprintf("%s Read %d\n", id, len(testdata)) {
		t.Errorf("log output for Read does not match expected")
	}

	_, err = f.Read(data2)
	if err != io.EOF {
		t.Errorf("expected io.EOF, but got %v", err)
	}
	n, _ = buff.Read(data)
	if string(data[:n]) != fmt.Sprintf("%s Read %d\n%s Read error EOF\n", id, 0, id) {
		t.Errorf("log output for Read does not match expected")
	}

	f.Close()
	n, _ = buff.Read(data)
	if string(data[:n]) != fmt.Sprintf("%s Close\n", id) {
		t.Errorf("log output for Close does not match expected")
	}
}

func TestLoggerStoreRemove(t *testing.T) {
	path := os.TempDir()
	path = filepath.Join(path, "logger_remove_test")
	err := os.Mkdir(path, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)
	
	buff := new(bytes.Buffer)
	logger := NewLogger(Folder(path), buff, LogRemove|LogError|LogCreate|LogWrite|LogClose)
	
	// Create test data
	testdata := "test data for logger removal"
	id := c4.Identify(strings.NewReader(testdata))
	
	// Create a file
	w, err := logger.Create(id)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(testdata))
	w.Close()
	
	// Clear the buffer
	buff.Reset()
	
	// Test Remove function
	err = logger.Remove(id)
	if err != nil {
		t.Fatal("Remove should succeed:", err)
	}
	
	// Check log output
	data := make([]byte, 512)
	n, _ := buff.Read(data)
	expected := fmt.Sprintf("%s Remove\n", id)
	if string(data[:n]) != expected {
		t.Errorf("log output for Remove does not match expected, got: %q, want: %q", string(data[:n]), expected)
	}
	
	// Verify file no longer exists
	_, err = logger.Open(id)
	if err == nil {
		t.Error("File should not exist after removal")
	}
}

func TestLoggerStoreErrorCases(t *testing.T) {
	path := os.TempDir()
	path = filepath.Join(path, "logger_error_test")
	err := os.Mkdir(path, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)
	
	buff := new(bytes.Buffer)
	logger := NewLogger(Folder(path), buff, LogRemove|LogError|LogOpen|LogCreate)
	
	// Test Remove non-existent file
	nonExistentID := c4.Identify(strings.NewReader("non-existent"))
	err = logger.Remove(nonExistentID)
	if err == nil {
		t.Error("Remove should fail for non-existent file")
	}
	
	// Check error log output
	data := make([]byte, 512)
	n, _ := buff.Read(data)
	expectedLog := fmt.Sprintf("%s Remove error", nonExistentID)
	if !strings.Contains(string(data[:n]), expectedLog) {
		t.Errorf("log should contain error message, got: %q", string(data[:n]))
	}
	
	// Test Open non-existent file
	buff.Reset()
	_, err = logger.Open(nonExistentID)
	if err == nil {
		t.Error("Open should fail for non-existent file")
	}
	
	// Check error log output
	n, _ = buff.Read(data)
	expectedLog = fmt.Sprintf("%s Open error", nonExistentID)
	if !strings.Contains(string(data[:n]), expectedLog) {
		t.Errorf("log should contain error message, got: %q", string(data[:n]))
	}
	
	// Test Create with error (try to create in non-existent directory)
	buff.Reset()
	badLogger := NewLogger(Folder("/non/existent/path"), buff, LogCreate|LogError)
	testID := c4.Identify(strings.NewReader("test"))
	_, err = badLogger.Create(testID)
	if err == nil {
		t.Error("Create should fail for non-existent directory")
	}
	
	// Check error log output
	n, _ = buff.Read(data)
	expectedLog = fmt.Sprintf("%s Create error", testID)
	if !strings.Contains(string(data[:n]), expectedLog) {
		t.Errorf("log should contain error message, got: %q", string(data[:n]))
	}
}

func TestLoggerWriterErrorCases(t *testing.T) {
	path := os.TempDir()
	path = filepath.Join(path, "logger_writer_error_test")
	err := os.Mkdir(path, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(path)
	
	buff := new(bytes.Buffer)
	logger := NewLogger(Folder(path), buff, 0)
	
	testdata := "test data"
	id := c4.Identify(strings.NewReader(testdata))
	
	// Create a writer
	w, err := logger.Create(id)
	if err != nil {
		t.Fatal(err)
	}
	
	// Write to it successfully first
	_, err = w.Write([]byte(testdata))
	if err != nil {
		t.Fatal(err)
	}
	
	// Close it
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	
	// Now try to write to closed writer
	buff.Reset()
	_, err = w.Write([]byte("more data"))
	if err == nil {
		t.Error("Write should fail on closed writer")
	}
	
	// Try to close again
	err = w.Close()
	if err == nil {
		t.Error("Close should fail on already closed writer")
	}
}
