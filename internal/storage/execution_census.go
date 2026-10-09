// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash"
)

// Pure bounded streaming consumer of validated ledger bytes, not a native
// namespace observation, original lease, durable acknowledgement or clearance.
// Single caller owns input throughout each call, under the future native writer.
// Copies cannot alter the original consumer or finish a partial observation.
type executionCensus struct {
	original *executionCensus
	entries  []stateRecordName
	index    int
	catalog  *stateCatalogBuilder
	hash     hash.Hash
	retired  bool
}

func newExecutionCensus(binding executionJournalBinding, names []string, limits stateCatalogLimits) (*executionCensus, error) {
	if !validExecutionJournalBinding(binding) {
		return nil, ErrStateRecord
	}
	entries, err := stateCatalogNames(names, limits)
	if err != nil {
		return nil, err
	}
	b := &executionCensus{entries: entries, catalog: newStateCatalogBuilder(limits), hash: sha256.New()}
	b.original = b
	_, _ = b.hash.Write([]byte("horvium:execution-census:v1\n"))
	// Canonical bounded binding includes installation, directory, original lock
	// and both independently established system-backing digests. Caller-supplied
	// hashes here are metadata, not proof that those original objects exist.
	wire, _ := json.Marshal(binding)
	censusUint(b.hash, uint64(len(wire)))
	_, _ = b.hash.Write(wire)
	censusUint(b.hash, uint64(len(entries)+1))
	return b, nil
}
func (b *executionCensus) add(name string, data []byte) error {
	if b == nil || b.original != b || b.retired {
		return ErrDenied
	}
	fail := func(err error) error { b.retired = true; return err }
	if b.index >= len(b.entries) || name != b.entries[b.index].name {
		return fail(ErrDrift)
	}
	if err := b.catalog.add(b.entries[b.index], data); err != nil {
		return fail(err)
	}
	censusRecord(b.hash, name, data)
	b.index++
	return nil
}
func (b *executionCensus) finish() (string, error) {
	if b == nil || b.original != b || b.retired {
		return "", ErrDenied
	}
	b.retired = true
	if b.index != len(b.entries) {
		return "", ErrDrift
	}
	// All accepted records sort before writer.lock. Its zero bytes and independently
	// bound original identity are included even for an otherwise empty namespace.
	censusRecord(b.hash, "writer.lock", nil)
	return hex.EncodeToString(b.hash.Sum(nil)), nil
}
func censusUint(h hash.Hash, n uint64) {
	var bytes [8]byte
	binary.BigEndian.PutUint64(bytes[:], n)
	_, _ = h.Write(bytes[:])
}
func censusRecord(h hash.Hash, name string, data []byte) {
	censusUint(h, uint64(len(name)))
	_, _ = h.Write([]byte(name))
	censusUint(h, uint64(len(data)))
	sum := sha256.Sum256(data)
	_, _ = h.Write(sum[:])
}
