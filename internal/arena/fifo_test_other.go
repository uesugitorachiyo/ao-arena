//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package arena

import "testing"

func makeFIFOForTest(t *testing.T, _ string) {
	t.Helper()
	t.Skip("FIFO creation is unavailable on this platform")
}
