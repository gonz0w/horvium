// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"testing"
	"time"
)

func TestDeadlineReturnDisabledActivationGuardIdentity(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "canceled", true: "deadline"}[deadline], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if deadline {
				cancel()
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			}
			cancel()
			original := ctx.Err()
			r := startupReviewFixture(func(context.Context) (ExecutionDomainsObservation, error) {
				t.Fatal("entry guard observed domains")
				return ExecutionDomainsObservation{}, nil
			}, func() error { t.Fatal("entry guard released leases"); return nil })
			r.activate = func(context.Context) (*ExecutionRestartActivation, error) {
				t.Fatal("entry guard invoked activation")
				return nil, ErrDenied
			}
			if owner, err := r.Activate(ctx); owner != nil || err != original || r.blocked || r.transferred {
				t.Fatal("disabled entry guard altered identity or transfer state")
			}
		})
	}
}
