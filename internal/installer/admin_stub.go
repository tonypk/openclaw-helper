//go:build !windows

package installer

// IsAdmin always returns true on non-Windows (no elevation needed).
func IsAdmin() bool {
	return true
}
