//go:build windows

package logger

import "golang.org/x/sys/windows"

func checkTerminalVT(fd int) bool {
	var mode uint32
	if err := windows.GetConsoleMode(windows.Handle(fd), &mode); err != nil {
		return false
	}
	return mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0
}
