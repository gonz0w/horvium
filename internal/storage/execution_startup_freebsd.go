// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Original native review and its one-shot active consumer. Historical bytes
// never become an old publication acknowledgement.
type executionStartupObjects struct {
	directory, lock unix.Stat_t
	backing         propertyStateBacking
}
type executionStartupNative struct {
	original      *executionStartupNative
	pair          *ExecutionDomainLeases
	snapshot      executionJournalSnapshot
	objects       [2]executionStartupObjects
	currentBoot   [16]byte
	reviewed      bool
	activated     bool
	finished      bool
	policy        ExecutionStartupPolicySource
	policyDigests map[string][32]byte
	read          func(context.Context, int) (ExecutionBackingObservation, error)
	system        func(context.Context) (PoolSystemUse, error)
	boot          func(context.Context) ([16]byte, error)
}

// Trusted root composition; production fixes native producers. Inspection only.
func ReviewExecutionStartup(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) (*ExecutionStartupReview, error) {
	r, _, e := reviewExecutionStartup(ctx, journal, ledger, read, system, boot)
	return r, e
}
func reviewExecutionStartup(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) (*ExecutionStartupReview, *executionStartupNative, error) {
	return reviewExecutionStartupPolicy(ctx, journal, ledger, read, system, boot, nil)
}
func reviewExecutionStartupPolicy(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error), policy ExecutionStartupPolicySource) (*ExecutionStartupReview, *executionStartupNative, error) {
	// Behavioral red candidate: policy startup is deliberately unavailable.
	if policy != nil {
		return nil, nil, ErrDenied
	}
	if ctx == nil || read == nil || system == nil || boot == nil || os.Geteuid() != 0 {
		return nil, nil, ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return nil, nil, e
	}
	pair, e := OpenExecutionDomainLeases(ctx, journal, ledger)
	if e != nil {
		return nil, nil, e
	}
	n := &executionStartupNative{pair: pair, read: read, system: system, boot: boot, policy: policy}
	n.original = n
	_, snapshot, current, e := n.inspect(ctx)
	if e != nil {
		return nil, nil, errors.Join(e, pair.Close())
	}
	n.snapshot = snapshot
	n.currentBoot = current
	n.reviewed = true
	q := &ExecutionStartupReview{observe: n.observe, release: pair.Close, activate: func(c context.Context) (*ExecutionRestartActivation, error) { return n.activate(c, rand.Reader, nil) }}
	q.original = q
	return q, n, nil
}
func (n *executionStartupNative) observe(ctx context.Context) (ExecutionDomainsObservation, error) {
	got, _, _, e := n.inspect(ctx)
	return got, e
}

// Original pair, then journal and ledger mutexes surround every file, native and
// boot observation. No descriptor or decoded metadata transfers authority.
func (n *executionStartupNative) inspect(ctx context.Context) (ExecutionDomainsObservation, executionJournalSnapshot, [16]byte, error) {
	zero := ExecutionDomainsObservation{}
	empty := executionJournalSnapshot{}
	rawZero := [16]byte{}
	if ctx == nil || n == nil || n.original != n || n.pair == nil || n.pair.origin != n.pair || n.read == nil || n.system == nil || n.boot == nil || os.Geteuid() != 0 {
		return zero, empty, rawZero, ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return zero, empty, rawZero, e
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	p := n.pair
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.native == nil {
		return zero, empty, rawZero, ErrDenied
	}
	if p.blocked {
		return zero, empty, rawZero, ErrStateUncertain
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || j.original != j || j.lease == nil || j.lease.propertyOrigin != j.lease || l == nil || l.propertyOrigin != l || j.lease == l {
		return zero, empty, rawZero, ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	return n.inspectLocked(ctx)
}

// Caller holds the original pair, journal and ledger mutexes. Preserve the exact
// closed-review checks; only its original acknowledged consumer enters active mode.
func (n *executionStartupNative) inspectLocked(ctx context.Context) (ExecutionDomainsObservation, executionJournalSnapshot, [16]byte, error) {
	zero := ExecutionDomainsObservation{}
	empty := executionJournalSnapshot{}
	rawZero := [16]byte{}
	p := n.pair
	j, l := p.native.journal, p.native.ledger
	succeeded := false
	defer func() {
		if !succeeded {
			p.blocked = true
			j.retireLocked()
			l.publicationBlocked = true
		}
	}()
	if e := ctx.Err(); e != nil {
		return zero, empty, rawZero, e
	}
	if j.failed || j.lease.publicationBlocked || l.publicationBlocked {
		return zero, empty, rawZero, ErrStateUncertain
	}
	// Ordinary reopen must remain observation-only. Never synthesize an old ack.
	if j.initializable || j.acknowledged != n.activated {
		return zero, empty, rawZero, ErrDenied
	}
	if n.activated && (j.ack != n.snapshot.record || !bytes.Equal(j.ackBytes, n.snapshot.data) || !sameOwnershipMetadata(j.ackMetadata, n.snapshot.metadata)) {
		return zero, empty, rawZero, ErrDrift
	}
	var objects [2]executionStartupObjects
	for i, state := range []*StateDirectory{j.lease, l} {
		value, e := readExecutionStartupObjectsLocked(state)
		if e != nil {
			return zero, empty, rawZero, e
		}
		if n.reviewed && !sameExecutionStartupObjects(value, n.objects[i]) {
			return zero, empty, rawZero, ErrDrift
		}
		objects[i] = value
	}
	var got ExecutionDomainsObservation
	var before executionJournalSnapshot
	var current [16]byte
	var ledgerErr, work error
	_, journalErr := j.lease.inspectExecutionBindingObjectsLocked(func() {
		_, ledgerErr = l.inspectExecutionBindingObjectsLocked(func() {
			inspect := func() error {
				var e error
				before, e = j.inspectLocked("", nil)
				if e != nil {
					return e
				}
				if !before.present {
					return ErrDenied
				}
				if e = n.checkRecord(before.record, before.record.Binding, before.record.Census); e != nil {
					return e
				}
				if n.reviewed && (before.record != n.snapshot.record || !bytes.Equal(before.data, n.snapshot.data) || !sameOwnershipMetadata(before.metadata, n.snapshot.metadata)) {
					return ErrDrift
				}
				current, e = n.boot(ctx)
				if e != nil {
					return e
				}
				if n.reviewed && current != n.currentBoot {
					return ErrDrift
				}
				// Repeated object inspection also guards actual mount names and metadata.
				jb, e := propertyStateFilesystem(int(j.lease.directory.Fd()))
				if e != nil {
					return e
				}
				lb, e := propertyStateFilesystem(int(l.directory.Fd()))
				if e != nil {
					return e
				}
				if jb.kind == "zfs" && lb.kind == "zfs" && (strings.HasPrefix(jb.source, lb.source+"/") || strings.HasPrefix(lb.source, jb.source+"/")) {
					return ErrDenied
				}
				got, e = p.observeLocked(ctx, before.record.Binding.Installation, n.read, n.system)
				if e != nil {
					return e
				}
				if !got.journalPresent || !got.systemObserved || got.journal != before.record {
					return ErrDrift
				}
				if e = n.checkRecord(got.journal, got.binding, got.census); e != nil {
					return e
				}
				if p.bound && p.binding != got.binding {
					return ErrDrift
				}
				if n.policy != nil {
					if e = n.inspectPoliciesLocked(ctx, got); e != nil {
						return e
					}
					if len(n.policyDigests) != 0 {
						final, e := p.observeLocked(ctx, before.record.Binding.Installation, n.read, n.system)
						if e != nil {
							return e
						}
						if final != got {
							return ErrDrift
						}
					}
				}
				again, e := n.boot(ctx)
				if e != nil {
					return e
				}
				if again != current {
					return ErrDrift
				}
				last, e := j.inspectLocked("", nil)
				if e != nil {
					return e
				}
				if !last.present || last.record != before.record || !bytes.Equal(last.data, before.data) || !sameOwnershipMetadata(last.metadata, before.metadata) {
					return ErrDrift
				}
				if e = ctx.Err(); e != nil {
					return e
				}
				if os.Geteuid() != 0 {
					return ErrDenied
				}
				return nil
			}
			work = inspect()
		})
	})
	for _, e := range []error{journalErr, ledgerErr, work, ctx.Err()} {
		if e != nil {
			return zero, empty, rawZero, e
		}
	}
	if os.Geteuid() != 0 {
		return zero, empty, rawZero, ErrDenied
	}
	for i, state := range []*StateDirectory{j.lease, l} {
		value, e := readExecutionStartupObjectsLocked(state)
		if e != nil {
			return zero, empty, rawZero, e
		}
		if !sameExecutionStartupObjects(value, objects[i]) {
			return zero, empty, rawZero, ErrDrift
		}
	}
	if e := ctx.Err(); e != nil {
		return zero, empty, rawZero, e
	}
	if os.Geteuid() != 0 {
		return zero, empty, rawZero, ErrDenied
	}
	if !n.reviewed {
		n.objects = objects
	}
	p.binding, p.bound = got.binding, true
	succeeded = true
	return got, before, current, nil
}

// Called only under the original state mutex. No reopened/replacement descriptor
// is retained. Full mutable metadata matters for this read-only review lifetime.
func readExecutionStartupObjectsLocked(s *StateDirectory) (executionStartupObjects, error) {
	zero := executionStartupObjects{}
	if s == nil || s.propertyOrigin != s || os.Geteuid() != 0 {
		return zero, ErrDenied
	}
	dir, e := s.ownershipDirectoryLocked()
	if e != nil {
		return zero, e
	}
	var got executionStartupObjects
	if unix.Fstat(dir, &got.directory) != nil || unix.Fstat(int(s.writer.Fd()), &got.lock) != nil {
		return zero, ErrStateDirectory
	}
	got.backing, e = propertyStateFilesystem(dir)
	if e != nil {
		return zero, e
	}
	return got, nil
}
func sameExecutionStartupObjects(a, b executionStartupObjects) bool {
	return sameOwnershipMetadata(a.directory, b.directory) && sameOwnershipMetadata(a.lock, b.lock) && a.backing == b.backing
}
