// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

func testOutput(value string) readOutput {
	return readOutput{data: []byte(value), consumed: len(value)}
}

func fixtureRead(_ context.Context, probe probe, _ int) (readOutput, error) {
	switch probe {
	case versions:
		return testOutput("15.1-RELEASE-p3\n15.1-RELEASE-p2\n15.1-RELEASE-p3\n"), nil
	case architecture:
		return testOutput("amd64\n"), nil
	case memory:
		return testOutput("17179869184\n"), nil
	case zfsVersion:
		return testOutput("zfs-2.4.2-1-FreeBSD_g6330a45b0\nzfs-kmod-2.4.2-1-FreeBSD_g6330a45b0\n"), nil
	case geom:
		return testOutput("<mesh/>\n"), nil
	case pools:
		return testOutput("tank\t1\t1000\t400\t600\tONLINE\n"), nil
	case datasets:
		return testOutput("tank/data\t2\t0\t600\t0\n"), nil
	default:
		return readOutput{}, ErrUnavailable
	}
}

func testSource(read reader) *Source {
	return &Source{limits: inventory.DefaultLimits(), read: read, commandTimeout: time.Second, totalTimeout: time.Second}
}

func TestCompleteObservation(t *testing.T) {
	// M1-NATIVE-001 component portion: distinguish installed/running versions.
	result := testSource(fixtureRead).Collect(context.Background())
	if !result.Complete() || result.Host.Data.InstalledKernel != "15.1-RELEASE-p3" || result.Host.Data.RunningKernel != "15.1-RELEASE-p2" || result.Host.Data.PhysicalMemory == nil || *result.Host.Data.PhysicalMemory != 17179869184 {
		t.Fatalf("wrong host result: %+v", result.Host)
	}
	if len(result.Datasets.Data) != 1 || result.Pools.Data[0].GUID != "1" {
		t.Fatal("inventory missing")
	}
	if result.StartedAt.IsZero() || result.Host.FinishedAt.Before(result.Host.StartedAt) || result.FinishedAt.Before(result.StartedAt) {
		t.Fatal("missing collection provenance")
	}
}

func TestPartialObservationIsHonest(t *testing.T) {
	// M1-FAIL-001: failure is not healthy/empty/zero, and stderr is not an error UI.
	for _, failure := range []error{ErrUnavailable, ErrPermission, ErrExit, ErrLimit, context.DeadlineExceeded} {
		t.Run(failure.Error(), func(t *testing.T) {
			source := testSource(func(ctx context.Context, p probe, max int) (readOutput, error) {
				if p == pools || p == memory {
					return testOutput("sensitive native output"), failure
				}
				return fixtureRead(ctx, p, max)
			})
			result := source.Collect(context.Background())
			if result.Complete() || result.Pools.Status != inventory.Unavailable || len(result.Pools.Data) != 0 || len(result.Pools.Problems) != 1 {
				t.Fatalf("failure became empty success: %+v", result.Pools)
			}
			if result.Host.Status != inventory.Partial || result.Host.Data.PhysicalMemory != nil || result.Host.Data.Architecture != "amd64" {
				t.Fatal("lost partial host provenance")
			}
			if result.Datasets.Status != inventory.OK || result.Pools.Problems[0].Code == "sensitive native output" {
				t.Fatal("error leaked or unrelated source lost")
			}
		})
	}
}

func TestSourceValidatesHostOutput(t *testing.T) {
	for _, p := range []probe{versions, architecture, memory, zfsVersion} {
		t.Run(string(p), func(t *testing.T) {
			result := testSource(func(ctx context.Context, item probe, max int) (readOutput, error) {
				if p == item {
					return testOutput("bogus\x00\n"), nil
				}
				return fixtureRead(ctx, item, max)
			}).Collect(context.Background())
			if result.Host.Status != inventory.Partial || len(result.Host.Problems) != 1 || result.Host.Problems[0].Code != "malformed" {
				t.Fatalf("invalid host source accepted: %+v", result.Host)
			}
		})
	}
}

func TestCollectionBudgetAndCancellation(t *testing.T) {
	source := testSource(fixtureRead)
	source.limits.MaxBytes = 80
	result := source.Collect(context.Background())
	if result.Complete() || result.GEOM.Status != inventory.Unavailable || result.GEOM.Problems[0].Code != "limit" {
		t.Fatal("aggregate budget ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	source = testSource(func(context.Context, probe, int) (readOutput, error) { calls++; return readOutput{}, nil })
	result = source.Collect(ctx)
	if calls != 0 || result.Complete() || result.Host.Status != inventory.Unavailable {
		t.Fatal("cancelled collection ran work or became fresh")
	}
	if _, err := New(inventory.Limits{}); !errors.Is(err, inventory.ErrInvalidLimits) {
		t.Fatalf("invalid limits accepted: %v", err)
	}
}

func TestBudgetCountsDiscardedErrorOutput(t *testing.T) {
	calls := 0
	source := testSource(func(_ context.Context, _ probe, max int) (readOutput, error) {
		calls++
		return readOutput{consumed: max}, ErrExit
	})
	result := source.Collect(context.Background())
	if calls != 1 || result.GEOM.Problems[0].Code != "limit" {
		t.Fatal("discarding error output reset the aggregate safety budget")
	}
}
