// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"sort"
	"strings"
)

const maxOwnershipCatalogBytes uint64 = 64 << 20

// OwnershipSummary describes a validated private record, not enrollment or
// native identity. It cannot authorize a storage operation.
type OwnershipSummary struct {
	PoolGUID string
	Revision uint64
	Bytes    uint64
}

// OwnershipCatalog is a bounded, independent observation of published ownership
// metadata only. It is not a transaction, recovery result or full job store.
type OwnershipCatalog struct {
	Records []OwnershipSummary
	Bytes   uint64
}

// Classify the complete bounded name set before reading any record. Unknown
// names and orphan temporaries never turn into an empty or partial catalog.
func ownershipCatalogNames(names []string, limit int) ([]string, error) {
	if limit < 1 || limit > maxRecords || len(names) > limit+1 {
		return nil, ErrLimit
	}
	seen := make(map[string]bool, len(names))
	guids := make([]string, 0, len(names))
	writer := false
	for _, name := range names {
		if seen[name] {
			return nil, ErrDrift
		}
		seen[name] = true
		if name == "writer.lock" {
			writer = true
			continue
		}
		if strings.HasPrefix(name, ".ownership-") && strings.HasSuffix(name, ".tmp") {
			nonce := strings.TrimSuffix(strings.TrimPrefix(name, ".ownership-"), ".tmp")
			if len(nonce) == 32 && strings.IndexFunc(nonce, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) < 0 {
				return nil, ErrStateUncertain
			}
		}
		if !strings.HasPrefix(name, "ownership-") || !strings.HasSuffix(name, ".json") {
			return nil, ErrStateRecord
		}
		guid := strings.TrimSuffix(strings.TrimPrefix(name, "ownership-"), ".json")
		if len(guid) > 20 || !validGUID(guid) {
			return nil, ErrStateRecord
		}
		guids = append(guids, guid)
	}
	if !writer {
		return nil, ErrDrift
	}
	if len(guids) > limit {
		return nil, ErrLimit
	}
	sort.Strings(guids)
	return guids, nil
}
