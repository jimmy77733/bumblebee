//go:build windows

package htmlreport

import "golang.org/x/sys/windows"

func downloadsDir() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Downloads, 0)
}
