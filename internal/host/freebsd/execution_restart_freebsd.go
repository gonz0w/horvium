// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"os"
)

// Fixed original root composition and the same process-wide reservation as
// commissioning/review. No operator route, worker or native launch is granted.
func activateNativeClosedExecutionOwner(ctx context.Context, journal, ledger string) (*commissionedExecutionOwner, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	return commissionExecutionOwner(ctx, nativeExecutionOwners, func() bool { return os.Geteuid() == 0 }, func(c context.Context) (executionOwnerSource, error) {
		source, e := NewSystemUseSource()
		if e != nil {
			return executionOwnerSource{}, e
		}
		policy, e := newExecutionStartupPolicySource()
		if e != nil {
			return executionOwnerSource{}, e
		}
		r, e := storage.ReviewExecutionPolicyStartup(c, journal, ledger, readNativeExecutionBacking, source.Collect, readNativeActivationBoot, policy)
		if e != nil {
			return executionOwnerSource{}, e
		}
		a, e := r.Activate(c)
		if e != nil {
			return executionOwnerSource{}, errors.Join(e, r.Close())
		}
		return executionOwnerSource{observe: a.Observe, finish: a.FinishIdle, release: a.Close}, nil
	})
}
