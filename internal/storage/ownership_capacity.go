// SPDX-License-Identifier: BSD-3-Clause
package storage

// Fixed production ceilings; reduced bounds are package-private native fixtures.
// No operator tuning, job budget or storage admission authority is provided.
type ownershipPublicationLimits struct {
	records int
	bytes   uint64
}

func (limits ownershipPublicationLimits) valid() bool {
	return limits.records >= 1 && limits.records <= maxRecords && limits.bytes > 0 && limits.bytes <= maxOwnershipCatalogBytes
}

// The catalog must come from a complete locked native inspection. Reserve the
// entire new record while the old destination still exists; never evict state.
func allowOwnershipPublication(c OwnershipCatalog, next Policy, expected, size uint64, limits ownershipPublicationLimits) error {
	if !limits.valid() || size == 0 || size > maxStateRecordBytes || len(c.Records) > limits.records || c.Bytes > limits.bytes {
		return ErrLimit
	}
	if !validOwnershipPublicationRevision(next.Revision, expected) {
		return ErrDrift
	}
	found := false
	for _, record := range c.Records {
		if record.PoolGUID == next.PoolGUID {
			if expected == 0 || record.Revision != expected {
				return ErrDrift
			}
			found = true
		}
	}
	if !found && expected != 0 {
		return ErrDrift
	}
	if !found && len(c.Records) == limits.records || size > limits.bytes-c.Bytes {
		return ErrLimit
	}
	return nil
}
