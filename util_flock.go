package logrus_extension

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
)

func isUnix() bool {
	osName := strings.ToLower(runtime.GOOS)
	if _, ok := unixSystemMap[osName]; ok {
		return true
	}
	return false
}

func accuireFileLock(fd uintptr) error {
	if isUnix() {
		if _, _, e := unix.Syscall(unix.SYS_FLOCK, fd, unix.LOCK_EX, 0); e != 0 {
			return e
		}
	}
	return nil
}

func releaseFileLock(fd uintptr) error {
	if isUnix() {
		if _, _, e := unix.Syscall(unix.SYS_FLOCK, fd, unix.LOCK_UN, 0); e != 0 {
			return e
		}
	}
	return nil
}
