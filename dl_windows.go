//go:build windows

package reparos

import "syscall"

func openLibrary(path string) (uintptr, error) {
	h, err := syscall.LoadLibrary(path)
	return uintptr(h), err
}
