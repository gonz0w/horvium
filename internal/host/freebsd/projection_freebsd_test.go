// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"

	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/helper"
)

// This fixture exists only in a separately compiled test binary. The product
// has no alternate inventory source or projection command.
func TestNativeProjectionEmitter(t *testing.T) {
	args := flag.Args()
	if len(args) != 1 {
		return
	}
	count, err := strconv.Atoi(args[0])
	if err != nil || count != 10000 {
		os.Exit(2)
	}
	for i := 0; i < count; i++ {
		fmt.Printf("fixture/dataset-%06d\t%d\t%d\t1099511627776\t%d\n", i, i+100, i, i)
	}
	os.Exit(0)
}

func TestNativeProjectionHelper(t *testing.T) {
	args := flag.Args()
	if len(args) != 1 {
		return
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("projection fixture requires explicit disposable performance guest")
	}
	c, err := admin.LoadConfig(args[0])
	if err != nil {
		t.Fatal("fixture config")
	}
	source, err := New(c.InventoryLimits())
	if err != nil {
		t.Fatal("actual native source")
	}
	source.read = func(ctx context.Context, p probe, limit int) (readOutput, error) {
		if p == datasets {
			return executeNative(ctx, os.Args[0], []string{"-test.run=^TestNativeProjectionEmitter$", "--", "10000"}, limit)
		}
		return readNative(ctx, p, limit)
	}
	server, err := helper.NewServer(c.HelperConfig(), source)
	if err != nil {
		t.Fatal("kernel-authorized fixture helper")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	if server.Serve(ctx) != nil && ctx.Err() == nil {
		t.Fatal("projection fixture helper exit")
	}
}
