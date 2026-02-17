package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	c4 "github.com/bgyss/c4/id"
)

var openFile = os.Open
var stdoutStat = func() (os.FileInfo, error) {
	return os.Stdout.Stat()
}

func encode(src io.Reader) *c4.ID {
	e := c4.NewEncoder()
	_, err := io.Copy(e, src)
	if err != nil {
		panic(err)
	}
	return e.ID()
}

func fileID(path string) (id *c4.ID) {
	// Clean the path to prevent issues with malformed paths
	cleanPath := filepath.Clean(path)
	f, err := openFile(cleanPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to identify %s. %v\n", path, err)
		exitFn(1)
		return nullId()
	}
	id = encode(f)
	_ = f.Close()
	return
}

func nullId() *c4.ID {
	e := c4.NewEncoder()
	_, _ = io.Copy(e, strings.NewReader(``))
	return e.ID()
}

func printID(id *c4.ID) {
	stat, err := stdoutStat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) != 0 {
		fmt.Printf("%s\n", id.String())
	} else {
		fmt.Printf("%s", id.String())
	}
}
