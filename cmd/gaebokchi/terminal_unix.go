//go:build unix

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func terminalSize(file *os.File) (int, int, error) {
	size, err := unix.IoctlGetWinsize(int(file.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(size.Col), int(size.Row), nil
}
