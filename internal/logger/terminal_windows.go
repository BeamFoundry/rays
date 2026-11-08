//go:build windows

package logger

import "golang.org/x/sys/windows"

// checkTerminalVT verifies and enables virtual terminal processing on Windows console handles.
func checkTerminalVT(fd int) bool {
	var mode uint32
	h := windows.Handle(fd)
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	return windows.SetConsoleMode(h, mode) == nil
}
