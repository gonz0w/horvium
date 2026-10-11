//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"github.com/gonz0w/horvium/internal/inventory"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func policyStorageInputs(t *testing.T, f restoreFixture) (Policy, Observation, []inventory.DiskObservation, PoolSystemUse) {
	p, o, d, _ := enrollmentFixture(t)
	u := activationSystemFixture(t, f.journal, f.ledger, f.read)
	for i := range u.pools {
		if u.pools[i].GUID == p.PoolGUID {
			u.pools[i].Disks = []inventory.Provider{d[0].Provider, d[1].Provider}
		}
	}
	return p, o, d, u
}
func policyStorageSeed(t *testing.T, f restoreFixture, name string) (string, string, Policy, ExecutionStartupPolicySource, PoolSystemUse) {
	t.Helper()
	p, o, d, u := policyStorageInputs(t, f)
	j, l := policyOnlyStorageSeed(t, f, name, []Policy{p}, u)
	source := func(context.Context, Policy) (Observation, []inventory.DiskObservation, PoolSystemUse, error) {
		return o, d, u, nil
	}
	return j, l, p, source, u
}
func policyOnlyStorageSeed(t *testing.T, f restoreFixture, name string, policies []Policy, u PoolSystemUse) (string, string) {
	t.Helper()
	ctx := context.Background()
	j, l := filepath.Join(f.journal, name), filepath.Join(f.ledger, name)
	a, e := activateFreshExecutionJournal(ctx, j, l, f.read, func(context.Context) (PoolSystemUse, error) { return u, nil }, activationBoot)
	if e != nil {
		t.Fatal("fresh seed", e)
	}
	defer a.fresh.close()
	pair := a.fresh.pair
	for _, p := range policies {
		if e = pair.publishJournaledOwnership(ctx, p, 0, f.read, nil); e != nil {
			t.Fatal("original policy publisher", e)
		}
	}
	closed := pair.native.journal.ack
	closed.Generation++
	closed.Owner.State = "closed"
	if e = pair.native.journal.publish(closed, nil); e != nil {
		t.Fatal("controlled closed publisher", e)
	}
	if e = pair.Close(); e != nil {
		t.Fatal(e)
	}
	return j, l
}
func policyJobStorageSeed(t *testing.T, f restoreFixture, name string, data []byte) (string, string) {
	t.Helper()
	return policyHistoryStorageSeed(t, f, name, nil, [][]byte{data})
}
func policyHistoryStorageSeed(t *testing.T, f restoreFixture, name string, policies []Policy, records [][]byte) (string, string) {
	// Denial-only metadata fixture. Strict real job publishers establish history
	// before original journal initialization/close; no worker ran or clean runtime
	// shutdown is asserted. Never write an encoded record directly to disk.
	t.Helper()
	ctx := context.Background()
	j, l := filepath.Join(f.journal, name), filepath.Join(f.ledger, name)
	for _, path := range []string{j, l} {
		if os.Mkdir(path, 0700) != nil || os.WriteFile(filepath.Join(path, stateWriterName), nil, 0600) != nil {
			t.Fatal("owned job domains")
		}
	}
	store, e := openExecutionJournalStore(j)
	if e != nil {
		t.Fatal(e)
	}
	defer store.close()
	ledger, e := openStateDirectory(l, false)
	if e != nil {
		t.Fatal(e)
	}
	defer ledger.Close()
	for _, p := range policies {
		if e = ledger.PublishOwnership(p, 0); e != nil {
			t.Fatal("strict history ownership publisher", e)
		}
	}
	for _, data := range records {
		job, e := decodeCatalogJob(data)
		if e != nil {
			t.Fatal(e)
		}
		prop, pe := DecodePropertyJob(data)
		for i := range job.History {
			prefix := job
			prefix.History = append([]JobTransition(nil), job.History[:i+1]...)
			if pe == nil {
				bound, e := BindPropertyJob(prefix, prop.Intent())
				if e != nil {
					t.Fatal(e)
				}
				e = ledger.PublishPropertyJob(bound, uint64(i))
				if e != nil {
					t.Fatal(e)
				}
			} else if e = ledger.PublishJob(prefix, uint64(i)); e != nil {
				t.Fatal(e)
			}
		}
	}
	pair := &ExecutionDomainLeases{native: &executionDomainLeaseNative{store, ledger}}
	pair.origin = pair
	got, e := pair.Observe(ctx, strings.Repeat("1", 32), f.read)
	if e != nil {
		t.Fatal(e)
	}
	record := executionJournalFixture()
	record.Binding = got.binding
	record.Census = got.census
	if e = store.initialize(record, nil); e != nil {
		t.Fatal(e)
	}
	record.Generation++
	record.Owner.State = "closed"
	if e = store.publish(record, nil); e != nil {
		t.Fatal(e)
	}
	if e = pair.Close(); e != nil {
		t.Fatal(e)
	}
	return j, l
}
func TestNativePolicyStartupUFS(t *testing.T) { nativePolicyStartupStorage(t, "ufs") }
func TestNativePolicyStartupZFS(t *testing.T) { nativePolicyStartupStorage(t, "zfs") }
func nativePolicyStartupStorage(t *testing.T, kind string) {
	restartStorageGuard(t)
	f := newRestoreFixture(t, kind)
	ctx := context.Background()
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("policy storage", name)
		}
		for _, s := range seen {
			if s == name+"=pass" {
				t.Fatal("duplicate")
			}
		}
		seen = append(seen, name+"=pass")
	}
	original := func(name string) (*ExecutionStartupReview, *executionStartupNative, *ExecutionRestartActivation, string, string) {
		j, l, _, source, u := policyStorageSeed(t, f, name)
		r, n, e := reviewExecutionStartupPolicy(ctx, j, l, f.read, func(context.Context) (PoolSystemUse, error) { return u, nil }, activationBoot, source)
		if e != nil {
			t.Fatal("original policy review", e)
		}
		defer func() {
			if r != nil {
				_ = r.Close()
			}
		}()
		a, e := r.Activate(ctx)
		if e != nil {
			t.Fatal("original policy activation", e)
		}
		t.Cleanup(func() { _ = a.Close() })
		return r, n, a, j, l
	}
	r, n, a, j, l := original("positive")
	defer a.Close()
	ledgerBefore := freshNamespaceSnapshot(t, l)
	before := n.snapshot.record
	check("policy_bound", len(n.policyDigests) == 1)
	check("leases_held", restartLocksHeld(t, j, l))
	_, e := a.Observe(ctx)
	check("original_observation", e == nil)
	check("no_old_release", r.Close() != nil && restartLocksHeld(t, j, l))
	check("original_close", a.FinishIdle(ctx) == nil)
	got, e := n.pair.native.journal.read(nil)
	check("exact_closed", e == nil && got.Generation == before.Generation+1 && got.Owner.State == "closed" && got.Census == before.Census && got.Binding == before.Binding && got.Owner.ID == before.Owner.ID)
	check("exact_ledger", reflect.DeepEqual(ledgerBefore, freshNamespaceSnapshot(t, l)))
	_, e = a.Observe(ctx)
	check("finish_retired", e != nil && a.FinishIdle(ctx) != nil)
	check("original_release", a.Close() == nil && restartLocksReleased(t, j, l))
	// The ordinary metadata path must not acquire the policy-aware exception.
	old, _, e := reviewExecutionStartup(ctx, j, l, f.read, func(context.Context) (PoolSystemUse, error) { return n.system(ctx) }, activationBoot)
	check("ordinary_metadata_only", e == nil && old != nil)
	oldActivation, e := old.Activate(ctx)
	check("ordinary_activation_denied", e != nil && oldActivation == nil && old.Close() == nil)
	for _, mode := range []string{"collector", "alias", "cancel", "legacy", "scope", "member", "missing-system", "context-deadline"} {
		_, nn, aa, jj, ll := original("observe-" + mode)
		c, stop := context.WithCancel(ctx)
		calls := 0
		native := nn.policy
		nn.policy = func(c context.Context, p Policy) (Observation, []inventory.DiskObservation, PoolSystemUse, error) {
			calls++
			o, d, u, e := native(c, p)
			switch mode {
			case "collector":
				return Observation{}, nil, PoolSystemUse{}, ErrUnavailable
			case "alias":
				p.Members[0].Serial = "CHANGED"
				d = append([]inventory.DiskObservation(nil), d...)
				d[0].Provider.Identifiers = []inventory.Identifier{{Kind: "serial", Value: "CHANGED", Source: p.Members[0].SerialSource}}
			case "cancel":
				stop()
			case "legacy":
				d = nil
			case "scope":
				o.Filesystems = append([]Filesystem(nil), o.Filesystems...)
				o.Filesystems[0].GUID = "999"
			case "member":
				d = d[:1]
			case "missing-system":
				u = PoolSystemUse{}
			case "context-deadline":
				return Observation{}, nil, PoolSystemUse{}, context.DeadlineExceeded
			}
			return o, d, u, e
		}
		bj, bl := freshNamespaceSnapshot(t, jj), freshNamespaceSnapshot(t, ll)
		_, e = aa.Observe(c)
		check(mode+"_denied", e != nil && calls > 0)
		_, e = aa.Observe(ctx)
		check(mode+"_sticky", e != nil && calls == 1)
		check(mode+"_preserved", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, jj)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, ll)))
		check(mode+"_released", aa.Close() == nil && restartLocksReleased(t, jj, ll))
		stop()
	}
	for _, stage := range []string{"activate", "finish"} {
		for _, point := range []string{"syncFile", "rename", "syncDirectory", "post-policy", "post-directory"} {
			mode := stage + "-" + point
			jj, ll, _, source, u := policyStorageSeed(t, f, "cut-"+mode)
			rr, nn, e := reviewExecutionStartupPolicy(ctx, jj, ll, f.read, func(context.Context) (PoolSystemUse, error) { return u, nil }, activationBoot, source)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { _ = rr.Close() })
			wrap := func(tx ownershipPublication) ownershipPublication {
				switch point {
				case "syncFile":
					tx.syncFile = func() error { return ErrUnavailable }
				case "rename":
					tx.rename = func() error { return ErrUnavailable }
				case "syncDirectory":
					tx.syncDirectory = func() error { return ErrUnavailable }
				case "post-directory":
					old := tx.syncDirectory
					tx.syncDirectory = func() error {
						if e := old(); e != nil {
							return e
						}
						return os.Chmod(jj, 0700)
					}
				}
				return tx
			}
			native := nn.policy
			finishing := false
			nn.policy = func(c context.Context, p Policy) (Observation, []inventory.DiskObservation, PoolSystemUse, error) {
				current, e := nn.pair.native.journal.inspectLocked("", nil)
				if e != nil {
					return Observation{}, nil, PoolSystemUse{}, e
				}
				changed := current.record.Owner.State == "active"
				if stage == "finish" {
					changed = finishing && current.record.Owner.State == "closed"
				}
				if point == "post-policy" && changed {
					return Observation{}, nil, PoolSystemUse{}, ErrUnavailable
				}
				return native(c, p)
			}
			var aa *ExecutionRestartActivation
			if stage == "activate" {
				rr.activate = func(c context.Context) (*ExecutionRestartActivation, error) {
					return nn.activate(c, strings.NewReader(strings.Repeat("x", 16)), wrap)
				}
				aa, e = rr.Activate(ctx)
			} else {
				aa, e = rr.Activate(ctx)
				if e != nil {
					t.Fatal(e)
				}
				finishing = true
				aa.finish = func(c context.Context) error { return nn.finishIdle(c, wrap) }
				e = aa.FinishIdle(ctx)
			}
			check(mode+"_denied", e != nil)
			check(mode+"_retired", nn.pair.blocked && nn.pair.native.journal.failed && nn.pair.native.ledger.publicationBlocked)
			bj, bl := freshNamespaceSnapshot(t, jj), freshNamespaceSnapshot(t, ll)
			if aa != nil {
				check(mode+"_retry_preserved", aa.FinishIdle(ctx) != nil && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, jj)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, ll)))
				check(mode+"_released", aa.Close() == nil && restartLocksReleased(t, jj, ll))
			} else {
				_, e = rr.Activate(ctx)
				check(mode+"_retry_preserved", e != nil && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, jj)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, ll)))
				check(mode+"_released", rr.Close() == nil && restartLocksReleased(t, jj, ll))
			}
		}
	}
	for _, format := range []string{"legacy", "property"} {
		for _, tc := range []struct {
			state  JobState
			reason string
		}{{JobQueued, "admitted"}, {JobApplying, "launch-intent"}, {JobObserving, "native-observation"}, {JobSucceeded, "verified"}, {JobFailed, "verified-failure"}, {JobCancelled, "operator-cancelled"}, {JobReconcileRequired, "review-expired"}} {
			job := jobFixture()
			property := propertyJobFixture(t)
			if format == "property" {
				job = property.Job()
			}
			job = restartJobFixture(t, job, tc.state, JobQueued, tc.reason)
			var data []byte
			var e error
			if format == "property" {
				bound, e := BindPropertyJob(job, property.Intent())
				if e != nil {
					t.Fatal(e)
				}
				data, e = EncodePropertyJob(bound)
			} else {
				data, e = EncodeJobRecord(job)
			}
			if e != nil {
				t.Fatal(e)
			}
			name := format + "-" + string(tc.state)
			jj, ll := policyJobStorageSeed(t, f, "job-"+name, data)
			u := activationSystemFixture(t, f.journal, f.ledger, f.read)
			calls := 0
			source := func(context.Context, Policy) (Observation, []inventory.DiskObservation, PoolSystemUse, error) {
				calls++
				return Observation{}, nil, PoolSystemUse{}, ErrDenied
			}
			bj, bl := freshNamespaceSnapshot(t, jj), freshNamespaceSnapshot(t, ll)
			rr, _, e := reviewExecutionStartupPolicy(ctx, jj, ll, f.read, func(context.Context) (PoolSystemUse, error) { return u, nil }, activationBoot, source)
			check(name+"_denied", e != nil && rr == nil && calls == 0)
			check(name+"_preserved", reflect.DeepEqual(bj, freshNamespaceSnapshot(t, jj)) && reflect.DeepEqual(bl, freshNamespaceSnapshot(t, ll)))
			read, e := os.ReadFile(filepath.Join(ll, "job-"+job.ID+".json"))
			holds, he := BuildJobReservations([][]byte{read})
			check(name+"_holds", e == nil && he == nil && bytes.Equal(read, data) && len(holds.Pools()) == 1 && len(holds.Pools()[0].Jobs) == 1)
			check(name+"_released", restartLocksReleased(t, jj, ll))
		}
	}
	nativePolicyHistoryCases(t, f, check)
	nativeMultiplePolicyCases(t, f, check)
	t.Log("POLICY_STARTUP_STORAGE_" + strings.ToUpper(kind) + "_RESULT " + strings.Join(seen, " "))
}
