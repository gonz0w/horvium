// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"context"
	"testing"
)

func TestExecutionSystemObservationUnavailableOffTarget(t *testing.T) {
	called := false
	for _, owner := range []*ExecutionDomainLeases{nil, {}, {}} {
		got, e := owner.ObserveSystem(context.Background(), "", func(context.Context, int) (ExecutionBackingObservation, error) {
			called = true
			return ExecutionBackingObservation{}, nil
		}, func(context.Context) (PoolSystemUse, error) { called = true; return PoolSystemUse{}, nil })
		if e != ErrUnavailable || got != (ExecutionDomainsObservation{}) || called {
			t.Fatal("off-target native system observation", e)
		}
	}
}
