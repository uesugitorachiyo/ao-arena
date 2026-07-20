//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package arena

import (
	"os"
	"syscall"
)

func openReadNoFollow(path string, expected os.FileInfo) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(expected, info) {
		_ = file.Close()
		return nil, syscall.EINVAL
	}
	if err := syscall.SetNonblock(fd, false); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func openRootReadNoFollow(root *os.Root, name string, expected os.FileInfo) (*os.File, error) {
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(expected, info) {
		_ = file.Close()
		return nil, syscall.EINVAL
	}
	if err := syscall.SetNonblock(int(file.Fd()), false); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}
