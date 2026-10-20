// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"

	"github.com/gonz0w/horvium/internal/storage"
)

// The existing registry is claimed before entering the fixed native factory.
// Entry and confirmed release belong to this attempt, never supplied clearance.
func nativeControlOwner(ctx context.Context, t storage.ControlTargets, action string) controlOwnerAttempt {
	result := controlOwnerAttempt{released: true}
	result.owner, result.err = commissionExecutionOwner(ctx, nativeExecutionOwners, executionRoot, func(c context.Context) (executionOwnerSource, error) {
		result.entered = true
		wrap := func(source executionOwnerSource) executionOwnerSource {
			release := source.release
			source.release = func() error {
				rooted := executionRoot()
				e := release()
				if !rooted || !executionRoot() {
					e = errors.Join(e, ErrPermission)
				}
				result.released = e == nil
				return e
			}
			return source
		}
		if action == "commission-fresh" {
			result.released = false
			a, e := activateNativeFreshExecutionJournal(c, t.JournalDirectory, t.LedgerDirectory)
			if a == nil {
				return executionOwnerSource{}, e
			}
			return wrap(executionOwnerSource{observe: a.Observe, finish: a.FinishIdle, release: a.Close}), e
		}
		if action != "start-closed" {
			return executionOwnerSource{}, storage.ErrInvalid
		}
		system, e := NewSystemUseSource()
		if e != nil {
			return executionOwnerSource{}, e
		}
		result.released = false
		// Fixed empty-history review. The original Activate checks the complete
		// canonical empty census under its original locks before entropy/publication.
		// The policy-aware producer remains unchanged and cannot waive #305 here.
		r, e := storage.ReviewExecutionStartup(c, t.JournalDirectory, t.LedgerDirectory, readNativeExecutionBacking, system.Collect, readNativeActivationBoot)
		if r == nil {
			return executionOwnerSource{}, e
		}
		reviewed := executionOwnerSource{observe: r.Observe, finish: func(context.Context) error { return storage.ErrDenied }, release: r.Close}
		if e != nil {
			return wrap(reviewed), e
		}
		a, e := r.Activate(c)
		if a == nil {
			return wrap(reviewed), e
		}
		return wrap(executionOwnerSource{observe: a.Observe, finish: a.FinishIdle, release: a.Close}), e
	})
	return result
}

func nativeControlInspect(ctx context.Context, t storage.ControlTargets) controlInspectionAttempt {
	result := controlInspectionAttempt{released: true}
	if !executionRoot() {
		result.err = ErrPermission
		return result
	}
	system, e := NewSystemUseSource()
	if e != nil {
		result.err = e
		return result
	}
	result.released = false
	pair, e := storage.OpenExecutionDomainLeases(ctx, t.JournalDirectory, t.LedgerDirectory)
	if pair == nil {
		result.err = e
		return result
	}
	if e == nil {
		result.state, e = pair.InspectSystem(ctx, readNativeExecutionBacking, system.Collect)
	}
	rooted := executionRoot()
	ce := pair.Close()
	if !rooted || !executionRoot() {
		ce = errors.Join(ce, ErrPermission)
	}
	result.released = ce == nil
	result.err = errors.Join(e, ce)
	return result
}
