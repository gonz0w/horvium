// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"encoding/hex"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"time"
)

// Original pair/journal/ledger mutexes remain held from review through every
// publication and post-publication check. wrap/entropy are private fixture seams.
func (n *executionStartupNative) activate(ctx context.Context, entropy io.Reader, wrap func(ownershipPublication) ownershipPublication) (*ExecutionRestartActivation, error) {
	if ctx == nil || n == nil || n.original != n || n.pair == nil || n.pair.origin != n.pair || n.read == nil || n.system == nil || n.boot == nil || entropy == nil || os.Geteuid() != 0 {
		return nil, ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	p := n.pair
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.native == nil {
		return nil, ErrDenied
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || j.original != j || j.lease == nil || j.lease.propertyOrigin != j.lease || l == nil || l.propertyOrigin != l || j.lease == l {
		return nil, ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	success := false
	defer func() {
		if !success {
			p.blocked = true
			j.retireLocked()
			l.publicationBlocked = true
		}
	}()
	if !n.reviewed || n.activated || p.blocked {
		return nil, ErrDenied
	}
	got, _, raw, e := n.inspectLocked(ctx)
	if e != nil {
		return nil, e
	}
	// Nonempty retained work requires its own startup policy/history consumer.
	census, e := newExecutionCensus(got.binding, []string{stateWriterName}, defaultStateCatalogLimits)
	if e != nil {
		return nil, e
	}
	empty, e := census.finish()
	if e != nil {
		return nil, e
	}
	if got.census != empty && n.policy == nil {
		return nil, ErrDenied
	}
	var identity [16]byte
	if _, e = io.ReadFull(entropy, identity[:]); e != nil {
		return nil, e
	}
	next, e := closedExecutionJournalSuccessor(got.journal, hex.EncodeToString(identity[:]), raw)
	if e != nil {
		return nil, e
	}
	// Entropy collection and other original transfers cannot absorb drift.
	if _, _, _, e = n.inspectLocked(ctx); e != nil {
		return nil, e
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	if os.Geteuid() != 0 {
		return nil, ErrDenied
	}
	if e = n.publishClosedStartupLocked(next, wrap); e != nil {
		return nil, e
	}
	// Capture only the original pipeline's acknowledged namespace mutation,
	// before post-publication producers can change any retained object.
	current, e := readExecutionStartupObjectsLocked(j.lease)
	if e != nil {
		return nil, e
	}
	old := n.objects[0]
	if current.directory.Dev != old.directory.Dev || executionStatIdentity(current.directory) != executionStatIdentity(old.directory) || current.directory.Mode != old.directory.Mode || current.directory.Uid != old.directory.Uid || current.directory.Gid != old.directory.Gid || current.directory.Nlink != old.directory.Nlink || current.directory.Flags != old.directory.Flags || !sameOwnershipMetadata(current.lock, old.lock) || current.backing != old.backing {
		return nil, ErrDrift
	}
	n.objects[0] = current
	n.snapshot = executionJournalSnapshot{present: true, record: j.ack, data: append([]byte(nil), j.ackBytes...), metadata: j.ackMetadata}
	n.activated = true
	if _, _, _, e = n.inspectLocked(ctx); e != nil {
		return nil, e
	}
	a := &ExecutionRestartActivation{observe: n.observe, release: p.Close, finish: func(c context.Context) error { return n.finishIdle(c, nil) }}
	a.original = a
	success = true
	return a, nil
}

// New acknowledgement only. Never populate an old closed record as s.ack.
// Caller holds both original state mutexes after full original native review.
func (n *executionStartupNative) publishClosedStartupLocked(next executionJournal, wrap func(ownershipPublication) ownershipPublication) error {
	s := n.pair.native.journal
	before := n.snapshot
	if s.original != s || s.initializable || s.acknowledged || s.failed || !before.present {
		return ErrDenied
	}
	expected, e := closedExecutionJournalSuccessor(before.record, next.Owner.ID, n.currentBoot)
	if e != nil {
		return e
	}
	if next != expected {
		return ErrDenied
	}
	data, e := encodeExecutionJournal(next)
	if e != nil {
		return e
	}
	predecessor := func() error {
		current, e := s.lease.readStateBytesLocked(executionJournalName)
		if e != nil {
			return e
		}
		if !bytes.Equal(current, before.data) {
			return ErrDrift
		}
		return nil
	}
	inspect := func(temporary string) (StateCatalog, error) {
		_, e := s.inspectLocked(temporary, nil)
		return StateCatalog{}, e
	}
	var acknowledged executionJournalSnapshot
	var acknowledgedObjects executionStartupObjects
	acknowledge := func(dir int, name string, data []byte, written unix.Stat_t) error {
		if e := verifyPublishedState(dir, name, data, written); e != nil {
			return e
		}
		var e error
		acknowledged, e = s.inspectLocked("", nil)
		if e != nil {
			return e
		}
		if !acknowledged.present || acknowledged.record != next || !bytes.Equal(acknowledged.data, data) || acknowledged.metadata.Dev != written.Dev || acknowledged.metadata.Ino != written.Ino {
			return ErrDrift
		}
		acknowledgedObjects, e = readExecutionStartupObjectsLocked(s.lease)
		return e
	}
	e = s.lease.publishStateAcknowledgedLocked(executionJournalName, executionJournalTemporaryPrefix, data, before.record.Generation, func(StateCatalog) error { return nil }, predecessor, inspect, acknowledge, wrap)
	if e != nil {
		s.retireLocked()
		return e
	}
	final, e := s.inspectLocked("", nil)
	if e != nil || !acknowledged.present || !final.present || final.record != next || !bytes.Equal(final.data, data) || !sameOwnershipMetadata(final.metadata, acknowledged.metadata) {
		s.retireLocked()
		return ErrStateUncertain
	}
	current, e := readExecutionStartupObjectsLocked(s.lease)
	if e != nil || !sameExecutionStartupObjects(current, acknowledgedObjects) {
		s.retireLocked()
		return ErrDrift
	}
	s.acknowledged = true
	s.initializable = false
	s.ack = next
	s.ackBytes = append([]byte(nil), data...)
	s.ackMetadata = final.metadata
	return nil
}

// The original lifetime checks its exact active or original closed successor
// and confirmed empty census; this is not general recovery clearance.
func (n *executionStartupNative) checkRecord(j executionJournal, b executionJournalBinding, census string) error {
	if !n.activated {
		return checkClosedExecutionStartup(j, b, census)
	}
	if e := validateExecutionJournal(j); e != nil {
		return e
	}
	state := "active"
	if n.finished {
		state = "closed"
	}
	if j.Owner.State != state || j.Metadata.Pending || j.Native.Pending || j.Disarmed {
		return ErrDenied
	}
	if j != n.snapshot.record || j.Binding != b || j.Census != census {
		return ErrDrift
	}
	return nil
}
