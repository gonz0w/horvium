// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"sort"
	"strings"
)

// JobSummary describes validated lifecycle metadata. It cannot authorize
// execution, reconciliation, pruning or release of a reservation.
type JobSummary struct {
	ID, SubmissionID string
	Revision         uint64
	State            JobState
	Bytes            uint64
}

// StateCatalog is an independent, bounded census of ownership and job records.
// It is an observation, not a cross-record transaction or recovered/admitted job.
type StateCatalog struct {
	Ownership             []OwnershipSummary
	Jobs                  []JobSummary
	Bytes                 uint64
	Nonterminal, Terminal int
}

type stateCatalogLimits struct {
	ownership, nonterminal, terminal int
	bytes                            uint64
}

var defaultStateCatalogLimits = stateCatalogLimits{maxRecords, 64, 1000, maxOwnershipCatalogBytes}

func (l stateCatalogLimits) valid() bool {
	return l.ownership > 0 && l.ownership <= maxRecords && l.nonterminal > 0 && l.nonterminal <= 64 && l.terminal > 0 && l.terminal <= 1000 && l.bytes > 0 && l.bytes <= maxOwnershipCatalogBytes
}

type stateRecordName struct{ name, poolGUID, jobID string }

// Validate all names before opening records. No recognized temporary is ignored;
// unknown entries never silently narrow the census.
func stateCatalogNames(names []string, limits stateCatalogLimits) ([]stateRecordName, error) {
	if !limits.valid() || len(names) > limits.ownership+limits.nonterminal+limits.terminal+1 {
		return nil, ErrLimit
	}
	seen := make(map[string]bool, len(names))
	result := make([]stateRecordName, 0, len(names))
	writer, owners, jobs := false, 0, 0
	for _, name := range names {
		if seen[name] {
			return nil, ErrDrift
		}
		seen[name] = true
		if name == "writer.lock" {
			writer = true
			continue
		}
		if (strings.HasPrefix(name, ".ownership-") || strings.HasPrefix(name, ".job-")) && strings.HasSuffix(name, ".tmp") {
			prefix := ".ownership-"
			if strings.HasPrefix(name, ".job-") {
				prefix = ".job-"
			}
			nonce := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".tmp")
			if len(nonce) == 32 && strings.IndexFunc(nonce, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) < 0 {
				return nil, ErrStateUncertain
			}
		}
		entry := stateRecordName{name: name}
		switch {
		case strings.HasPrefix(name, "ownership-") && strings.HasSuffix(name, ".json"):
			entry.poolGUID = strings.TrimSuffix(strings.TrimPrefix(name, "ownership-"), ".json")
			if len(entry.poolGUID) > 20 || !validGUID(entry.poolGUID) {
				return nil, ErrStateRecord
			}
			owners++
		case strings.HasPrefix(name, "job-") && strings.HasSuffix(name, ".json"):
			entry.jobID = strings.TrimSuffix(strings.TrimPrefix(name, "job-"), ".json")
			if !validJobDigest(entry.jobID) {
				return nil, ErrStateRecord
			}
			jobs++
		default:
			return nil, ErrStateRecord
		}
		if owners > limits.ownership || jobs > limits.nonterminal+limits.terminal {
			return nil, ErrLimit
		}
		result = append(result, entry)
	}
	if !writer {
		return nil, ErrDrift
	}
	sort.Slice(result, func(i, j int) bool { return result[i].name < result[j].name })
	return result, nil
}

type stateCatalogBuilder struct {
	limits      stateCatalogLimits
	catalog     StateCatalog
	submissions map[string]bool
}

func newStateCatalogBuilder(limits stateCatalogLimits) *stateCatalogBuilder {
	return &stateCatalogBuilder{limits: limits, catalog: StateCatalog{Ownership: []OwnershipSummary{}, Jobs: []JobSummary{}}, submissions: make(map[string]bool)}
}

// Caller supplies a selector from stateCatalogNames and bounded private bytes.
// Failed additions leave the accumulated observation unchanged. The native
// inspector discards the entire builder on any error.
func (b *stateCatalogBuilder) add(entry stateRecordName, data []byte) error {
	size := uint64(len(data))
	if !b.limits.valid() || size > maxStateRecordBytes || b.catalog.Bytes > b.limits.bytes || size > b.limits.bytes-b.catalog.Bytes {
		return ErrLimit
	}
	if entry.poolGUID != "" {
		p, err := DecodeOwnershipRecord(data)
		if err != nil {
			return err
		}
		if p.PoolGUID != entry.poolGUID {
			return ErrDrift
		}
		if len(b.catalog.Ownership) >= b.limits.ownership {
			return ErrLimit
		}
		b.catalog.Ownership = append(b.catalog.Ownership, OwnershipSummary{PoolGUID: p.PoolGUID, Revision: p.Revision, Bytes: size})
	} else {
		j, err := decodeCatalogJob(data)
		if err != nil {
			return err
		}
		if j.ID != entry.jobID || b.submissions[j.Intent.SubmissionID] {
			return ErrDrift
		}
		terminal := j.Terminal()
		if terminal && b.catalog.Terminal >= b.limits.terminal || !terminal && b.catalog.Nonterminal >= b.limits.nonterminal {
			return ErrLimit
		}
		last := j.History[len(j.History)-1]
		b.catalog.Jobs = append(b.catalog.Jobs, JobSummary{ID: j.ID, SubmissionID: j.Intent.SubmissionID, Revision: last.Revision, State: last.State, Bytes: size})
		if terminal {
			b.catalog.Terminal++
		} else {
			b.catalog.Nonterminal++
		}
		b.submissions[j.Intent.SubmissionID] = true
	}
	b.catalog.Bytes += size
	return nil
}

// Each decoder independently enforces its exact complete shape/kind/checksum.
// Trying the two accepted formats cannot synthesize a missing review baseline.
// Only census consumes this projection; typed readers never discard a baseline.
func decodeCatalogJob(data []byte) (Job, error) {
	j, err := DecodeJobRecord(data)
	if err == nil {
		return j, nil
	}
	p, err := DecodePropertyJob(data)
	if err != nil {
		return Job{}, err
	}
	return p.Job(), nil
}
