// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"context"
	"testing"
)

func TestExecutionDomainsUnavailable(t *testing.T) {
	p, e := OpenExecutionDomainLeases(context.Background(), "/journal", "/ledger")
	if p != nil || e != ErrUnavailable {
		t.Fatal("off-target leases", e)
	}
	called := false
	read := func(context.Context, int) (ExecutionBackingObservation, error) {
		called = true
		return ExecutionBackingObservation{}, nil
	}
	for _, p := range []*ExecutionDomainLeases{nil, {}} {
		o, e := p.Observe(context.Background(), "11111111111111111111111111111111", read)
		if e != ErrUnavailable || o != (ExecutionDomainsObservation{}) || called {
			t.Fatal("off-target joint observation", e)
		}
	}
}
