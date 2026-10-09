// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
)

// Private binding representation only. Supplied backing facts and installation
// identities are metadata, never native provenance, commissioning or authority.
type executionBackingIdentity struct {
	kind          string
	ufs           [2]uint32 // On-disk UFS fs_id, not a kernel mount/device identifier.
	pool, dataset uint64    // Persistent ZFS pool and dataset GUIDs.
}
type executionObjectIdentity struct {
	inode, generation uint64
	birthSeconds      int64
	birthNanos        uint32
}
type executionDomainIdentity struct {
	backing         executionBackingIdentity
	directory, lock executionObjectIdentity
}

func makeExecutionBinding(installation string, ledger, journal executionDomainIdentity) (executionJournalBinding, error) {
	if !validJournalIdentity(installation) || !ledger.valid() || !journal.valid() {
		return executionJournalBinding{}, ErrStateRecord
	}
	if ledger.backing == journal.backing {
		// Concurrent domains cannot alias an inode on the same filesystem,
		// regardless of supplied generation/birth metadata. ZFS domains must
		// have distinct datasets; this alone does not prove restore independence.
		if ledger.backing.kind == "zfs" || ledger.directory.inode == journal.directory.inode || ledger.lock.inode == journal.lock.inode || ledger.directory.inode == journal.lock.inode || ledger.lock.inode == journal.directory.inode {
			return executionJournalBinding{}, ErrDenied
		}
	}
	return executionJournalBinding{
		Installation:   installation,
		Directory:      executionObjectPairDigest("directory", ledger, journal),
		Lock:           executionObjectPairDigest("lock", ledger, journal),
		LedgerBacking:  executionBackingDigest(ledger.backing),
		JournalBacking: executionBackingDigest(journal.backing),
	}, nil
}

func (b executionBackingIdentity) valid() bool {
	switch b.kind {
	case "ufs":
		return b.ufs[0] != 0 && b.ufs[1] != 0 && b.pool == 0 && b.dataset == 0
	case "zfs":
		return b.pool != 0 && b.dataset != 0 && b.ufs == ([2]uint32{})
	}
	return false
}
func (o executionObjectIdentity) valid() bool {
	return o.inode != 0 && o.generation != 0 && o.birthSeconds > 0 && o.birthNanos < 1000000000
}
func (d executionDomainIdentity) valid() bool {
	return d.backing.valid() && d.directory.valid() && d.lock.valid() && d.directory.inode != d.lock.inode
}
func executionBackingDigest(b executionBackingIdentity) string {
	h := sha256.New()
	_, _ = h.Write([]byte("horvium:execution-backing:v1\n"))
	executionBindingString(h, b.kind)
	censusUint(h, uint64(b.ufs[0]))
	censusUint(h, uint64(b.ufs[1]))
	censusUint(h, b.pool)
	censusUint(h, b.dataset)
	return hex.EncodeToString(h.Sum(nil))
}
func executionObjectPairDigest(role string, ledger, journal executionDomainIdentity) string {
	h := sha256.New()
	_, _ = h.Write([]byte("horvium:execution-objects:v1\n"))
	executionBindingString(h, role)
	for i, d := range []executionDomainIdentity{ledger, journal} {
		censusUint(h, uint64(i)) // Fixed ordered ledger and journal roles.
		executionBindingString(h, executionBackingDigest(d.backing))
		o := d.directory
		if role == "lock" {
			o = d.lock
		}
		censusUint(h, o.inode)
		censusUint(h, o.generation)
		censusUint(h, uint64(o.birthSeconds))
		censusUint(h, uint64(o.birthNanos))
	}
	return hex.EncodeToString(h.Sum(nil))
}
func executionBindingString(h hash.Hash, s string) {
	censusUint(h, uint64(len(s)))
	_, _ = h.Write([]byte(s))
}
