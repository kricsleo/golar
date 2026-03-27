// Source: https://github.com/microsoft/typescript-go/blob/2f9b360a6f4d43e5f8ebdd5cf01e38ecbcff7dae/cmd/tsgo/enablevtprocessing_windows.go

package main

import (
	"golang.org/x/sys/windows"
)

func init() {
	h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil || h == windows.InvalidHandle {
		return
	}
	fileType, err := windows.GetFileType(h)
	if err != nil || fileType == windows.FILE_TYPE_CHAR {
		var mode uint32
		if err := windows.GetConsoleMode(h, &mode); err != nil {
			return
		}
		if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
			_ = windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
		}
	}
}
