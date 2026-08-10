//go:build unix

package scanner

import (
	"errors"
	"syscall"
)

func isSyscallPermission(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.EACCES, syscall.EPERM:
			return true
		}
	}
	return false
}
