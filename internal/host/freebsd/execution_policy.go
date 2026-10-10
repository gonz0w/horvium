// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// Fixed producers borrow canonical policy supplied by the original already-
// locked ledger. They never reopen the state directory or read its mutex again.
func newExecutionStartupPolicySource() (storage.ExecutionStartupPolicySource, error) {
	pool, e := NewPoolPropertySource()
	if e != nil {
		return nil, e
	}
	providers, e := NewPoolProviderSource()
	if e != nil {
		return nil, e
	}
	system, e := NewSystemUseSource()
	if e != nil {
		return nil, e
	}
	return func(ctx context.Context, p storage.Policy) (storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse, error) {
		zero := func(e error) (storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse, error) {
			return storage.Observation{}, nil, storage.PoolSystemUse{}, e
		}
		o, e := pool.Collect(ctx, p.PoolGUID, p.PoolName)
		if e != nil {
			return zero(e)
		}
		d, e := providers.Collect(ctx, o)
		if e != nil {
			return zero(e)
		}
		u, e := system.Collect(ctx)
		if e != nil {
			return zero(e)
		}
		if e = ctx.Err(); e != nil {
			return zero(e)
		}
		return o, d, u, nil
	}, nil
}
