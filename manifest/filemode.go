package manifest

import (
	"errors"
	"os"
	"strings"
)

func ParseFileMode(input string) (os.FileMode, error) {
	var mode os.FileMode

	if len(input) < 10 {
		return 0, errors.New("unable to parse file mode string too short")
	}
	input = strings.ToLower(input)
	switch input[0] {
	case '-':
	case 'd':
		mode |= os.ModeDir | os.ModeDevice // d: directory or device file
	case 'a':
		mode |= os.ModeAppend // a: append-only
	case 'l':
		mode |= os.ModeExclusive | os.ModeSymlink // l: exclusive use or symbolic link
	case 't':
		mode |= os.ModeTemporary | os.ModeSticky // t: temporary file or sticky
	case 'p':
		mode |= os.ModeNamedPipe // p: named pipe (FIFO)
	case 's':
		mode |= os.ModeSocket // s: Unix domain socket
	case 'u':
		mode |= os.ModeSetuid // u: setuid
	case 'g':
		mode |= os.ModeSetgid // g: setgid
	case 'c':
		mode |= os.ModeCharDevice // c: Unix character device, when ModeDevice is set
	}

	if input[1] == 'r' {
		mode |= OS_USER_R
	}
	if input[2] == 'w' {
		mode |= OS_USER_W
	}
	if input[3] == 'x' {
		mode |= OS_USER_X
	}
	if input[4] == 'r' {
		mode |= OS_GROUP_R
	}
	if input[5] == 'w' {
		mode |= OS_GROUP_W
	}
	if input[6] == 'x' {
		mode |= OS_GROUP_X
	}
	if input[7] == 'r' {
		mode |= OS_OTH_R
	}
	if input[8] == 'w' {
		mode |= OS_OTH_W
	}
	if input[9] == 'x' {
		mode |= OS_OTH_X
	}

	return mode, nil
}

const (
	OS_READ        = 04
	OS_WRITE       = 02
	OS_EX          = 01
	OS_USER_SHIFT  = 6
	OS_GROUP_SHIFT = 3
	OS_OTH_SHIFT   = 0

	OS_USER_R   = OS_READ << OS_USER_SHIFT
	OS_USER_W   = OS_WRITE << OS_USER_SHIFT
	OS_USER_X   = OS_EX << OS_USER_SHIFT
	OS_USER_RW  = OS_USER_R | OS_USER_W
	OS_USER_RWX = OS_USER_RW | OS_USER_X

	OS_GROUP_R   = OS_READ << OS_GROUP_SHIFT
	OS_GROUP_W   = OS_WRITE << OS_GROUP_SHIFT
	OS_GROUP_X   = OS_EX << OS_GROUP_SHIFT
	OS_GROUP_RW  = OS_GROUP_R | OS_GROUP_W
	OS_GROUP_RWX = OS_GROUP_RW | OS_GROUP_X

	OS_OTH_R   = OS_READ << OS_OTH_SHIFT
	OS_OTH_W   = OS_WRITE << OS_OTH_SHIFT
	OS_OTH_X   = OS_EX << OS_OTH_SHIFT
	OS_OTH_RW  = OS_OTH_R | OS_OTH_W
	OS_OTH_RWX = OS_OTH_RW | OS_OTH_X

	OS_ALL_R   = OS_USER_R | OS_GROUP_R | OS_OTH_R
	OS_ALL_W   = OS_USER_W | OS_GROUP_W | OS_OTH_W
	OS_ALL_X   = OS_USER_X | OS_GROUP_X | OS_OTH_X
	OS_ALL_RW  = OS_ALL_R | OS_ALL_W
	OS_ALL_RWX = OS_ALL_RW | OS_GROUP_X
)
