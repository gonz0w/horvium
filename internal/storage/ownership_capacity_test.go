// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"testing"
)

func TestOwnershipPublicationCapacity(t *testing.T) {
	p := Policy{PoolGUID: "100", Revision: 1}
	empty := OwnershipCatalog{}
	full := OwnershipCatalog{Records: []OwnershipSummary{{PoolGUID: "100", Revision: 1, Bytes: 100}}, Bytes: 100}
	check := func(c OwnershipCatalog, next Policy, expected, size uint64, limits ownershipPublicationLimits, want error) {
		t.Helper()
		got := allowOwnershipPublication(c, next, expected, size, limits)
		if !errors.Is(got, want) {
			t.Fatal("capacity/revision decision", got, "want", want)
		}
	}
	check(empty, p, 0, 50, ownershipPublicationLimits{1, 50}, nil)
	check(empty, p, 0, 51, ownershipPublicationLimits{1, 50}, ErrLimit)
	q := p
	q.PoolGUID = "200"
	check(full, q, 0, 50, ownershipPublicationLimits{1, 150}, ErrLimit)
	check(full, q, 0, 50, ownershipPublicationLimits{2, 150}, nil)
	p.Revision = 2
	// Replacement leaves the old record present until rename: reserve 100+50,
	// even though the eventual published bytes would drop to 50.
	check(full, p, 1, 50, ownershipPublicationLimits{1, 150}, nil)
	check(full, p, 1, 50, ownershipPublicationLimits{1, 149}, ErrLimit)
	check(full, p, 0, 50, ownershipPublicationLimits{1, 150}, ErrDrift)
	check(empty, p, 1, 50, ownershipPublicationLimits{1, 150}, ErrDrift)
	check(full, p, 2, 50, ownershipPublicationLimits{1, 150}, ErrDrift)
	p.Revision = 0
	check(full, p, ^uint64(0), 50, ownershipPublicationLimits{1, 150}, ErrDrift)
	p.Revision = 2
	for _, limits := range []ownershipPublicationLimits{{0, 150}, {-1, 150}, {maxRecords + 1, 150}, {1, 0}, {1, maxOwnershipCatalogBytes + 1}} {
		check(full, p, 1, 50, limits, ErrLimit)
	}
	for _, size := range []uint64{0, maxStateRecordBytes + 1, ^uint64(0)} {
		check(full, p, 1, size, ownershipPublicationLimits{1, maxOwnershipCatalogBytes}, ErrLimit)
	}
	oversized := full
	oversized.Bytes = ^uint64(0)
	check(oversized, p, 1, 50, ownershipPublicationLimits{1, 150}, ErrLimit)
}
