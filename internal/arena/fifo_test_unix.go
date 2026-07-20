//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package arena

import (
	"syscall"
	"testing"
)

func makeFIFOForTest(t *testing.T, path string) {
	t.Helper()
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
}
