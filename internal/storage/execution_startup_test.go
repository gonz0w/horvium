// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestClosedExecutionStartupCandidate(t *testing.T) {
	j := executionJournalFixture()
	j.Generation = 2
	j.Owner.State = "closed"
	before := j
	if e := checkClosedExecutionStartup(j, j.Binding, j.Census); e != nil || j != before {
		t.Fatal("closed candidate", e)
	}
	// A closed prior boot may differ; this predicate supplies no native provenance.
	j.Owner.Boot = strings.Repeat("f", 32)
	if checkClosedExecutionStartup(j, j.Binding, j.Census) != nil {
		t.Fatal("closed prior boot metadata")
	}
	for _, mode := range []string{"absent", "active", "version", "overflow", "census", "installation", "directory", "lock", "ledger-backing", "journal-backing", "metadata", "native", "restore", "bad-owner"} {
		t.Run(mode, func(t *testing.T) {
			q := before
			b := q.Binding
			c := q.Census
			switch mode {
			case "absent":
				q = executionJournal{}
			case "active":
				q.Owner.State = "active"
			case "version":
				q.Version++
			case "overflow":
				q.Generation = math.MaxUint64
			case "census":
				c = strings.Repeat("f", 64)
			case "installation":
				b.Installation = strings.Repeat("3", 32)
			case "directory":
				b.Directory = strings.Repeat("f", 64)
			case "lock":
				b.Lock = strings.Repeat("f", 64)
			case "ledger-backing":
				b.LedgerBacking = strings.Repeat("f", 64)
			case "journal-backing":
				b.JournalBacking = strings.Repeat("f", 64)
			case "metadata":
				q.Metadata = journalMetadataFixture(q)
			case "native":
				q.Native = journalNativeFixture(q)
			case "restore":
				q.Disarmed = true
			case "bad-owner":
				q.Owner.ID = "invalid"
			}
			old := q
			if checkClosedExecutionStartup(q, b, c) == nil || q != old {
				t.Fatal("unsafe closed candidate")
			}
		})
	}
	next := before
	next.Generation++
	next.Owner.State = "active"
	next.Owner.ID = strings.Repeat("3", 32)
	if checkExecutionJournalSuccessor(before, next) == nil {
		t.Fatal("candidate predicate enabled ordinary owner replacement")
	}
}
func startupReviewFixture(read func(context.Context) (ExecutionDomainsObservation, error), release func() error) *ExecutionStartupReview {
	r := &ExecutionStartupReview{observe: read, release: release}
	r.original = r
	return r
}
func TestExecutionStartupReviewLifecycle(t *testing.T) {
	ctx := context.Background()
	reads, releases := 0, 0
	p := &ExecutionDomainLeases{}
	p.origin = p
	want := ExecutionDomainsObservation{origin: p, census: strings.Repeat("a", 64)}
	r := startupReviewFixture(func(context.Context) (ExecutionDomainsObservation, error) { reads++; return want, nil }, func() error { releases++; return nil })
	copied := &ExecutionStartupReview{original: r, observe: r.observe, release: r.release}
	wire, e := json.Marshal(r)
	var decoded ExecutionStartupReview
	if e != nil || string(wire) != "{}" || json.Unmarshal(wire, &decoded) != nil {
		t.Fatal("serialized startup review")
	}
	for _, q := range []*ExecutionStartupReview{nil, {}, copied, &decoded} {
		if got, e := q.Observe(ctx); e == nil || got != (ExecutionDomainsObservation{}) || q.Close() == nil || reads != 0 || releases != 0 {
			t.Fatal("foreign review reached source")
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for _, c := range []context.Context{nil, cancelled} {
		if got, e := r.Observe(c); e == nil || got != (ExecutionDomainsObservation{}) || reads != 0 {
			t.Fatal("invalid context consumed source")
		}
	}
	if got, e := r.Observe(ctx); e != nil || got != want || reads != 1 {
		t.Fatal("original closed review", e)
	}
	if r.Close() != nil || r.Close() != nil || releases != 1 {
		t.Fatal("original release")
	}
	if got, e := r.Observe(ctx); e == nil || got != (ExecutionDomainsObservation{}) || reads != 1 {
		t.Fatal("closed review reused")
	}
}
func TestExecutionStartupReviewFailures(t *testing.T) {
	for _, mode := range []string{"read", "cancel-after", "release"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reads, releases := 0, 0
			r := startupReviewFixture(func(context.Context) (ExecutionDomainsObservation, error) {
				reads++
				if mode == "read" {
					return ExecutionDomainsObservation{}, ErrDrift
				}
				if mode == "cancel-after" {
					cancel()
				}
				return ExecutionDomainsObservation{}, nil
			}, func() error {
				releases++
				if mode == "release" {
					return ErrStateUncertain
				}
				return nil
			})
			_, e := r.Observe(ctx)
			if (mode != "release") != (e != nil) {
				t.Fatal("read result", e)
			}
			if mode != "release" {
				if got, e := r.Observe(context.Background()); e == nil || got != (ExecutionDomainsObservation{}) || reads != 1 {
					t.Fatal("failed review revived")
				}
			}
			first := r.Close()
			if !errors.Is(r.Close(), first) || releases != 1 {
				t.Fatal("release repeated")
			}
			if mode == "release" && !errors.Is(first, ErrStateUncertain) {
				t.Fatal("release failure lost")
			}
		})
	}
}
func TestExecutionStartupReviewExclusion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	entered, resume := make(chan struct{}), make(chan struct{})
	released := make(chan struct{}, 1)
	r := startupReviewFixture(func(c context.Context) (ExecutionDomainsObservation, error) {
		close(entered)
		select {
		case <-resume:
			return ExecutionDomainsObservation{}, nil
		case <-c.Done():
			return ExecutionDomainsObservation{}, c.Err()
		}
	}, func() error { released <- struct{}{}; return nil })
	done := make(chan error, 1)
	go func() { _, e := r.Observe(ctx); done <- e }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("review did not enter")
	}
	closed := make(chan error, 1)
	go func() { closed <- r.Close() }()
	select {
	case <-released:
		t.Fatal("release overlapped review")
	case <-time.After(30 * time.Millisecond):
	}
	close(resume)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if e := <-closed; e != nil {
		t.Fatal(e)
	}
	select {
	case <-released:
	default:
		t.Fatal("original release missing")
	}
}
