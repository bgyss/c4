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

var (
	parseFlags      = flag.Parse
	flagArgs        = flag.Args
	exitFunc        = os.Exit
	absPath         = filepath.Abs
	evalSymlinks    = filepath.EvalSymlinks
	identifyPipeFn  = identify_pipe
	identifyFileFn  = identify_file
	identifyFilesFn = identify_files
)

func versionString() string {
	return `c4 version ` + version_number + ` (` + runtime.GOOS + `)`
}

func main() {
	parseFlags()
	file_list := flagArgs()
	if version_flag {
		fmt.Println(versionString())
		exitFunc(0)
	}

	if len(file_list) == 0 {
		identifyPipeFn()
	} else if len(file_list) == 1 && !recursive_flag && !include_meta && depth == 0 {
		identifyFileFn(file_list[0])
	} else {
		identifyFilesFn(file_list)
	}
	exitFunc(0)
}

func identify_pipe() {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		printID(encode(reader))
	} else {
		flag.Usage()
	}
}

func identify_file(filename string) {
	path, err := absPath(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to find absolute path for %s. %s\n", filename, err)
		exitFunc(1)
	}
	id := walkFilesystem(-1, path, "")
	printID(id)
}

func identify_files(file_list []string) {
	for _, file := range file_list {
		path, err := absPath(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to find absolute path for %s. %s\n", file, err)
			exitFunc(1)
		}
		if depth < 0 {
			depth = 0
		}
		walkFilesystem(depth, path, "")
	}
}
