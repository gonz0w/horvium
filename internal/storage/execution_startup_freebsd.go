// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"time"
)

// Original read-only native review. Historical bytes never become an old
// publication acknowledgement, and this handle cannot publish a new incarnation.
type executionStartupNative struct {
	original    *executionStartupNative
	pair        *ExecutionDomainLeases
	snapshot    executionJournalSnapshot
	currentBoot [16]byte
	reviewed    bool
	read        func(context.Context, int) (ExecutionBackingObservation, error)
	system      func(context.Context) (PoolSystemUse, error)
	boot        func(context.Context) ([16]byte, error)
}

// Trusted root composition; production fixes native producers. Inspection only.
func ReviewExecutionStartup(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) (*ExecutionStartupReview, error) {
	if ctx == nil || read == nil || system == nil || boot == nil || os.Geteuid() != 0 {
		return nil, ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	pair, e := OpenExecutionDomainLeases(ctx, journal, ledger)
	if e != nil {
		return nil, e
	}
	n := &executionStartupNative{pair: pair, read: read, system: system, boot: boot}
	n.original = n
	_, snapshot, current, e := n.inspect(ctx)
	if e != nil {
		return nil, errors.Join(e, pair.Close())
	}
	n.snapshot = snapshot
	n.currentBoot = current
	n.reviewed = true
	q := &ExecutionStartupReview{observe: n.observe, release: pair.Close}
	q.original = q
	return q, nil
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
	if j.initializable || j.acknowledged {
		return zero, empty, rawZero, ErrDenied
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
				if e = checkClosedExecutionStartup(before.record, before.record.Binding, before.record.Census); e != nil {
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
				if e = checkClosedExecutionStartup(got.journal, got.binding, got.census); e != nil {
					return e
				}
				if p.bound && p.binding != got.binding {
					return ErrDrift
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
	p.binding, p.bound = got.binding, true
	succeeded = true
	return got, before, current, nil
}
