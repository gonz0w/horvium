// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"crypto/sha256"
	"reflect"
)

// Original closed review with complete policy-only validation. Trusted root
// composition supplies fixed native facts; no runtime consumer is registered.
func ReviewExecutionPolicyStartup(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error), policy ExecutionStartupPolicySource) (*ExecutionStartupReview, error) {
	if policy == nil {
		return nil, ErrDenied
	}
	r, _, e := reviewExecutionStartupPolicy(ctx, journal, ledger, read, system, boot, policy)
	return r, e
}

// Caller holds original pair, journal and ledger mutexes. Stream each policy
// through fixed native observations without reentering ReadOwnership's mutex.
// Retain only bounded canonical digests, never full inventory or caller slices.
func (n *executionStartupNative) inspectPoliciesLocked(ctx context.Context, got ExecutionDomainsObservation) error {
	l := n.pair.native.ledger
	catalog, census, e := l.inspectExecutionCensusLocked(got.binding, defaultStateCatalogLimits, nil)
	if e != nil {
		return e
	}
	if census != got.census {
		return ErrDrift
	}
	if e = checkExecutionStartupPolicyCatalog(catalog); e != nil {
		return e
	}
	digests := make(map[string][32]byte, len(catalog.Ownership))
	_, e = l.inspectStateStreamLocked(defaultStateCatalogLimits, nil, "", nil, func(entry stateRecordName, data []byte) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if entry.poolGUID == "" {
			return ErrDenied
		}
		p, e := DecodeOwnershipRecord(data)
		if e != nil {
			return e
		}
		canonical, e := EncodeOwnershipRecord(p)
		if e != nil {
			return e
		}
		borrowed, e := DecodeOwnershipRecord(canonical)
		if e != nil {
			return e
		}
		o, disks, use, e := n.policy(ctx, borrowed)
		if e != nil {
			return e
		}
		if e = checkExecutionStartupPolicyFacts(p, o, disks, use); e != nil {
			return e
		}
		jb, e := propertyStateFilesystem(int(n.pair.native.journal.lease.directory.Fd()))
		if e != nil {
			return e
		}
		lb, e := propertyStateFilesystem(int(l.directory.Fd()))
		if e != nil {
			return e
		}
		jID, e := n.pair.native.journal.lease.observeExecutionDomainLocked(ctx, n.read)
		if e != nil {
			return e
		}
		lID, e := l.observeExecutionDomainLocked(ctx, n.read)
		if e != nil {
			return e
		}
		if e = checkExecutionSystemDomains(jb, lb, jID.identity.backing, lID.identity.backing, catalog, use); e != nil {
			return e
		}
		digests[p.PoolGUID] = sha256.Sum256(canonical)
		return ctx.Err()
	})
	if e != nil {
		return e
	}
	if len(digests) != len(catalog.Ownership) {
		return ErrDrift
	}
	_, last, e := l.inspectExecutionCensusLocked(got.binding, defaultStateCatalogLimits, nil)
	if e != nil {
		return e
	}
	if last != census {
		return ErrDrift
	}
	if n.reviewed && !reflect.DeepEqual(digests, n.policyDigests) {
		return ErrDrift
	}
	if !n.reviewed {
		n.policyDigests = digests
	}
	return ctx.Err()
}
