//go:build !windows

package logger

func checkTerminalVT(_ int) bool {
	return true
}
