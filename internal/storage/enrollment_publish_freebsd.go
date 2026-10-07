// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
	"os"
	"time"
)

// Complete census, exact predecessor/baseline and publication share one original
// lease/mutex. The accepted descriptor-relative sync/rename engine is unchanged.
func (p PropertyEnrollmentState) publishEnrollmentMetadata(ctx context.Context, rootContext string, o Observation, disks []inventory.DiskObservation, u PoolSystemUse, now func() time.Time, limits stateCatalogLimits, wrap func(ownershipPublication) ownershipPublication) error {
	if p.lease == nil || os.Geteuid() != 0 {
		return ErrStateDirectory
	}
	if ctx == nil || now == nil {
		return ErrInvalid
	}
	if !limits.valid() {
		return ErrLimit
	}
	var guardErr error
	var stagedTemporary string
	guard := func() error {
		if e := ctx.Err(); e != nil {
			guardErr = e
			return e
		}
		err := p.plan.Validate(now(), rootContext, p.plan.review.Policy, o, disks, u)
		if err != nil {
			guardErr = err
		}
		return err
	}
	if e := guard(); e != nil {
		return e
	}
	data, err := EncodeOwnershipRecord(p.plan.review.Policy)
	if err != nil {
		return err
	}
	s := p.lease
	s.mu.Lock()
	defer s.mu.Unlock()
	inspect := func(temporary string) (StateCatalog, error) {
		if e := guard(); e != nil {
			return StateCatalog{}, e
		}
		c, digest, e := s.inspectEnrollmentStateLocked(p.plan.review.Policy, p.previous, u, limits, nil, temporary)
		if e != nil {
			guardErr = e
			return StateCatalog{}, e
		}
		if digest != p.digest {
			return StateCatalog{}, ErrDrift
		}
		if e = guard(); e != nil {
			return StateCatalog{}, e
		}
		if temporary != "" {
			stagedTemporary = temporary
		}
		return c, nil
	}
	predecessor := func() error {
		if e := guard(); e != nil {
			return e
		}
		current, e := s.readOwnershipLocked(p.plan.review.Policy.PoolGUID)
		if e != nil {
			return e
		}
		before, e := EncodeOwnershipRecord(p.previous)
		if e != nil {
			return e
		}
		after, e := EncodeOwnershipRecord(current)
		if e != nil {
			return e
		}
		if !bytes.Equal(before, after) {
			return ErrDrift
		}
		return nil
	}
	wrapped := func(tx ownershipPublication) ownershipPublication {
		// Fixture wrappers surround actual syscalls; the mandatory guard stays inside
		// their verify/rename boundaries, including expiry after staging verification.
		for _, slot := range []*func() error{&tx.verify, &tx.rename} {
			original := *slot
			*slot = func() error {
				if e := guard(); e != nil {
					guardErr = e
					return e
				}
				return original()
			}
		}
		rename := tx.rename
		tx.rename = func() error {
			if stagedTemporary == "" {
				return ErrStateUncertain
			}
			if _, e := inspect(stagedTemporary); e != nil {
				return e
			}
			return rename()
		}
		if wrap != nil {
			tx = wrap(tx)
		}
		return tx
	}
	err = s.publishStateInspectedLocked("ownership-"+p.plan.review.Policy.PoolGUID+".json", ".ownership-", data, p.previous.Revision,
		func(c StateCatalog) error {
			return allowMixedOwnershipPublication(c, p.plan.review.Policy, p.previous.Revision, uint64(len(data)), limits)
		}, predecessor, inspect, wrapped)
	// The common engine maps unknown pre-rename errors to a storage error. Preserve
	// typed cancellation/expiry only when cleanup succeeded; uncertainty wins.
	if guardErr != nil && errors.Is(err, ErrStateDirectory) {
		return guardErr
	}
	return err
}
