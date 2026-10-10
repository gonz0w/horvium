// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
)

// Streaming one-record successor projection. It retains one bounded successor,
// never ledger bodies, and returns no digest from a partial/foreign observation.
// This is metadata, not a transaction receipt or commissioning authority.
type executionCensusChange struct {
	original          *executionCensusChange
	before, after     *executionCensus
	name              string
	data              []byte
	previous          string
	inserted, retired bool
}

func newExecutionCensusChange(binding executionJournalBinding, names []string, name string, data []byte, limits stateCatalogLimits) (*executionCensusChange, error) {
	if !limits.valid() || len(data) == 0 || len(data) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	target, e := stateCatalogNames([]string{name, "writer.lock"}, limits)
	if e != nil || len(target) != 1 {
		return nil, ErrStateRecord
	}
	if e = newStateCatalogBuilder(limits).add(target[0], data); e != nil {
		return nil, e
	}
	before, e := newExecutionCensus(binding, names, limits)
	if e != nil {
		return nil, e
	}
	successorNames := slices.Clone(names)
	if !slices.Contains(successorNames, name) {
		successorNames = append(successorNames, name)
	}
	after, e := newExecutionCensus(binding, successorNames, limits)
	if e != nil {
		return nil, e
	}
	b := &executionCensusChange{before: before, after: after, name: name, data: slices.Clone(data)}
	b.original = b
	return b, nil
}
func (b *executionCensusChange) add(name string, data []byte) error {
	if b == nil || b.original != b || b.retired {
		return ErrDenied
	}
	fail := func(e error) error { b.retired = true; return e }
	if e := b.before.add(name, data); e != nil {
		return fail(e)
	}
	if !b.inserted && b.after.index < len(b.after.entries) && b.after.entries[b.after.index].name == b.name {
		if e := b.after.add(b.name, b.data); e != nil {
			return fail(e)
		}
		b.inserted = true
	}
	if name == b.name {
		b.previous = executionRecordDigest(data)
		return nil
	}
	if e := b.after.add(name, data); e != nil {
		return fail(e)
	}
	return nil
}
func (b *executionCensusChange) finish() (string, string, string, error) {
	if b == nil || b.original != b || b.retired {
		return "", "", "", ErrDenied
	}
	b.retired = true
	before, e := b.before.finish()
	if e != nil {
		return "", "", "", e
	}
	if !b.inserted {
		if e = b.after.add(b.name, b.data); e != nil {
			return "", "", "", e
		}
	}
	after, e := b.after.finish()
	if e != nil {
		return "", "", "", e
	}
	return before, after, b.previous, nil
}
func executionRecordDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
