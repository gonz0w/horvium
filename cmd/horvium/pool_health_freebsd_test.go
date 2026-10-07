// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/host/freebsd"
)

// Called only by the opted-in owned mirror fixture, after all assessment checks.
// The outer fixture verifies the exact pool GUID/backing before destruction.
func nativePoolHealth(t *testing.T, ctx context.Context, pool, guid, leaf string, must func(string, ...string) string, raw func() string, original string) {
	t.Helper()
	source, err := freebsd.NewPoolHealthSource()
	if err != nil {
		t.Fatal("native health unavailable", err)
	}
	collect := func(t *testing.T) freebsd.PoolHealthObservation {
		t.Helper()
		o, e := source.Collect(ctx, guid, pool)
		if e != nil || o.Pool.GUID != guid || o.Pool.LoadGUID == "" || o.Topology.PoolGUID != guid || len(o.Topology.Nodes) != 3 {
			t.Fatal("native health identity unavailable", e)
		}
		return o
	}
	t.Run("fresh-pool", func(t *testing.T) {
		o := collect(t)
		if o.Pool.Health.Effective != "ONLINE" || o.Topology.Health != "ONLINE" || o.Topology.ScanReported || o.Topology.Scan != "" || o.Scan.Kind != "unreported" || o.Scan.State != "unreported" || o.Scan.ProgressReported || o.Scan.ErrorsReported {
			t.Fatal("invented initial scan/health")
		}
	})
	for _, kind := range []string{"wrong-guid", "missing-pool"} {
		t.Run(kind, func(t *testing.T) {
			selectedGUID, selectedPool := guid, pool
			if kind == "wrong-guid" {
				selectedGUID = "1"
				if guid == "1" {
					selectedGUID = "2"
				}
			} else {
				selectedPool = pool + "_absent"
			}
			o, e := source.Collect(ctx, selectedGUID, selectedPool)
			if e == nil || !reflect.DeepEqual(o, freebsd.PoolHealthObservation{}) {
				t.Fatal("failed selection exposed facts", e)
			}
		})
	}
	// Progress can finish between probes. Retry unavailable/drifting observations
	// only inside a bounded test wait; the product performs no implicit retry.
	wait := func(t *testing.T, accept func(freebsd.PoolHealthObservation) bool) freebsd.PoolHealthObservation {
		t.Helper()
		waitCtx, stop := context.WithTimeout(ctx, 30*time.Second)
		defer stop()
		for waitCtx.Err() == nil {
			o, e := source.Collect(waitCtx, guid, pool)
			if e == nil && accept(o) {
				return o
			}
			select {
			case <-waitCtx.Done():
				t.Fatal("native health context expired")
			case <-time.After(100 * time.Millisecond):
			}
		}
		t.Fatal("native health observation did not converge")
		return freebsd.PoolHealthObservation{}
	}
	t.Run("scrub-completed", func(t *testing.T) {
		must("/sbin/zpool", "scrub", pool)
		o := wait(t, func(o freebsd.PoolHealthObservation) bool {
			return o.Scan.Kind == "scrub" && o.Scan.State == "completed"
		})
		if !o.Scan.ErrorsReported || o.Scan.Errors != 0 || o.Scan.EndedOn == "" || o.Scan.DurationDisplay == "" || o.Scan.ProgressReported {
			t.Fatal("incorrect native scan facts")
		}
	})
	t.Run("degraded", func(t *testing.T) {
		must("/sbin/zpool", "offline", pool, leaf)
		o := collect(t)
		offline := false
		for _, node := range o.Topology.Nodes {
			if node.Address == leaf && node.Health == "OFFLINE" {
				offline = true
			}
		}
		if o.Pool.Health.Effective != "DEGRADED" || o.Topology.Health != "DEGRADED" || !offline {
			t.Fatal("degraded pool reported healthy")
		}
		must("/sbin/zpool", "online", pool, leaf)
	})
	t.Run("restored-and-preserved", func(t *testing.T) {
		o := wait(t, func(o freebsd.PoolHealthObservation) bool {
			if o.Topology.Health != "ONLINE" {
				return false
			}
			for _, node := range o.Topology.Nodes {
				if node.Health != "ONLINE" {
					return false
				}
			}
			return true
		})
		if o.Topology.ReadErrors != 0 || o.Topology.WriteErrors != 0 || o.Topology.ChecksumErrors != 0 || o.Topology.DataErrors != 0 || raw() != original {
			t.Fatal("owned native state not restored")
		}
	})
}
