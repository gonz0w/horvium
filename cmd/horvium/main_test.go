// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNoArgumentCredentialDisclosure(t *testing.T) {
	for _, args := range [][]string{{"unknown-secret"}, {"inventory", "-token", "secret"}, {"serve", "-config", "relative"}} {
		var out, errout bytes.Buffer
		if run(context.Background(), args, strings.NewReader(""), &out, &errout) == 0 {
			t.Fatal("invalid command accepted")
		}
		if strings.Contains(errout.String(), "secret") || strings.Contains(out.String(), "secret") {
			t.Fatal("argument disclosed")
		}
	}
}
func TestTokenInputBoundaries(t *testing.T) {
	for _, s := range []string{"", strings.Repeat("a", 63), strings.Repeat("A", 64), strings.Repeat("a", 65), strings.Repeat("a", 64) + "\nextra"} {
		if _, e := readToken(strings.NewReader(s)); e == nil {
			t.Fatal("malformed input accepted")
		}
	}
	for _, s := range []string{strings.Repeat("a", 64), strings.Repeat("a", 64) + "\n"} {
		if _, e := readToken(strings.NewReader(s)); e != nil {
			t.Fatal("protected token rejected")
		}
	}
}

func TestProtectedInputPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if os.WriteFile(path, []byte("fixture"), 0600) != nil {
		t.Fatal("fixture")
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if !protectedInput(f) {
		t.Fatal("owned protected regular input rejected")
	}
	if os.Chmod(path, 0644) != nil {
		t.Fatal("fixture chmod")
	}
	if protectedInput(f) {
		t.Fatal("public credential input accepted")
	}
	r, w, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	defer w.Close()
	if !protectedInput(r) {
		t.Fatal("protected pipe rejected")
	}
}
func TestCancelledCLIClosesBlockedInput(t *testing.T) {
	r, w, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	defer w.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- run(ctx, []string{"inventory"}, r, io.Discard, io.Discard) }()
	cancel()
	select {
	case code := <-done:
		if code == 0 {
			t.Fatal("cancelled input succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled CLI blocked on token input")
	}
}
