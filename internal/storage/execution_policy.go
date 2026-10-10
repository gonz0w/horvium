// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"github.com/gonz0w/horvium/internal/inventory"
)

// Trusted native observations only. The original locked ledger supplies policy;
// no caller policy, boolean, descriptor or reconstructed receipt grants authority.
type ExecutionStartupPolicySource func(context.Context, Policy) (Observation, []inventory.DiskObservation, PoolSystemUse, error)

func checkExecutionStartupPolicyFacts(p Policy, o Observation, disks []inventory.DiskObservation, system PoolSystemUse) error {
	if o.Eligible {
		return ErrDenied
	}
	if err := ValidatePropertyMembers(p, o, disks); err != nil {
		return err
	}
	if err := CheckPropertySystemExclusions(o, system); err != nil {
		return err
	}
	return CheckPropertySystemProviders(o.PoolGUID, system, disks)
}
func checkExecutionStartupPolicyCatalog(c StateCatalog) error {
	if len(c.Jobs) != 0 || c.Nonterminal != 0 || c.Terminal != 0 {
		return ErrDenied
	}
	return nil
}
