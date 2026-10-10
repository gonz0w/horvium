// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"os"
	"time"
)

// Retain the original pair/journal/ledger locks through the original successor
// pipeline and complete post-publication checks. wrap is a private fixture seam.
func (n *executionStartupNative) finishIdle(ctx context.Context, wrap func(ownershipPublication) ownershipPublication) error {
	if ctx == nil || n == nil || n.original != n || n.pair == nil || n.pair.origin != n.pair || n.read == nil || n.system == nil || n.boot == nil || os.Geteuid() != 0 {
		return ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	p := n.pair
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.native == nil {
		return ErrDenied
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || j.original != j || j.lease == nil || j.lease.propertyOrigin != j.lease || l == nil || l.propertyOrigin != l || j.lease == l {
		return ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	// All entered outcomes, including a successful close, retire both stores.
	defer func() { p.blocked = true; j.retireLocked(); l.publicationBlocked = true }()
	if !n.reviewed || !n.activated || n.finished || p.blocked {
		return ErrDenied
	}
	got, _, _, e := n.inspectLocked(ctx)
	if e != nil {
		return e
	}
	census, e := newExecutionCensus(got.binding, []string{stateWriterName}, defaultStateCatalogLimits)
	if e != nil {
		return e
	}
	empty, e := census.finish()
	if e != nil {
		return e
	}
	if got.census != empty {
		return ErrDenied
	}
	next, e := unusedRestartClosedSuccessor(got.journal)
	if e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if os.Geteuid() != 0 {
		return ErrDenied
	}
	// Capture the namespace immediately after the real sync/acknowledgement,
	// inside that operation and before any outer wrapper can mutate objects.
	var acknowledgedObjects executionStartupObjects
	captured := false
	capture := func(tx ownershipPublication) ownershipPublication {
		syncDirectory := tx.syncDirectory
		tx.syncDirectory = func() error {
			if e := syncDirectory(); e != nil {
				return e
			}
			var e error
			acknowledgedObjects, e = readExecutionStartupObjectsLocked(j.lease)
			captured = e == nil
			return e
		}
		if wrap != nil {
			return wrap(tx)
		}
		return tx
	}
	if e = j.publishSuccessorLocked(next, capture); e != nil {
		return e
	}
	if !captured {
		return ErrStateUncertain
	}
	current, e := readExecutionStartupObjectsLocked(j.lease)
	if e != nil {
		return e
	}
	if !sameExecutionStartupObjects(current, acknowledgedObjects) {
		return ErrDrift
	}
	old := n.objects[0]
	if current.directory.Dev != old.directory.Dev || executionStatIdentity(current.directory) != executionStatIdentity(old.directory) || current.directory.Mode != old.directory.Mode || current.directory.Uid != old.directory.Uid || current.directory.Gid != old.directory.Gid || current.directory.Nlink != old.directory.Nlink || current.directory.Flags != old.directory.Flags || !sameOwnershipMetadata(current.lock, old.lock) || current.backing != old.backing {
		return ErrDrift
	}
	n.objects[0] = current
	n.snapshot = executionJournalSnapshot{present: true, record: j.ack, data: append([]byte(nil), j.ackBytes...), metadata: j.ackMetadata}
	n.finished = true
	_, _, _, e = n.inspectLocked(ctx)
	return e
}
