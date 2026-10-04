//go:build freebsd || darwin || linux

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestNamedFIFORequiresProtectedOwnershipAndMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.fifo")
	if syscall.Mkfifo(path, 0600) != nil {
		t.Fatal("fixture FIFO")
	}
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if !protectedInput(f) {
		t.Fatal("owned private FIFO rejected")
	}
	if os.Chmod(path, 0666) != nil {
		t.Fatal("fixture chmod")
	}
	if protectedInput(f) {
		t.Fatal("public named FIFO accepted as protected input")
	}
}
