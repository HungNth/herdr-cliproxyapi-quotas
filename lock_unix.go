//go:build !windows

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}

func unlockFile(file *os.File) {
	unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
