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
	char := input[0]
	input = strings.ToLower(input)
	switch char {
	case '-':
	case 'd':
		// In the string representation produced by os.FileMode,
		// the letter 'd' is used for both directories and device files
		// (uppercase 'D').  If any execute bits are present we assume a
		// directory, otherwise a device file.
		if strings.ContainsRune(input[1:10], 'x') {
			mode |= os.ModeDir
		} else {
			mode |= os.ModeDevice
		}
	case 'D':
		// device file
		mode |= os.ModeDevice
	case 'a':
		// append-only
		mode |= os.ModeAppend
	case 'l':
		// symbolic link
		mode |= os.ModeSymlink
	case 't':
		// temporary file
		mode |= os.ModeTemporary
	case 'p':
		// named pipe (FIFO)
		mode |= os.ModeNamedPipe
	case 's':
		// Unix domain socket
		mode |= os.ModeSocket
	case 'u':
		// setuid
		mode |= os.ModeSetuid
	case 'g':
		// setgid
		mode |= os.ModeSetgid
	case 'c':
		// Unix character device, when ModeDevice is set
		mode |= os.ModeDevice | os.ModeCharDevice
	case 'b':
		// Unix block device
		mode |= os.ModeDevice
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
