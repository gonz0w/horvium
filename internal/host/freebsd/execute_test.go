// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Re-exec the test binary as a deterministic child; production paths are fixed.
func TestCommandProcess(t *testing.T) {
	if len(flag.Args()) != 1 {
		return
	}
	if os.Getenv("LC_ALL") != "C" || os.Getenv("HORVIUM_TEST_SECRET") != "" {
		os.Exit(2)
	}
	switch flag.Args()[0] {
	case "ok":
		fmt.Print("safe\n")
	case "overflow":
		fmt.Print(strings.Repeat("x", 1024*1024))
	case "stderr":
		fmt.Fprint(os.Stderr, strings.Repeat("sensitive", 1024*1024))
	case "hang":
		time.Sleep(10 * time.Second)
	case "exit":
		fmt.Fprint(os.Stderr, "sensitive failure")
		os.Exit(2)
	}
	os.Exit(0)
}

func TestCommandBoundsAndErrors(t *testing.T) {
	t.Setenv("HORVIUM_TEST_SECRET", "must not reach native tools")
	for _, item := range []struct {
		mode string
		want error
	}{
		{"ok", nil}, {"overflow", ErrLimit}, {"stderr", ErrLimit}, {"exit", ErrExit}, {"hang", context.DeadlineExceeded},
	} {
		t.Run(item.mode, func(t *testing.T) {
			// Race-instrumented child binaries take extra time at process exit.
			timeout := 5 * time.Second
			if item.mode == "hang" {
				timeout = 300 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			out, err := execute(ctx, os.Args[0], []string{"-test.run=^TestCommandProcess$", "--", item.mode}, 128)
			if !errors.Is(err, item.want) {
				t.Fatalf("wanted %v, got %v", item.want, err)
			}
			if item.want == nil && string(out.data) != "safe\n" {
				t.Fatalf("output: %q", out.data)
			}
			if item.want != nil && len(out.data) != 0 {
				t.Fatal("failed command published output")
			}
		})
	}
	if _, err := execute(context.Background(), "/nonexistent/horvium-probe", nil, 128); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing tool: %v", err)
	}
}
