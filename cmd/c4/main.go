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

type cliHooks struct {
	parseFlags      func()
	flagArgs        func() []string
	exit            func(int)
	absPath         func(string) (string, error)
	evalSymlinks    func(string) (string, error)
	identifyPipeFn  func()
	identifyFileFn  func(string)
	identifyFilesFn func([]string)
}

var hooks cliHooks

func init() {
	hooks.parseFlags = flag.Parse
	hooks.flagArgs = flag.Args
	hooks.exit = os.Exit
	hooks.absPath = filepath.Abs
	hooks.evalSymlinks = filepath.EvalSymlinks
	hooks.identifyPipeFn = identify_pipe
	hooks.identifyFileFn = identify_file
	hooks.identifyFilesFn = identify_files
}

func versionString() string {
	return `c4 version ` + version_number + ` (` + runtime.GOOS + `)`
}

func main() {
	hooks.parseFlags()
	file_list := hooks.flagArgs()
	if version_flag {
		fmt.Println(versionString())
		hooks.exit(0)
	}

	if len(file_list) == 0 {
		hooks.identifyPipeFn()
	} else if len(file_list) == 1 && !recursive_flag && !include_meta && depth == 0 {
		hooks.identifyFileFn(file_list[0])
	} else {
		hooks.identifyFilesFn(file_list)
	}
	hooks.exit(0)
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
	path, err := hooks.absPath(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to find absolute path for %s. %s\n", filename, err)
		hooks.exit(1)
	}
	id := walkFilesystem(-1, path, "")
	printID(id)
}

func identify_files(file_list []string) {
	for _, file := range file_list {
		path, err := hooks.absPath(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to find absolute path for %s. %s\n", file, err)
			hooks.exit(1)
		}
		if depth < 0 {
			depth = 0
		}
		walkFilesystem(depth, path, "")
	}
}
