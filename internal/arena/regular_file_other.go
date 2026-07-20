//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package arena

import "os"

func openReadNoFollow(path string, expected os.FileInfo) (*os.File, error) {
	file, err := os.Open(path)
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
		return nil, os.ErrInvalid
	}
	return file, nil
}

func openRootReadNoFollow(root *os.Root, name string, expected os.FileInfo) (*os.File, error) {
	file, err := root.Open(name)
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
		return nil, os.ErrInvalid
	}
	return file, nil
}
