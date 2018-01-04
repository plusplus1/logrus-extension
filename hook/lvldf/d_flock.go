package lvldf

import (
	"runtime"
	"strings"
)
import (
	"golang.org/x/sys/unix"
)

var (
	unixSystemMap = map[string]int{
		"darwin":  1,
		"linux":   1,
		"freebsd": 1,
		"netbsd":  1,
		"openbsd": 1,
		"solaris": 1,
	}

	flockUtil = flock{}
)

type flock struct{}

func (u *flock) isUnix() bool {
	osName := strings.ToLower(runtime.GOOS)
	if _, ok := unixSystemMap[osName]; ok {
		return true
	}
	return false
}

// Lock, file lock
func (u *flock) Lock(fd uintptr) error {
	if !u.isUnix() {
		return nil
	}
	_, _, errNo := unix.Syscall(unix.SYS_FLOCK, fd, unix.LOCK_EX, 0)
	if errNo == 0 {
		return nil
	}
	return errNo
}

// UnLock, file unlock
func (u *flock) UnLock(fd uintptr) error {
	if !u.isUnix() {
		return nil
	}
	_, _, errNo := unix.Syscall(unix.SYS_FLOCK, fd, unix.LOCK_UN, 0)
	if errNo == 0 {
		return nil
	}
	return errNo

}
