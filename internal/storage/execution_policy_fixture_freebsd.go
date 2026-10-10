//go:build freebsd && horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"os"
)

// SeedExecutionPolicyFixture exists only in explicitly tagged disposable test
// binaries. It creates new domains with the actual commissioning/publisher
// pipeline. Closing controlled metadata is fixture setup, not worker shutdown.
// Normal product builds cannot call this bridge or obtain its original receipt.
func SeedExecutionPolicyFixture(ctx context.Context, journal, ledger string, read func(context.Context, int) (ExecutionBackingObservation, error), system func(context.Context) (PoolSystemUse, error), boot func(context.Context) ([16]byte, error), policies []Policy) (result error) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_CLOSED_STARTUP_OWNER_VM") != "1" || os.Geteuid() != 0 {
		return ErrDenied
	}
	a, e := activateFreshExecutionJournal(ctx, journal, ledger, read, system, boot)
	if e != nil {
		return e
	}
	defer func() { result = errors.Join(result, a.fresh.close()) }()
	p := a.fresh.pair
	for _, policy := range policies {
		if e = p.publishJournaledOwnership(ctx, policy, 0, read, nil); e != nil {
			return e
		}
	}
	before := p.native.journal.ack
	next := before
	next.Generation++
	next.Owner.State = "closed"
	if e = p.native.journal.publish(next, nil); e != nil {
		return e
	}
	got, e := p.ObserveSystem(ctx, before.Binding.Installation, read, system)
	if e != nil {
		return e
	}
	if got.journal != next || got.census != next.Census {
		return ErrDrift
	}
	return p.Close()
}
