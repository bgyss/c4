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

func identifyFileMain(filename string) {
	_ = identify_file(filename)
}

func identifyFilesMain(fileList []string) {
	_ = identify_files(fileList)
}

var (
	// Compatibility test seams. By default they delegate through hooks.
	exitFn = func(code int) {
		hooks.exit(code)
	}
	absPath = func(path string) (string, error) {
		return hooks.absPath(path)
	}
	identifyPipeFn  = identify_pipe
	identifyFileFn  = identify_file
	identifyFilesFn = identify_files
	stdinStat       = func() (os.FileInfo, error) {
		return os.Stdin.Stat()
	}
)

func init() {
	hooks.parseFlags = flag.Parse
	hooks.flagArgs = flag.Args
	hooks.exit = os.Exit
	hooks.absPath = filepath.Abs
	hooks.evalSymlinks = filepath.EvalSymlinks
	hooks.identifyPipeFn = identify_pipe
	hooks.identifyFileFn = identifyFileMain
	hooks.identifyFilesFn = identifyFilesMain
}

func versionString() string {
	return `c4 version ` + version_number + ` (` + runtime.GOOS + `)`
}

func main() {
	hooks.parseFlags()
	fileList := hooks.flagArgs()
	if version_flag {
		fmt.Println(versionString())
		hooks.exit(0)
	}

	if len(fileList) == 0 {
		hooks.identifyPipeFn()
	} else if len(fileList) == 1 && !recursive_flag && !include_meta && depth == 0 {
		hooks.identifyFileFn(fileList[0])
	} else {
		hooks.identifyFilesFn(fileList)
	}
	hooks.exit(0)
}

func run() int {
	flag.Parse()
	fileList := flag.Args()
	if version_flag {
		fmt.Println(versionString())
		return 0
	}

	if len(fileList) == 0 {
		identifyPipeFn()
		return 0
	}
	if len(fileList) == 1 && !recursive_flag && !include_meta && depth == 0 {
		return identifyFileFn(fileList[0])
	}
	return identifyFilesFn(fileList)
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
		exitFn(1)
		return 1
	}
	id := walkFilesystem(-1, path, "")
	printID(id)
	return 0
}

func identify_files(fileList []string) int {
	for _, file := range fileList {
		path, err := absPath(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to find absolute path for %s. %s\n", file, err)
			exitFn(1)
			return 1
		}
		if depth < 0 {
			depth = 0
		}
		walkFilesystem(depth, path, "")
	}
	return 0
}
