//go:build windows

package roots

import (
	"path/filepath"
	"testing"
)

func TestIsDriveRootWindows(t *testing.T) {
	if !IsDriveRoot(`C:\`) {
		t.Fatal("expected C:\\ to be a drive root")
	}
	if IsDriveRoot(filepath.Join(`C:\`, "Windows")) {
		t.Fatal("C:\\Windows should not be a drive root")
	}
}

func TestIsBroadHomeRootWindowsUsers(t *testing.T) {
	if !IsBroadHomeRoot(`C:\Users`) || !IsBroadHomeRoot(`C:\Users\someone`) {
		t.Fatal("expected Windows user home patterns")
	}
	if IsBroadHomeRoot(`C:\Users\someone\code`) {
		t.Fatal("nested user path should not be broad")
	}
}
