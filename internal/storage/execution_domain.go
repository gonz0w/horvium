// SPDX-License-Identifier: BSD-3-Clause
package storage

// ExecutionBackingObservation transfers bounded native facts from the trusted
// FreeBSD collector. Supplied values are metadata, never commissioning authority.
type ExecutionBackingObservation struct {
	Kind          string
	UFS           [2]uint32
	Pool, Dataset uint64
}

// ExecutionDomainObservation retains private identity and original lease
// provenance. It is not serialized, a publication receipt or launch permission.
type ExecutionDomainObservation struct {
	origin   *StateDirectory
	identity executionDomainIdentity
}

func (b ExecutionBackingObservation) identity() (executionBackingIdentity, error) {
	i := executionBackingIdentity{b.Kind, b.UFS, b.Pool, b.Dataset}
	if !i.valid() {
		return executionBackingIdentity{}, ErrStateRecord
	}
	return i, nil
}
