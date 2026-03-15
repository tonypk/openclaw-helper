//go:build windows

package installer

import (
	"golang.org/x/sys/windows"
)

// IsAdmin returns true if the current process has administrator privileges.
func IsAdmin() bool {
	token := windows.GetCurrentProcessToken()
	return token.IsElevated()
}
