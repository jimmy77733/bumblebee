//go:build unix

package endpoint

import (
	"os"
	"strconv"
)

func fallbackUID() string {
	return strconv.Itoa(os.Getuid())
}
