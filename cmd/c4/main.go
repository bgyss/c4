package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	flag "github.com/ogier/pflag"
)

const version_number = "0.8.1"

func versionString() string {
	return `c4 version ` + version_number + ` (` + runtime.GOOS + `)`
}

var (
	exitFn          = os.Exit
	absPath         = filepath.Abs
	identifyPipeFn  = identify_pipe
	identifyFileFn  = identify_file
	identifyFilesFn = identify_files
	stdinStat       = func() (os.FileInfo, error) {
		return os.Stdin.Stat()
	}
)

func main() {
	exitFn(run())
}

func run() int {
	flag.Parse()
	file_list := flag.Args()
	if version_flag {
		fmt.Println(versionString())
		return 0
	}

	if len(file_list) == 0 {
		identifyPipeFn()
	} else if len(file_list) == 1 && !recursive_flag && !include_meta && depth == 0 {
		return identifyFileFn(file_list[0])
	} else {
		return identifyFilesFn(file_list)
	}
	return 0
}

func identify_pipe() {
	stat, err := stdinStat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		printID(encode(reader))
	} else {
		flag.Usage()
	}
}

func identify_file(filename string) int {
	path, err := absPath(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to find absolute path for %s. %s\n", filename, err)
		return 1
	}
	id := walkFilesystem(-1, path, "")
	printID(id)
	return 0
}

func identify_files(file_list []string) int {
	for _, file := range file_list {
		path, err := absPath(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to find absolute path for %s. %s\n", file, err)
			return 1
		}
		if depth < 0 {
			depth = 0
		}
		walkFilesystem(depth, path, "")
	}
	return 0
}
