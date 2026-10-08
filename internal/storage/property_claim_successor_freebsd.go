// SPDX-License-Identifier: BSD-3-Clause
package storage

// PublishPropertyJobSuccessor acknowledges one exact observing/reconciliation/
// success metadata append following an acknowledgement by this original claim.
// Decoded or raw-published records cannot supply that volatile lineage. Native
// control/quiescence, observation attribution and outcome authorization belong
// to the complete original backend; this metadata primitive supplies none.
// A failed valid attempt cannot retry. Success releases no claim or hold.
func (c *PropertyStateClaim) PublishPropertyJobSuccessor(next PropertyJob) (PropertyJob, error) {
	return c.publishClaimPropertyJobBounded(next, defaultStateCatalogLimits, nil, false)
}
