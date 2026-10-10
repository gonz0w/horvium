// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"time"
)

type executionDomainLeaseNative struct {
	journal *executionJournalStore
	ledger  *StateDirectory
}

// OpenExecutionDomainLeases opens existing protected domains in journal-before-
// ledger order. Both locks must already exist. No file is initialized/repaired.
// The returned owner supplies observation only, never provisioning or clearance.
func OpenExecutionDomainLeases(ctx context.Context, journalPath, ledgerPath string) (*ExecutionDomainLeases, error) {
	if ctx == nil || os.Geteuid() != 0 {
		return nil, ErrStateDirectory
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	j, e := openExecutionJournalStore(journalPath)
	if e != nil {
		return nil, e
	}
	fail := func(e error) (*ExecutionDomainLeases, error) { return nil, errors.Join(e, j.close()) }
	if ctx.Err() != nil {
		return fail(ctx.Err())
	}
	l, e := openStateDirectory(ledgerPath, false)
	if e != nil {
		return fail(e)
	}
	if _, e = l.InspectState(); e != nil {
		return fail(errors.Join(e, l.Close()))
	}
	if ctx.Err() != nil || os.Geteuid() != 0 {
		return fail(errors.Join(ErrStateDirectory, ctx.Err(), l.Close()))
	}
	// This owner exposes observation only; even empty domains must not retain
	// an incidental journal initialization capability.
	j.initializable = false
	p := &ExecutionDomainLeases{native: &executionDomainLeaseNative{j, l}}
	p.origin = p
	return p, nil
}

// Observe retains both original mutexes, journal first, across two complete
// native identity passes and exact bounded namespace reads. read is a trusted
// native metadata transfer and must obey the single-domain borrowing contract.
// Installation is metadata only; formatting cannot prove fresh commissioning.
func (p *ExecutionDomainLeases) Observe(ctx context.Context, installation string, read func(context.Context, int) (ExecutionBackingObservation, error)) (ExecutionDomainsObservation, error) {
	return p.observe(ctx, installation, read, nil)
}

// ObserveSystem requires fresh complete system observations inside both original
// leases. Supplied callbacks remain trusted metadata, never runtime clearance.
func (p *ExecutionDomainLeases) ObserveSystem(ctx context.Context, installation string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error)) (ExecutionDomainsObservation, error) {
	if system == nil {
		return ExecutionDomainsObservation{}, ErrUnavailable
	}
	return p.observe(ctx, installation, read, system)
}

func (p *ExecutionDomainLeases) observe(ctx context.Context, installation string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error)) (ExecutionDomainsObservation, error) {
	zero := ExecutionDomainsObservation{}
	if p == nil || ctx == nil || read == nil || os.Geteuid() != 0 {
		return zero, ErrStateDirectory
	}
	if !validJournalIdentity(installation) {
		return zero, ErrStateRecord
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.origin != p || p.closed || p.native == nil {
		return zero, ErrDenied
	}
	if p.blocked {
		return zero, ErrStateUncertain
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || l == nil || j.original != j || j.lease == nil || j.lease == l {
		p.blocked = true
		return zero, ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	succeeded := false
	defer func() {
		if !succeeded {
			p.blocked = true
		}
	}()
	if j.failed || j.lease.publicationBlocked || l.publicationBlocked {
		return zero, ErrStateUncertain
	}
	var result ExecutionDomainsObservation
	var ledgerErr, workErr error
	_, journalErr := j.lease.inspectExecutionBindingObjectsLocked(func() {
		_, ledgerErr = l.inspectExecutionBindingObjectsLocked(func() {
			result, workErr = p.observeLocked(ctx, installation, read, system)
		})
	})
	if journalErr != nil {
		return zero, journalErr
	}
	if ledgerErr != nil {
		return zero, ledgerErr
	}
	if workErr != nil {
		return zero, workErr
	}
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if os.Geteuid() != 0 {
		return zero, ErrStateDirectory
	}
	if p.bound && p.binding != result.binding {
		return zero, ErrDrift
	}
	p.binding, p.bound = result.binding, true
	succeeded = true
	return result, nil
}

// Caller holds owner, original journal and ledger mutexes, in that order.
func (p *ExecutionDomainLeases) observeLocked(ctx context.Context, installation string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error)) (ExecutionDomainsObservation, error) {
	zero := ExecutionDomainsObservation{}
	j, l := p.native.journal, p.native.ledger
	before, e := j.inspectLocked("", nil)
	if e != nil {
		return zero, e
	}
	jj, e := j.lease.observeExecutionDomainLocked(ctx, read)
	if e != nil {
		return zero, e
	}
	ll, e := l.observeExecutionDomainLocked(ctx, read)
	if e != nil {
		return zero, e
	}
	binding, e := makeExecutionBinding(installation, ll.identity, jj.identity)
	if e != nil {
		return zero, e
	}
	catalog, first, e := l.inspectExecutionCensusLocked(binding, defaultStateCatalogLimits, nil)
	if e != nil {
		return zero, e
	}
	var firstSystem PoolSystemUse
	checkSystem := func(use PoolSystemUse) error {
		jb, e := propertyStateFilesystem(int(j.lease.directory.Fd()))
		if e != nil {
			return e
		}
		lb, e := propertyStateFilesystem(int(l.directory.Fd()))
		if e != nil {
			return e
		}
		return checkExecutionSystemDomains(jb, lb, jj.identity.backing, ll.identity.backing, catalog, use)
	}
	if system != nil {
		firstSystem, e = system(ctx)
		if e != nil {
			return zero, e
		}
		if e = checkSystem(firstSystem); e != nil {
			return zero, e
		}
	}
	againJ, e := j.lease.observeExecutionDomainLocked(ctx, read)
	if e != nil {
		return zero, e
	}
	againL, e := l.observeExecutionDomainLocked(ctx, read)
	if e != nil {
		return zero, e
	}
	if againJ != jj || againL != ll {
		return zero, ErrDrift
	}
	if system != nil {
		finalSystem, e := system(ctx)
		if e != nil {
			return zero, e
		}
		if !reflect.DeepEqual(firstSystem, finalSystem) {
			return zero, ErrDrift
		}
		if e = checkSystem(finalSystem); e != nil {
			return zero, e
		}
	}
	_, last, e := l.inspectExecutionCensusLocked(binding, defaultStateCatalogLimits, nil)
	if e != nil {
		return zero, e
	}
	if last != first {
		return zero, ErrDrift
	}
	after, e := j.inspectLocked("", nil)
	if e != nil {
		return zero, e
	}
	if before.present != after.present || before.record != after.record || !bytes.Equal(before.data, after.data) || !sameOwnershipMetadata(before.metadata, after.metadata) {
		return zero, ErrDrift
	}
	if e = checkExecutionDomainJournal(binding, after.present, after.record); e != nil {
		return zero, e
	}
	digest := ""
	if after.present {
		h := sha256.Sum256(after.data)
		digest = hex.EncodeToString(h[:])
	}
	return ExecutionDomainsObservation{origin: p, binding: binding, census: last, journalDigest: digest, journalPresent: after.present, journal: after.record, systemObserved: system != nil}, nil
}

// Close excludes observation and closes ledger before journal. It preserves all
// persistent names and does not publish a closed incarnation or prove quiescence.
func (p *ExecutionDomainLeases) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.origin != p {
		return ErrDenied
	}
	if p.closed {
		return nil
	}
	p.closed, p.blocked = true, true
	n := p.native
	p.native = nil
	if n == nil {
		return ErrDenied
	}
	var le, je error
	if n.ledger != nil {
		le = n.ledger.Close()
	}
	if n.journal != nil {
		je = n.journal.close()
	}
	return errors.Join(le, je)
}
