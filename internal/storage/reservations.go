// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"reflect"
	"sort"
)

// PoolReservation projects all retained job holders of one entire pool. Terminal
// labels do not establish native quiescence/postconditions and remain held here.
// Property changes need no new-device or new-namespace reservation. Other native
// operation kinds are not implemented by this model.
type PoolReservation struct {
	PoolGUID string
	Jobs     []JobSummary
}

type reservationRecord struct {
	job      Job
	property PropertyJob
}

// JobReservations is an immutable metadata view, not an acquired reservation,
// admission transaction, recovered scheduler or proof of a complete native store.
// It contains no release/claim/mutation method. Its zero value is unavailable.
type JobReservations struct {
	ready       bool
	limits      stateCatalogLimits
	bytes       uint64
	nonterminal int
	records     map[string]reservationRecord
	submissions map[string]string
	plans       map[string]string
	pools       []PoolReservation
}

// BuildJobReservations requires all job records in the caller's observed set.
// Both accepted formats are strictly decoded, including full property baselines.
// No filename, root lease, ownership record, live policy or native provenance is
// inferred from these bytes. A future native admission adapter must independently
// collect/bind the complete namespace under its writer lease and recheck it.
// Callers own the record slices throughout the call. Returned state owns copies.
func BuildJobReservations(records [][]byte) (JobReservations, error) {
	return buildJobReservations(records, defaultStateCatalogLimits)
}

func buildJobReservations(records [][]byte, limits stateCatalogLimits) (JobReservations, error) {
	if !limits.valid() || len(records) > limits.nonterminal+limits.terminal {
		return JobReservations{}, ErrLimit
	}
	var size uint64
	// Bound the entire input before decoding or allocating maps/copies. Count
	// actual record presentation bytes, not a smaller canonical re-encoding.
	for _, data := range records {
		if len(data) > maxStateRecordBytes || uint64(len(data)) > limits.bytes-size {
			return JobReservations{}, ErrLimit
		}
		size += uint64(len(data))
	}
	b, err := newReservationBuilder(limits)
	if err != nil {
		return JobReservations{}, err
	}
	for _, data := range records {
		if err := b.add(data); err != nil {
			return JobReservations{}, err
		}
	}
	return b.finish(), nil
}

// Stream native records without retaining a second whole set of encoded bytes.
// An addition commits only after all validation; finish transfers ownership and
// seals the builder before the immutable view can escape.
type reservationBuilder struct {
	view     JobReservations
	terminal int
	pools    map[string][]JobSummary
}

func newReservationBuilder(limits stateCatalogLimits) (*reservationBuilder, error) {
	if !limits.valid() {
		return nil, ErrLimit
	}
	return &reservationBuilder{view: JobReservations{ready: true, limits: limits,
		records:     make(map[string]reservationRecord),
		submissions: make(map[string]string), plans: make(map[string]string)},
		pools: make(map[string][]JobSummary)}, nil
}

func (b *reservationBuilder) add(data []byte) error {
	if b == nil || !b.view.ready {
		return ErrInvalid
	}
	if len(b.view.records) >= b.view.limits.nonterminal+b.view.limits.terminal || len(data) > maxStateRecordBytes || uint64(len(data)) > b.view.limits.bytes-b.view.bytes {
		return ErrLimit
	}
	j, err := DecodeJobRecord(data)
	record := reservationRecord{job: j}
	if err != nil {
		p, err := DecodePropertyJob(data)
		if err != nil {
			return err
		}
		record = reservationRecord{job: p.Job(), property: p}
		j = record.job
	}
	if _, exists := b.view.records[j.ID]; exists || b.view.submissions[j.Intent.SubmissionID] != "" || b.view.plans[j.Intent.PlanID] != "" {
		return ErrDrift
	}
	terminal := j.Terminal()
	if terminal && b.terminal >= b.view.limits.terminal || !terminal && b.view.nonterminal >= b.view.limits.nonterminal {
		return ErrLimit
	}
	b.view.bytes += uint64(len(data))
	b.view.records[j.ID] = record
	b.view.submissions[j.Intent.SubmissionID] = j.ID
	b.view.plans[j.Intent.PlanID] = j.ID
	if terminal {
		b.terminal++
	} else {
		b.view.nonterminal++
	}
	last := j.History[len(j.History)-1]
	b.pools[j.Intent.PoolGUID] = append(b.pools[j.Intent.PoolGUID], JobSummary{ID: j.ID, SubmissionID: j.Intent.SubmissionID, Revision: last.Revision, State: last.State, Bytes: uint64(len(data))})
	return nil
}

func (b *reservationBuilder) finish() JobReservations {
	if b == nil || !b.view.ready {
		return JobReservations{}
	}
	b.view.pools = make([]PoolReservation, 0, len(b.pools))
	for guid, jobs := range b.pools {
		sort.Slice(jobs, func(a, c int) bool { return jobs[a].ID < jobs[c].ID })
		b.view.pools = append(b.view.pools, PoolReservation{PoolGUID: guid, Jobs: jobs})
	}
	sort.Slice(b.view.pools, func(a, c int) bool { return b.view.pools[a].PoolGUID < b.view.pools[c].PoolGUID })
	view := b.view
	b.view = JobReservations{}
	b.pools = nil
	return view
}

// Pools returns independent, lexically sorted holder metadata. Multiple existing
// jobs can hold a pool: inspection preserves that conflict instead of dropping
// one. No state label or elapsed time releases a holder in this prerequisite.
func (v JobReservations) Pools() []PoolReservation {
	result := make([]PoolReservation, len(v.pools))
	for i, pool := range v.pools {
		result[i] = PoolReservation{PoolGUID: pool.PoolGUID, Jobs: append([]JobSummary(nil), pool.Jobs...)}
	}
	return result
}

// CheckPropertySubmission evaluates metadata only. A duplicate returns an owned
// copy of the existing complete property job/current history; a new candidate
// returns zero/false/nil. Every denial returns zero/false/error. Neither success
// acknowledges durability, acquires a reservation, claims a plan or authorizes
// launch. Repeated/concurrent checks against an empty view can all pass.
//
// Candidate must be a complete, initial queued property job. Retry identity is
// exact selection/review/policy/session binding, ignoring a new generated job ID
// and capture clock only. Legacy jobs cannot prove that binding. Duplicate checks
// precede capacity/held-pool denial; reusing a plan for a new submission is denied.
// Live clock/session/policy/eligibility, total ownership/job/transient capacity,
// recovery and atomic durable admission remain independent native gates.
func (v JobReservations) CheckPropertySubmission(candidate PropertyJob) (PropertyJob, bool, error) {
	if !v.ready {
		return PropertyJob{}, false, ErrInvalid
	}
	p, err := BindPropertyJob(candidate.job, candidate.intent)
	if err != nil {
		return PropertyJob{}, false, err
	}
	if len(p.job.History) != 1 {
		return PropertyJob{}, false, ErrStateRecord
	}
	selection := p.job.Intent
	if current, exists := v.records[p.job.ID]; exists && current.job.Intent.SubmissionID != selection.SubmissionID {
		return PropertyJob{}, false, ErrDenied
	}
	if id := v.submissions[selection.SubmissionID]; id != "" {
		current := v.records[id].property
		if current.job.Intent != selection || current.intent.policyDigest != p.intent.policyDigest || current.intent.sessionDigest != p.intent.sessionDigest || !reflect.DeepEqual(current.intent.review, p.intent.review) {
			return PropertyJob{}, false, ErrDenied
		}
		return PropertyJob{job: current.Job(), intent: current.Intent()}, true, nil
	}
	if v.plans[selection.PlanID] != "" {
		return PropertyJob{}, false, ErrDenied
	}
	for _, pool := range v.pools {
		if pool.PoolGUID == selection.PoolGUID {
			return PropertyJob{}, false, ErrDenied
		}
	}
	data, err := EncodePropertyJob(p)
	if err != nil {
		return PropertyJob{}, false, err
	}
	if v.nonterminal >= v.limits.nonterminal || uint64(len(data)) > v.limits.bytes-v.bytes {
		return PropertyJob{}, false, ErrLimit
	}
	return PropertyJob{}, false, nil
}
