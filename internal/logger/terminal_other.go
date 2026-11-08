//go:build !windows

package logger

// checkTerminalVT reports whether virtual terminal processing is available (always true on non-Windows platforms).
func checkTerminalVT(_ int) bool {
	return true
}
