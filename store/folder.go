package store

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bgyss/c4"
)

var _ Store = Folder("")

var (
	folderAbs  = filepath.Abs
	idToString = func(id c4.ID) string { return id.String() }
)

// Folder is an implementation of the Store interface that uses c4 id nameed
// files in a filsystem folder.
type Folder string

// validatePath prevents directory traversal attacks by ensuring the resolved path
// stays within the folder boundary
func (f Folder) validatePath(id c4.ID) (string, error) {
	idStr := idToString(id)
	// Check for obvious path traversal attempts
	if strings.Contains(idStr, "..") || strings.Contains(idStr, "/") || strings.Contains(idStr, "\\") {
		return "", errors.New("invalid ID: contains path traversal characters")
	}

	basePath, err := folderAbs(string(f))
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(basePath, idStr)
	resolvedPath, err := folderAbs(targetPath)
	if err != nil {
		return "", err
	}

	// Ensure the resolved path is still within the base directory
	if !strings.HasPrefix(resolvedPath, basePath+string(filepath.Separator)) &&
		resolvedPath != basePath {
		return "", errors.New("path traversal attack detected")
	}

	return targetPath, nil
}

// Open opens a file named the given c4.ID in read-only mode from the folder. If
// the file does not exist an error is returned.
func (f Folder) Open(id c4.ID) (io.ReadCloser, error) {
	path, err := f.validatePath(id)
	if err != nil {
		return nil, err
	}
	return os.Open(path) // #nosec G304 - path is validated by validatePath function
}

// Create creates and opens for writting a file with the given c4 id as it's
// name if the file does not already exist. If it cannot open the file or the
// file already exists it returns an error.
func (f Folder) Create(id c4.ID) (io.WriteCloser, error) {
	path, err := f.validatePath(id)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return nil, &os.PathError{Op: "create", Path: path, Err: os.ErrExist}
	}
	return os.Create(path) // #nosec G304 - path is validated by validatePath function
}

func (f Folder) Remove(id c4.ID) error {
	path, err := f.validatePath(id)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
