// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

type multiplePolicyCapture struct {
	calls                []string
	deadline             time.Time
	deadlineOK, leasesOK bool
}

func (c *multiplePolicyCapture) complete() bool {
	if len(c.calls) < 2 || len(c.calls)%2 != 0 {
		return false
	}
	for i, guid := range c.calls {
		if guid != []string{"100", "200"}[i%2] {
			return false
		}
	}
	return true
}

func nativeMultiplePolicyCases(t *testing.T, f restoreFixture, check func(string, bool)) {
	t.Helper()
	ctx := context.Background()
	p, o, d, use := policyStorageInputs(t, f)
	q, other, disks := secondPolicyFixture(t)
	use.pools = append(use.pools, ImportedPoolUse{Name: q.PoolName, GUID: q.PoolGUID, Health: "ONLINE", Disks: []inventory.Provider{disks[0].Provider, disks[1].Provider}})
	use.topologies = append(use.topologies, other.Topology)
	for _, v := range []struct {
		p Policy
		o Observation
		d []inventory.DiskObservation
	}{{p, o, d}, {q, other, disks}} {
		if err := checkExecutionStartupPolicyFacts(v.p, v.o, v.d, use); err != nil {
			t.Fatal("valid supplied multiple-policy facts", err)
		}
	}
	want := newStateCatalogBuilder(defaultStateCatalogLimits)
	for _, policy := range []Policy{p, q} {
		data, err := EncodeOwnershipRecord(policy)
		if err != nil {
			t.Fatal(err)
		}
		if err = want.add(stateRecordName{poolGUID: policy.PoolGUID}, data); err != nil {
			t.Fatal(err)
		}
	}
	readRecord := func(j string) executionJournal {
		data, err := os.ReadFile(filepath.Join(j, executionJournalName))
		if err != nil {
			t.Fatal(err)
		}
		record, err := decodeExecutionJournal(data)
		if err != nil {
			t.Fatal(err)
		}
		return record
	}
	diagnostic := func(j, l string) bool {
		pair, err := OpenExecutionDomainLeases(ctx, j, l)
		if err != nil {
			return false
		}
		catalog, ce := pair.native.ledger.InspectState()
		holds, he := pair.native.ledger.InspectJobReservations()
		record, je := pair.native.journal.read(nil)
		closed := pair.Close()
		return ce == nil && he == nil && je == nil && closed == nil && reflect.DeepEqual(catalog, want.catalog) && len(holds.Pools()) == 0 && record == readRecord(j)
	}
	newCapture := func() *multiplePolicyCapture { return &multiplePolicyCapture{deadlineOK: true, leasesOK: true} }
	collector := func(j, l string, capture **multiplePolicyCapture, failure func(Policy) bool) ExecutionStartupPolicySource {
		return func(c context.Context, policy Policy) (Observation, []inventory.DiskObservation, PoolSystemUse, error) {
			got := *capture
			got.calls = append(got.calls, policy.PoolGUID)
			deadline, ok := c.Deadline()
			if got.deadline.IsZero() {
				got.deadline = deadline
			}
			got.deadlineOK = got.deadlineOK && ok && deadline == got.deadline && time.Until(deadline) > 0 && time.Until(deadline) <= 90*time.Second
			got.leasesOK = restartLocksHeld(t, j, l) && got.leasesOK
			if failure != nil && failure(policy) {
				return Observation{}, nil, PoolSystemUse{}, ErrUnavailable
			}
			switch {
			case multiplePolicyFixtureMatches(policy, p):
				return o, d, use, nil
			case multiplePolicyFixtureMatches(policy, q):
				return other, disks, use, nil
			default:
				return Observation{}, nil, PoolSystemUse{}, ErrDenied
			}
		}
	}
	system := func(context.Context) (PoolSystemUse, error) { return use, nil }
	j, l := policyOnlyStorageSeed(t, f, "multiple-positive", []Policy{p, q}, use)
	seed := readRecord(j)
	ledgerBefore := freshNamespaceSnapshot(t, l)
	check("multi_catalog", diagnostic(j, l))
	capture := newCapture()
	source := collector(j, l, &capture, nil)
	r, n, err := reviewExecutionStartupPolicy(ctx, j, l, f.read, system, activationBoot, source)
	if err != nil {
		t.Fatal("multiple original review", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	check("multi_reviewed", len(n.policyDigests) == 2 && n.snapshot.record == seed)
	captures := []*multiplePolicyCapture{capture}
	capture = newCapture()
	a, err := r.Activate(ctx)
	if err != nil {
		t.Fatal("multiple original activation", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	active := readRecord(j)
	check("multi_activated", active.Generation == seed.Generation+1 && active.Owner.State == "active" && active.Owner.ID != seed.Owner.ID)
	captures = append(captures, capture)
	capture = newCapture()
	got, err := a.Observe(ctx)
	check("multi_observed", err == nil && got.census == seed.Census)
	captures = append(captures, capture)
	capture = newCapture()
	check("multi_closed", a.FinishIdle(ctx) == nil)
	captures = append(captures, capture)
	closed := readRecord(j)
	check("multi_lineage", closed.Generation == seed.Generation+2 && closed.Owner.State == "closed" && closed.Owner.ID == active.Owner.ID && closed.Census == seed.Census && closed.Binding == seed.Binding && active.Census == seed.Census && active.Binding == seed.Binding)
	check("multi_ledger", reflect.DeepEqual(ledgerBefore, freshNamespaceSnapshot(t, l)))
	all, locks, deadlines := true, true, true
	for _, c := range captures {
		all = all && c.complete()
		locks = locks && c.leasesOK
		deadlines = deadlines && c.deadlineOK
	}
	check("multi_all-policies", all)
	check("multi_leases", locks)
	check("multi_shared-deadline", deadlines)
	_, err = a.Observe(ctx)
	_, againErr := r.Activate(ctx)
	check("multi_retired", err != nil && a.FinishIdle(ctx) != nil && againErr != nil && a.Close() == nil && restartLocksReleased(t, j, l))
	check("multi_reopened", diagnostic(j, l) && readRecord(j) == closed)

	for _, stage := range []string{"review", "activate", "observe", "finish", "post-activate", "post-finish"} {
		j, l := policyOnlyStorageSeed(t, f, "multiple-last-"+stage, []Policy{p, q}, use)
		capture := newCapture()
		armed := stage == "review"
		var before executionJournal
		source := collector(j, l, &capture, func(policy Policy) bool {
			if !armed || policy.PoolGUID != "200" {
				return false
			}
			if !strings.HasPrefix(stage, "post-") {
				return true
			}
			return readRecord(j).Generation == before.Generation+1
		})
		var r *ExecutionStartupReview
		var n *executionStartupNative
		var a *ExecutionRestartActivation
		var err error
		if stage != "review" {
			r, n, err = reviewExecutionStartupPolicy(ctx, j, l, f.read, system, activationBoot, source)
			if err != nil {
				t.Fatal("valid review before last-policy fault", err)
			}
			t.Cleanup(func() { _ = r.Close() })
			if stage == "observe" || stage == "finish" || stage == "post-finish" {
				a, err = r.Activate(ctx)
				if err != nil {
					t.Fatal("valid activation before last-policy fault", err)
				}
				t.Cleanup(func() { _ = a.Close() })
			}
		}
		before = readRecord(j)
		bj, bl := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
		capture = newCapture()
		armed = true
		zero := true
		switch stage {
		case "review":
			r, n, err = reviewExecutionStartupPolicy(ctx, j, l, f.read, system, activationBoot, source)
			zero = r == nil && n == nil
		case "activate", "post-activate":
			a, err = r.Activate(ctx)
			zero = a == nil
		case "observe":
			var observation ExecutionDomainsObservation
			observation, err = a.Observe(ctx)
			zero = observation == (ExecutionDomainsObservation{})
		case "finish", "post-finish":
			err = a.FinishIdle(ctx)
		}
		mode := "last-" + stage
		check(mode+"_denied", errors.Is(err, ErrUnavailable) && zero)
		check(mode+"_last-policy", capture.complete())
		check(mode+"_leases", capture.leasesOK)
		check(mode+"_deadline", capture.deadlineOK)
		visible := readRecord(j)
		preserved := reflect.DeepEqual(bl, freshNamespaceSnapshot(t, l))
		if strings.HasPrefix(stage, "post-") {
			state := "active"
			if stage == "post-finish" {
				state = "closed"
			}
			preserved = preserved && visible.Generation == before.Generation+1 && visible.Owner.State == state && visible.Census == before.Census && visible.Binding == before.Binding
			if stage == "post-activate" {
				preserved = preserved && visible.Owner.ID != before.Owner.ID
			} else {
				preserved = preserved && visible.Owner.ID == before.Owner.ID
			}
		} else {
			preserved = preserved && visible == before && reflect.DeepEqual(bj, freshNamespaceSnapshot(t, j))
		}
		check(mode+"_preserved", preserved)
		aj, al := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
		calls := len(capture.calls)
		sticky := false
		if stage == "review" {
			r, n, err = reviewExecutionStartupPolicy(ctx, j, l, f.read, system, activationBoot, source)
			sticky = errors.Is(err, ErrUnavailable) && r == nil && n == nil
		} else {
			armed = false
			if a != nil {
				_, err = a.Observe(ctx)
				sticky = err != nil && a.FinishIdle(ctx) != nil
			} else {
				_, err = r.Activate(ctx)
				_, observationErr := r.Observe(ctx)
				sticky = err != nil && observationErr != nil
			}
			sticky = sticky && len(capture.calls) == calls && n.pair.blocked && n.pair.native.journal.failed && n.pair.native.ledger.publicationBlocked
		}
		check(mode+"_sticky", sticky && reflect.DeepEqual(aj, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(al, freshNamespaceSnapshot(t, l)))
		released := true
		if a != nil {
			released = a.Close() == nil
		} else if r != nil {
			released = r.Close() == nil
		}
		check(mode+"_released", released && restartLocksReleased(t, j, l))
		check(mode+"_diagnostic", diagnostic(j, l) && visible == readRecord(j))
	}
}
