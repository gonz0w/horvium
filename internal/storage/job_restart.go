// SPDX-License-Identifier: BSD-3-Clause
package storage

import "sort"

const maxRestartPageJobs = 64

// JobRestartSummary separates the retained last durable metadata event from its
// conservative restart presentation. Neither state nor a stored observation
// digest establishes current native truth, executor quiescence or release.
// This projection contains no full review, session binding or executable intent.
type JobRestartSummary struct {
	ID, SubmissionID, PoolGUID string
	Last                       JobTransition
	State                      JobState
	Reason                     string
}

// RestartPage returns owned, ID-sorted restart summaries for all retained job
// kinds/states, including terminal holders. Queued work projects review-expired;
// applying/observing work projects executor-unverified. History and holds are
// unchanged. This is a read-only metadata projection, not a recovery coordinator,
// durable reconciliation, authority to replay, or validation of restored state.
//
// Use an empty cursor for the first page, then its last ID as an exclusive cursor
// on this same immutable view. A nonempty cursor must identify an existing job;
// it is not a transport token or snapshot identity for another collected view.
// Limits are 1..64. A complete empty/end page is nonnil; every error returns nil.
// BuildJobReservations supplies caller-owned bytes without native provenance;
// the native inspector's lease-bound view adds no executor/recovery authority.
func (v JobReservations) RestartPage(after string, limit int) ([]JobRestartSummary, error) {
	if !v.ready {
		return nil, ErrInvalid
	}
	if limit < 1 || limit > maxRestartPageJobs {
		return nil, ErrLimit
	}
	if after != "" {
		if !validJobDigest(after) {
			return nil, ErrInvalid
		}
		if _, exists := v.records[after]; !exists {
			return nil, ErrInvalid
		}
	}
	// The validated immutable view bounds this index to 64 nonterminal and
	// 1,000 terminal records. Do not copy full encoded records or baselines.
	ids := make([]string, 0, len(v.records))
	for id := range v.records {
		if id > after {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	ids = ids[:min(len(ids), limit)]
	result := make([]JobRestartSummary, 0, len(ids))
	for _, id := range ids {
		j := v.records[id].job
		state, reason, err := jobRestartProjection(j)
		if err != nil {
			return nil, err
		}
		result = append(result, JobRestartSummary{ID: j.ID, SubmissionID: j.Intent.SubmissionID,
			PoolGUID: j.Intent.PoolGUID, Last: j.History[len(j.History)-1], State: state, Reason: reason})
	}
	return result, nil
}
