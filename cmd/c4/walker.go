package main

import (
	"fmt"
	"os"
	"path/filepath"

	c4 "github.com/bgyss/c4/id"
)

func newItem(path string) (item map[string]interface{}) {
	item = make(map[string]interface{})
	f, err := os.Lstat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get status for \"%s\": %s\n", path, err)
		exitFunc(1)
	}

	item["folder"] = f.IsDir()
	item["link"] = f.Mode()&os.ModeSymlink == os.ModeSymlink
	item["socket"] = f.Mode()&os.ModeSocket == os.ModeSocket
	item["bytes"] = f.Size()
	item["modified"] = f.ModTime().UTC()

	return item
}

func walkFilesystem(depth int, filename string, relative_path string) (id *c4.ID) {
	path, err := absPath(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to find absolute path for %s. %s\n", filename, err)
		exitFunc(1)
	}

	item := newItem(path)
	if item["socket"] == true {
		id = nullId()
	} else if item["link"] == true && !links_flag {
		newFilepath, _ := evalSymlinks(filename)
		item["link"] = newFilepath
		id = nullId()
	} else if item["link"] == true {
		newFilepath, err := evalSymlinks(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to follow link %s. %s\n", newFilepath, err)
			item["link"] = newFilepath
			id = nullId()
		} else {
			item["link"] = newFilepath
			var linkId c4.Slice
			linkId.Insert(walkFilesystem(depth-1, newFilepath, relative_path))
			id = linkId.ID()
		}
	} else {
		if item["folder"] == true {
			files, err := os.ReadDir(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to read directory: %v\n", err)
				exitFunc(1)
			}
			var childIDs c4.Slice
			for _, file := range files {
				path := filename + string(filepath.Separator) + file.Name()
				childIDs.Insert(walkFilesystem(depth-1, path, relative_path))
			}
			id = childIDs.ID()
		} else {
			id = fileID(path)
		}
	}
	item["c4id"] = id.String()
	if depth >= 0 || recursive_flag {
		output(path, item)
	}
	return
}
