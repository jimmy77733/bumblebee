//go:build !unix

package scanner

import (
	"errors"
	"syscall"
)

func isSyscallPermission(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == syscall.EACCES
	}
	return false
}
