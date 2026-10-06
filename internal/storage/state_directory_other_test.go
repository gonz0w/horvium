// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStateDirectoryUnavailableOffFreeBSD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	if s, err := OpenStateDirectory(path); s != nil || !errors.Is(err, ErrUnavailable) {
		t.Fatal("non-FreeBSD state opened", err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("unsupported opener created state")
	}
	if (&StateDirectory{}).Close() != nil || (*StateDirectory)(nil).Close() != nil {
		t.Fatal("inert handle did not close")
	}
}
