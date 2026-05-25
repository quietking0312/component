//go:build windows

package mconsole

import (
	"os"

	"golang.org/x/sys/windows"
)

func isTerminal(f *os.File) bool {
	fd := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(fd, &mode); err != nil {
		return false
	}
	// 启用 Virtual Terminal Processing（ANSI 支持）
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
		_ = windows.SetConsoleMode(fd, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
	return true
}
