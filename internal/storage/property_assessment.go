// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"reflect"
	"time"
)

// PropertyJobAssessment separates fresh observed matches from unchanged durable
// history and its conservative restart projection. Neither, either or both
// comparisons can match. Even a postcondition match does not attribute an
// executor or establish success, quiescence, restored-state validity or release.
type PropertyJobAssessment struct {
	Restart                               JobRestartSummary
	BaselineMatches, PostconditionMatches bool
}

// AssessObservation checks a retained job and both reviewed property projections
// using a complete, independently fresh graph and trusted policy. The caller
// owns inputs during this call and is responsible for native/root provenance;
// no data label or encoded digest supplies it. False eligibility is accepted
// without relaxing plan/admission gates. No clock/session/plan is restored, no
// history is appended and no native operation, publication or hold release occurs.
// Errors return zero; a valid differing review or policy digest matches neither.
func (p PropertyJob) AssessObservation(policy Policy, observation Observation) (PropertyJobAssessment, error) {
	normalized, err := normalizePropertyJob(p)
	if err != nil {
		return PropertyJobAssessment{}, err
	}
	if err = preflight(policy, observation); err != nil {
		return PropertyJobAssessment{}, err
	}
	files, err := indexCollectedObservation(observation)
	if err != nil {
		return PropertyJobAssessment{}, err
	}
	actual, digest, err := prepareIndexed(policy, observation, normalized.intent.review.Intent, files)
	if err != nil {
		return PropertyJobAssessment{}, err
	}
	j := normalized.job
	state, reason, err := jobRestartProjection(j)
	if err != nil {
		return PropertyJobAssessment{}, err
	}
	result := PropertyJobAssessment{Restart: JobRestartSummary{ID: j.ID, SubmissionID: j.Intent.SubmissionID,
		PoolGUID: j.Intent.PoolGUID, Last: j.History[len(j.History)-1], State: state, Reason: reason}}
	if digest != normalized.intent.policyDigest {
		return result, nil
	}
	baseline := normalized.intent.Review()
	baseline.ID = ""
	baseline.CreatedAt = time.Time{}
	baseline.ExpiresAt = time.Time{}
	result.BaselineMatches = reflect.DeepEqual(actual, baseline)
	result.PostconditionMatches = reflect.DeepEqual(actual, propertyPostconditionReview(normalized.intent))
	return result, nil
}
