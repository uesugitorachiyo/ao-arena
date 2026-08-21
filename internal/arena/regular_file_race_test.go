//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package arena

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStrictJSONReaderDoesNotBlockWhenRegularFileIsReplacedByFIFO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		var target map[string]any
		_, err := readStrictBoundedJSONWithHook(path, "race input", 1024, &target, func() error {
			if err := os.Remove(path); err != nil {
				return err
			}
			return syscall.Mkfifo(path, 0o600)
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "regular non-link") {
			t.Fatalf("replacement race error = %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		unblock, openErr := os.OpenFile(path, os.O_RDWR, 0)
		if openErr == nil {
			_ = unblock.Close()
		}
		t.Fatal("replacement FIFO blocked the JSON reader")
	}
}

func TestRootedStrictJSONReaderRejectsAncestorSymlinkSwap(t *testing.T) {
	base := t.TempDir()
	evidenceDir := filepath.Join(base, "evidence")
	if err := os.Mkdir(evidenceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "result.json"), []byte("{\"trusted\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	attacker := t.TempDir()
	if err := os.WriteFile(filepath.Join(attacker, "result.json"), []byte("{\"trusted\":false}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	var target map[string]any
	_, err = readStrictBoundedJSONFromRootWithHook(root, "evidence/result.json", "race sidecar", 1024, &target, func() error {
		if err := os.Rename(evidenceDir, filepath.Join(base, "evidence-original")); err != nil {
			return err
		}
		requireTestSymlink(t, attacker, evidenceDir)
		return nil
	})
	if err == nil {
		t.Fatalf("ancestor swap accepted attacker sidecar: %#v", target)
	}
}
