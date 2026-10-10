// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// Original initial acknowledgement and retained leases only. No registered
// startup/global owner, clean shutdown, restore or execution credential.
// All producers are retained from the original trusted root composition.
type freshExecutionActivation struct {
	original *freshExecutionActivation
	fresh    *freshExecutionDomains
	journal  executionJournal
	read     func(context.Context, int) (ExecutionBackingObservation, error)
	system   func(context.Context) (PoolSystemUse, error)
	boot     func(context.Context) ([16]byte, error)
}

// Private root composition. No existing namespace is adopted or repaired.
// read/system/boot must be fixed native producers; fixture transfers do not
// qualify assembled commissioning. No runtime consumer is registered yet.
func activateFreshExecutionJournal(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error)) (*freshExecutionActivation, error) {
	if ctx == nil || read == nil || system == nil || boot == nil || os.Geteuid() != 0 {
		return nil, ErrDenied
	}
	f, e := provisionFreshExecutionDomains(ctx, journal, ledger, nil)
	if e != nil {
		return nil, e
	}
	a, e := f.activate(ctx, read, system, boot, rand.Reader, nil)
	if e != nil {
		return nil, errors.Join(e, f.close())
	}
	return a, nil
}

// entropy and wrap are private synchronous fault seams. The root composition
// fixes crypto/rand.Reader and the actual file/directory syscall pipeline.
func (f *freshExecutionDomains) activate(ctx context.Context, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error), entropy io.Reader, wrap func(ownershipPublication) ownershipPublication) (*freshExecutionActivation, error) {
	if f == nil || ctx == nil || read == nil || system == nil || boot == nil || entropy == nil || os.Geteuid() != 0 {
		return nil, ErrDenied
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if e := f.inspect(ctx); e != nil {
		return nil, e
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.original != f || f.closed || f.consumed || f.pair == nil {
		return nil, ErrDenied
	}
	p := f.pair
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.origin != p || p.closed || p.native == nil {
		return nil, ErrDenied
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || j.original != j || j.lease == nil || l == nil || j.lease == l {
		return nil, ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	f.consumed = true
	succeeded := false
	defer func() {
		if !succeeded {
			f.retireActivationLocked()
		}
	}()
	if f.blocked || p.blocked || p.bound || j.failed || j.acknowledged || j.initializable || j.lease.publicationBlocked || l.publicationBlocked {
		return nil, ErrDenied
	}
	if e := f.inspectActivationAnchorsLocked(); e != nil {
		return nil, e
	}
	var identities [32]byte
	if _, e := io.ReadFull(entropy, identities[:]); e != nil {
		return nil, e
	}
	if bytes.Equal(identities[:16], identities[16:]) {
		return nil, ErrDenied
	}
	installation, owner := hex.EncodeToString(identities[:16]), hex.EncodeToString(identities[16:])
	raw, e := boot(ctx)
	if e != nil {
		return nil, e
	}
	before, e := f.observeActivationDomainsLocked(ctx, installation, read, system)
	if e != nil {
		return nil, e
	}
	if before.journalPresent || !before.systemObserved {
		return nil, ErrDenied
	}
	census, e := newExecutionCensus(before.binding, []string{stateWriterName}, defaultStateCatalogLimits)
	if e != nil {
		return nil, e
	}
	empty, e := census.finish()
	if e != nil {
		return nil, e
	}
	if before.census != empty {
		return nil, ErrDrift
	}
	// Distinct GUIDs are already mandatory. Also refuse an ancestor relationship
	// between ZFS domains: a ledger dataset rollback scope must exclude journal.
	jb, lb := f.anchors[0].backing, f.anchors[1].backing
	if jb.kind == "zfs" && lb.kind == "zfs" && (strings.HasPrefix(jb.source, lb.source+"/") || strings.HasPrefix(lb.source, jb.source+"/")) {
		return nil, ErrDenied
	}
	if e = f.inspectActivationAnchorsLocked(); e != nil {
		return nil, e
	}
	again, e := boot(ctx)
	if e != nil {
		return nil, e
	}
	if again != raw {
		return nil, ErrDrift
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if os.Geteuid() != 0 {
		return nil, ErrDenied
	}
	initial := executionJournal{Version: 1, Kind: "execution-journal", Generation: 1, Binding: before.binding, Owner: executionJournalOwner{ID: owner, Boot: hex.EncodeToString(raw[:]), State: "active"}, Census: empty}
	// Only this original one-shot fresh receipt enables the already-locked
	// initializer. Ordinary/reopened journal stores cannot reconstruct it.
	j.initializable = true
	if e = j.initializeLocked(initial, wrap); e != nil {
		return nil, e
	}
	// Capture the post-publication directory before further producers run.
	// Later touches cannot be mistaken for our own acknowledged namespace change.
	var current unix.Stat_t
	old := f.anchors[0].directory
	if unix.Fstat(int(j.lease.directory.Fd()), &current) != nil || current.Dev != old.Dev || executionStatIdentity(current) != executionStatIdentity(old) || current.Mode != old.Mode || current.Uid != old.Uid || current.Gid != old.Gid || current.Nlink != old.Nlink || current.Flags != old.Flags {
		return nil, ErrDrift
	}
	after, e := f.observeActivationDomainsLocked(ctx, installation, read, system)
	if e != nil {
		return nil, e
	}
	if !after.journalPresent || !after.systemObserved || after.binding != before.binding || after.census != empty || after.journal != initial {
		return nil, ErrDrift
	}
	if e = f.checkActivationReceiptLocked(initial); e != nil {
		return nil, e
	}
	again, e = boot(ctx)
	if e != nil {
		return nil, e
	}
	if again != raw {
		return nil, ErrDrift
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if os.Geteuid() != 0 {
		return nil, ErrDenied
	}
	if e = f.checkActivationReceiptLocked(initial); e != nil {
		return nil, e
	}
	var finalDirectory unix.Stat_t
	if unix.Fstat(int(j.lease.directory.Fd()), &finalDirectory) != nil || !sameOwnershipMetadata(current, finalDirectory) {
		return nil, ErrDrift
	}
	// Only the exact original post-publication snapshot may replace freshness.
	f.anchors[0].directory = current
	if e = f.inspectActivationAnchorsLocked(); e != nil {
		return nil, e
	}
	p.binding, p.bound = before.binding, true
	a := &freshExecutionActivation{fresh: f, journal: initial, read: read, system: system, boot: boot}
	a.original = a
	succeeded = true
	return a, nil
}

func (a *freshExecutionActivation) observe(ctx context.Context) (ExecutionDomainsObservation, error) {
	zero := ExecutionDomainsObservation{}
	if a == nil || a.original != a || a.fresh == nil || a.read == nil || a.system == nil || a.boot == nil || ctx == nil || os.Geteuid() != 0 {
		return zero, ErrDenied
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	f := a.fresh
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.original != f || f.closed || !f.consumed || f.pair == nil {
		return zero, ErrDenied
	}
	p := f.pair
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.origin != p || p.closed || p.native == nil {
		return zero, ErrDenied
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || j.original != j || j.lease == nil || l == nil || j.lease == l {
		return zero, ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	succeeded := false
	defer func() {
		if !succeeded {
			f.retireActivationLocked()
		}
	}()
	if f.blocked || p.blocked || j.failed || j.lease.publicationBlocked || l.publicationBlocked {
		return zero, ErrStateUncertain
	}
	if !p.bound || p.binding != a.journal.Binding {
		return zero, ErrDrift
	}
	if e := f.inspectActivationAnchorsLocked(); e != nil {
		return zero, e
	}
	if e := f.checkActivationReceiptLocked(a.journal); e != nil {
		return zero, e
	}
	boot, e := a.boot(ctx)
	if e != nil {
		return zero, e
	}
	if hex.EncodeToString(boot[:]) != a.journal.Owner.Boot {
		return zero, ErrDrift
	}
	got, e := f.observeActivationDomainsLocked(ctx, p.binding.Installation, a.read, a.system)
	if e != nil {
		return zero, e
	}
	if !got.systemObserved || !got.journalPresent || got.binding != a.journal.Binding || got.journal != a.journal || got.census != a.journal.Census {
		return zero, ErrDrift
	}
	again, e := a.boot(ctx)
	if e != nil {
		return zero, e
	}
	if again != boot {
		return zero, ErrDrift
	}
	if e = f.checkActivationReceiptLocked(a.journal); e != nil {
		return zero, e
	}
	if e = f.inspectActivationAnchorsLocked(); e != nil {
		return zero, e
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if os.Geteuid() != 0 {
		return zero, ErrDenied
	}
	succeeded = true
	return got, nil
}
func (a *freshExecutionActivation) close() error {
	if a == nil || a.original != a || a.fresh == nil {
		return ErrDenied
	}
	return a.fresh.close()
}

// All helpers below require f, pair, journal then ledger mutexes.
func (f *freshExecutionDomains) retireActivationLocked() {
	f.blocked = true
	p := f.pair
	p.blocked = true
	p.native.journal.retireLocked()
	p.native.ledger.publicationBlocked = true
}
func (f *freshExecutionDomains) inspectActivationAnchorsLocked() error {
	for i, s := range []*StateDirectory{f.pair.native.journal.lease, f.pair.native.ledger} {
		if e := f.anchors[i].inspectLocked(s); e != nil {
			return e
		}
	}
	return nil
}
func (f *freshExecutionDomains) observeActivationDomainsLocked(ctx context.Context, installation string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error)) (ExecutionDomainsObservation, error) {
	p := f.pair
	var got ExecutionDomainsObservation
	var le, work error
	_, je := p.native.journal.lease.inspectExecutionBindingObjectsLocked(func() {
		_, le = p.native.ledger.inspectExecutionBindingObjectsLocked(func() { got, work = p.observeLocked(ctx, installation, read, system) })
	})
	if je != nil {
		return ExecutionDomainsObservation{}, je
	}
	if le != nil {
		return ExecutionDomainsObservation{}, le
	}
	if work != nil {
		return ExecutionDomainsObservation{}, work
	}
	if ctx.Err() != nil {
		return ExecutionDomainsObservation{}, ctx.Err()
	}
	if os.Geteuid() != 0 {
		return ExecutionDomainsObservation{}, ErrDenied
	}
	return got, nil
}
func (f *freshExecutionDomains) checkActivationReceiptLocked(expected executionJournal) error {
	j := f.pair.native.journal
	if !j.acknowledged || j.ack != expected {
		return ErrDenied
	}
	actual, e := j.inspectLocked("", nil)
	if e != nil {
		return e
	}
	if !actual.present || actual.record != expected || !bytes.Equal(actual.data, j.ackBytes) || !sameOwnershipMetadata(actual.metadata, j.ackMetadata) {
		return ErrDrift
	}
	return nil
}
