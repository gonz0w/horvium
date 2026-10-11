//go:build freebsd && horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

type policyCostReads struct {
	count   int
	elapsed time.Duration
}

func (r *policyCostReads) measure(run func() (readOutput, error)) (readOutput, error) {
	start := time.Now()
	out, err := run()
	r.count++
	r.elapsed += time.Since(start)
	return out, err
}

// Diagnostic reader wrappers forward every original result; they never supply
// fixed-factory qualification, reusable facts or an execution acknowledgement.
func TestNativePolicyStartupCosts(t *testing.T) {
	if os.Getenv("HORVIUM_POLICY_STARTUP_DIAGNOSTICS_VM") != "1" {
		t.Skip("owned diagnostic fixture only")
	}
	requirePolicyFactsVM(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	out, e := execute(ctx, "/sbin/sysctl", []string{"-n", "hw.ncpu"}, 4096)
	cancel()
	if e != nil || string(out.data) != "1\n" {
		t.Fatal("declared serial diagnostic kernel")
	}
	nativeMultiplePolicyFactsFixture(t, func(ctx context.Context, policies []storage.Policy, disk inventory.Provider) {
		stage := func(name string, reads *policyCostReads, run func() error) {
			t.Helper()
			start := time.Now()
			e := run()
			elapsed := time.Since(start)
			if e != nil {
				t.Fatalf("diagnostic stage %s failed: %v", name, e)
			}
			fmt.Printf("POLICY_STARTUP_COST stage=%s reads=%d elapsed_ns=%d read_ns=%d\n", name, reads.count, elapsed.Nanoseconds(), reads.elapsed.Nanoseconds())
		}
		for _, mode := range []string{"direct", "guarded"} {
			reads := new(policyCostReads)
			stage(mode+"-launch", reads, func() error {
				for i := 0; i < 8; i++ {
					c, stop := context.WithTimeout(ctx, 5*time.Second)
					out, e := reads.measure(func() (readOutput, error) {
						if mode == "guarded" {
							return executeNative(c, "/usr/bin/true", nil, 4096)
						}
						return execute(c, "/usr/bin/true", nil, 4096)
					})
					stop()
					if e != nil {
						return e
					}
					if len(out.data) != 0 || out.consumed != 0 {
						return storage.ErrDrift
					}
				}
				return nil
			})
		}
		system, e := NewSystemUseSource()
		if e != nil {
			t.Fatal(e)
		}
		reads := new(policyCostReads)
		system.read = func(c context.Context, p systemProbe, name string, limit int) (readOutput, error) {
			return reads.measure(func() (readOutput, error) { return readSystemNative(c, p, name, limit) })
		}
		stage("system", reads, func() error { _, e := system.Collect(ctx); return e })
		for i, p := range policies {
			name := []string{"first", "second"}[i]
			pool, e := NewPoolPropertySource()
			if e != nil {
				t.Fatal(e)
			}
			properties := new(policyCostReads)
			pool.read = func(c context.Context, probe poolProbe, name string, limit int) (readOutput, error) {
				return properties.measure(func() (readOutput, error) { return readPoolNative(c, probe, name, limit) })
			}
			var observation storage.Observation
			stage(name+"-properties", properties, func() error {
				var e error
				observation, e = pool.Collect(ctx, p.PoolGUID, p.PoolName)
				return e
			})
			providers, e := NewPoolProviderSource()
			if e != nil {
				t.Fatal(e)
			}
			members := new(policyCostReads)
			providers.source.read = func(c context.Context, probe systemProbe, name string, limit int) (readOutput, error) {
				return members.measure(func() (readOutput, error) { return readSystemNative(c, probe, name, limit) })
			}
			stage(name+"-providers", members, func() error {
				disks, e := providers.Collect(ctx, observation)
				if e != nil {
					return e
				}
				return storage.ValidatePropertyMembers(p, observation, disks)
			})
		}
		for _, kind := range []string{"ufs", "zfs"} {
			domains, mounts := nativeMultiplePolicyMetadataFixture(t, ctx, kind, disk, func(j, _ string) {
				f, e := os.Open(j)
				if e != nil {
					t.Fatal(e)
				}
				defer f.Close()
				// The original fixed collector remains opaque: no reader hook.
				stage(kind+"-backing", new(policyCostReads), func() error {
					b, e := collectNativePersistentBacking(ctx, int(f.Fd()))
					if e != nil {
						return e
					}
					if b.kind != kind {
						return storage.ErrDrift
					}
					return nil
				})
			})
			if !domains || !mounts {
				t.Fatal("diagnostic metadata teardown", kind)
			}
			fmt.Printf("POLICY_STARTUP_COST_CLEANUP %s filesystems=pass directories=pass mounts=pass\n", kind)
		}
	})
}
