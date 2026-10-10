// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"os"
	"time"
)

// ObserveExecutionDomain borrows the original directory descriptor only for a
// trusted native read. The callback must not retain/close it or re-enter this
// lease. The original mutex and full object/ACL/name/mount checks surround the
// read. Supplied callback facts cannot commission an owner or authorize writes.
func (s *StateDirectory) ObserveExecutionDomain(ctx context.Context, read func(context.Context, int) (ExecutionBackingObservation, error)) (ExecutionDomainObservation, error) {
	if ctx == nil || s == nil || read == nil || os.Geteuid() != 0 {
		return ExecutionDomainObservation{}, ErrStateDirectory
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.observeExecutionDomainLocked(ctx, read)
}

// Caller holds the original lease mutex. The fixed native collector supplies
// its own bounded probe context when called by a joint-domain observer.
func (s *StateDirectory) observeExecutionDomainLocked(ctx context.Context, read func(context.Context, int) (ExecutionBackingObservation, error)) (ExecutionDomainObservation, error) {
	if ctx == nil || s == nil || read == nil || os.Geteuid() != 0 {
		return ExecutionDomainObservation{}, ErrStateDirectory
	}
	if s.propertyOrigin != s {
		return ExecutionDomainObservation{}, ErrDenied
	}
	if s.publicationBlocked {
		return ExecutionDomainObservation{}, ErrStateUncertain
	}
	if ctx.Err() != nil {
		return ExecutionDomainObservation{}, ctx.Err()
	}
	var backing executionBackingIdentity
	var readErr error
	objects, err := s.inspectExecutionBindingObjectsLocked(func() {
		b, e := read(ctx, int(s.directory.Fd()))
		readErr = e
		if e == nil {
			backing, readErr = b.identity()
		}
	})
	if err != nil {
		return ExecutionDomainObservation{}, err
	}
	if readErr != nil {
		return ExecutionDomainObservation{}, readErr
	}
	if ctx.Err() != nil {
		return ExecutionDomainObservation{}, ctx.Err()
	}
	if os.Geteuid() != 0 || s.publicationBlocked {
		return ExecutionDomainObservation{}, ErrDenied
	}
	identity := executionDomainIdentity{backing, objects.directory, objects.lock}
	if backing.kind != objects.mount.kind || !identity.valid() {
		return ExecutionDomainObservation{}, ErrDrift
	}
	return ExecutionDomainObservation{origin: s, identity: identity}, nil
}
