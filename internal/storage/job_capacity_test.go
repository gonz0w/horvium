// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"testing"
)

func catalogForJob(j Job, size uint64) StateCatalog {
	last := j.History[len(j.History)-1]
	c := StateCatalog{Jobs: []JobSummary{{ID: j.ID, SubmissionID: j.Intent.SubmissionID, Revision: last.Revision, State: last.State, Bytes: size}}, Bytes: size}
	if j.Terminal() {
		c.Terminal = 1
	} else {
		c.Nonterminal = 1
	}
	return c
}

func TestJobPublicationCapacity(t *testing.T) {
	j := catalogJob(1, false)
	check := func(c StateCatalog, next Job, expected, size uint64, limits stateCatalogLimits, want error) {
		t.Helper()
		if e := allowJobPublication(c, next, expected, size, limits); !errors.Is(e, want) {
			t.Fatal("job capacity/revision", e, "want", want)
		}
	}
	limits := stateCatalogLimits{1, 1, 1, 100}
	check(StateCatalog{}, j, 0, 100, limits, nil)
	check(StateCatalog{}, j, 0, 101, limits, ErrLimit)
	old := catalogForJob(j, 100)
	limits.bytes = 200
	next, e := TransitionJob(j, JobApplying, 1001, "launch-intent", "")
	if e != nil {
		t.Fatal(e)
	}
	check(old, next, 1, 100, limits, nil)
	limits.bytes = 199
	check(old, next, 1, 100, limits, ErrLimit)
	limits.bytes = 200
	check(old, j, 0, 100, limits, ErrDrift)
	check(old, next, 2, 100, limits, ErrDrift)
	check(StateCatalog{}, next, 1, 100, limits, ErrDrift)
	check(StateCatalog{}, next, 0, 100, limits, ErrDrift)
	check(old, next, ^uint64(0), 100, limits, ErrDrift)
	another := catalogJob(2, false)
	check(old, another, 0, 100, limits, ErrLimit)
	another.Intent.SubmissionID = j.Intent.SubmissionID
	check(old, another, 0, 100, limits, ErrDrift)
	rewritten := next
	rewritten.Intent.SubmissionID = catalogJob(2, false).Intent.SubmissionID
	check(old, rewritten, 1, 100, limits, ErrDrift)
	done, e := TransitionJob(j, JobCancelled, 1001, "operator-cancelled", "")
	if e != nil {
		t.Fatal(e)
	}
	check(old, done, 1, 100, limits, nil)
	finished := catalogJob(2, true)
	s := catalogForJob(finished, 50)
	old.Jobs = append(old.Jobs, s.Jobs[0])
	old.Terminal = 1
	old.Bytes = 150
	limits.bytes = 300
	check(old, done, 1, 100, limits, ErrLimit)
	// A nonterminal successor occupies its existing slot even when the queue is full.
	check(old, next, 1, 100, limits, nil)
	for _, bad := range []stateCatalogLimits{{}, {1, 65, 1, 1000}, {1, 1, 1001, 1000}, {1, 1, 1, (64 << 20) + 1}} {
		check(StateCatalog{}, j, 0, 50, bad, ErrLimit)
	}
	for _, size := range []uint64{0, maxStateRecordBytes + 1, ^uint64(0)} {
		check(StateCatalog{}, j, 0, size, defaultStateCatalogLimits, ErrLimit)
	}
	overflow := StateCatalog{Bytes: ^uint64(0)}
	check(overflow, j, 0, 1, defaultStateCatalogLimits, ErrLimit)
	invalid := j
	invalid.ID = "../writer.lock"
	check(StateCatalog{}, invalid, 0, 100, limits, ErrStateRecord)
	if old.Nonterminal != 1 || old.Terminal != 1 || old.Bytes != 150 || len(j.History) != 1 {
		t.Fatal("capacity guard changed inputs")
	}
}

func TestMixedOwnershipPublicationCapacity(t *testing.T) {
	j := catalogJob(1, false)
	c := catalogForJob(j, 100)
	p := Policy{PoolGUID: "100", Revision: 1}
	limits := stateCatalogLimits{1, 1, 1, 150}
	if e := allowMixedOwnershipPublication(c, p, 0, 50, limits); e != nil {
		t.Fatal("job bytes plus owner", e)
	}
	limits.bytes--
	if e := allowMixedOwnershipPublication(c, p, 0, 50, limits); !errors.Is(e, ErrLimit) {
		t.Fatal("mixed bytes", e)
	}
	limits.bytes = 250
	c.Ownership = []OwnershipSummary{{PoolGUID: "100", Revision: 1, Bytes: 100}}
	c.Bytes = 200
	p.Revision = 2
	if e := allowMixedOwnershipPublication(c, p, 1, 50, limits); e != nil {
		t.Fatal("mixed replacement reservation", e)
	}
	limits.bytes--
	if e := allowMixedOwnershipPublication(c, p, 1, 50, limits); !errors.Is(e, ErrLimit) {
		t.Fatal("transient mixed bytes", e)
	}
	limits.bytes = 1000
	q := p
	q.PoolGUID = "200"
	q.Revision = 1
	if e := allowMixedOwnershipPublication(c, q, 0, 50, limits); !errors.Is(e, ErrLimit) {
		t.Fatal("ownership count", e)
	}
	c.Nonterminal = 2
	if e := allowMixedOwnershipPublication(c, p, 1, 50, limits); !errors.Is(e, ErrLimit) {
		t.Fatal("job class overflow cannot authorize owner", e)
	}
	c.Nonterminal = -1
	if e := allowMixedOwnershipPublication(c, p, 1, 50, limits); !errors.Is(e, ErrLimit) {
		t.Fatal("negative census accepted", e)
	}
}
