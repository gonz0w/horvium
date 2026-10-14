// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDashboardWitnessSamplerOriginalReadback(t *testing.T) {
	name := filepath.Join(t.TempDir(), "original")
	if os.WriteFile(name, []byte("original sample frames"), 0400) != nil {
		t.Fatal("fixture")
	}
	f, e := os.Open(name)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	data, e := witnessSamplerReadOriginal(f)
	if e != nil || string(data) != "original sample frames" {
		t.Fatal("sealed data file rejected as executable", e)
	}
	if os.Chmod(name, 0600) != nil {
		t.Fatal("fixture mode")
	}
	if _, e := witnessSamplerReadOriginal(f); e == nil {
		t.Fatal("mutable originals accepted")
	}
	f.Close()
	if _, e := witnessSamplerReadOriginal(f); e == nil {
		t.Fatal("closed original accepted")
	}
}
