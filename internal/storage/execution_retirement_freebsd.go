// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"encoding/hex"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// No reconstructed journal or caller declaration can finish an initial lifetime.
// wrap is a private synchronous fixture seam; production fixes the real pipeline.
func (a *freshExecutionActivation) finishIdle(ctx context.Context, wrap func(ownershipPublication) ownershipPublication) error {
	if a == nil || a.original != a || a.fresh == nil || a.fresh.original != a.fresh || a.read == nil || a.system == nil || a.boot == nil || ctx == nil || os.Geteuid() != 0 {
		return ErrDenied
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	f := a.fresh
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed || !f.consumed || f.pair == nil {
		return ErrDenied
	}
	p := f.pair
	if p.origin != p {
		return ErrDenied
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.native == nil {
		return ErrDenied
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || j.original != j || j.lease == nil || l == nil || j.lease == l {
		return ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	// Every entered outcome is terminal for this original initial lifetime. No
	// native work can be admitted through this handle; do not generalize to workers.
	defer f.retireActivationLocked()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if f.blocked || p.blocked || j.failed || j.lease.publicationBlocked || l.publicationBlocked {
		return ErrStateUncertain
	}
	if !p.bound || p.binding != a.journal.Binding || checkInitialExecutionJournal(a.journal) != nil {
		return ErrDenied
	}
	census, e := newExecutionCensus(a.journal.Binding, []string{stateWriterName}, defaultStateCatalogLimits)
	if e != nil {
		return e
	}
	empty, e := census.finish()
	if e != nil {
		return e
	}
	if a.journal.Census != empty {
		return ErrDenied
	}
	if e = f.inspectActivationAnchorsLocked(); e != nil {
		return e
	}
	if e = f.checkActivationReceiptLocked(a.journal); e != nil {
		return e
	}
	raw, e := a.boot(ctx)
	if e != nil {
		return e
	}
	if hex.EncodeToString(raw[:]) != a.journal.Owner.Boot {
		return ErrDrift
	}
	inspect := func(expected executionJournal) error {
		got, e := f.observeActivationDomainsLocked(ctx, p.binding.Installation, a.read, a.system)
		if e != nil {
			return e
		}
		if !got.systemObserved || !got.journalPresent || got.binding != p.binding || got.journal != expected || got.census != empty {
			return ErrDrift
		}
		return f.checkActivationReceiptLocked(expected)
	}
	if e = inspect(a.journal); e != nil {
		return e
	}
	again, e := a.boot(ctx)
	if e != nil {
		return e
	}
	if again != raw {
		return ErrDrift
	}
	if e = f.inspectActivationAnchorsLocked(); e != nil {
		return e
	}
	if e = f.checkActivationReceiptLocked(a.journal); e != nil {
		return e
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if os.Geteuid() != 0 {
		return ErrDenied
	}
	next := a.journal
	next.Generation++
	next.Owner.State = "closed"
	if e = j.publishSuccessorLocked(next, wrap); e != nil {
		return e
	}
	// Freeze only the exact directory change from our own acknowledged successor.
	// Later callback touches cannot be absorbed as expected publication metadata.
	var current unix.Stat_t
	old := f.anchors[0].directory
	if unix.Fstat(int(j.lease.directory.Fd()), &current) != nil || current.Dev != old.Dev || executionStatIdentity(current) != executionStatIdentity(old) || current.Mode != old.Mode || current.Uid != old.Uid || current.Gid != old.Gid || current.Nlink != old.Nlink || current.Flags != old.Flags {
		return ErrDrift
	}
	if e = inspect(next); e != nil {
		return e
	}
	again, e = a.boot(ctx)
	if e != nil {
		return e
	}
	if again != raw {
		return ErrDrift
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if os.Geteuid() != 0 {
		return ErrDenied
	}
	if e = f.checkActivationReceiptLocked(next); e != nil {
		return e
	}
	var final unix.Stat_t
	if unix.Fstat(int(j.lease.directory.Fd()), &final) != nil || !sameOwnershipMetadata(current, final) {
		return ErrDrift
	}
	f.anchors[0].directory = current
	return f.inspectActivationAnchorsLocked()
}
