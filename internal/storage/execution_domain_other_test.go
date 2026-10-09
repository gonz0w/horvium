// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"context"
	"testing"
)

func TestExecutionDomainUnavailable(t *testing.T) {
	called := false
	read := func(context.Context, int) (ExecutionBackingObservation, error) {
		called = true
		return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{1, 2}}, nil
	}
	for _, s := range []*StateDirectory{nil, {}} {
		got, e := s.ObserveExecutionDomain(context.Background(), read)
		if e != ErrUnavailable || got != (ExecutionDomainObservation{}) || called {
			t.Fatal("off-target observation accepted", e)
		}
	}
}
