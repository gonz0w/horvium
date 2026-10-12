// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"context"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

type overviewReader interface {
	Overview(context.Context, string) (observation.Overview, error)
	Pools(context.Context, observation.Query) (observation.Page[inventory.Pool], error)
}

// Compose two bounded cached reads. Never retry with a current generation if
// the first observation is evicted between reads; discard both projections.
func overviewPoolPage(ctx context.Context, r overviewReader, id string) (observation.Overview, observation.Page[inventory.Pool], error) {
	o, e := r.Overview(ctx, id)
	if e != nil {
		return observation.Overview{}, observation.Page[inventory.Pool]{}, e
	}
	p, e := r.Pools(ctx, observation.Query{Generation: o.Generation, Sort: "name", Limit: 100})
	if e != nil {
		return observation.Overview{}, observation.Page[inventory.Pool]{}, e
	}
	return o, p, nil
}
