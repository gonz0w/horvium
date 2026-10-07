// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"reflect"
	"time"
)

// ComparePostcondition compares one reviewed local property step with an
// independently fresh complete collected graph and trusted unchanged policy.
// A matching value is an observed fact, not attribution, native write eligibility,
// executor quiescence, recovery clearance or durable job success. In particular,
// an already-matching value or another root actor's identical change can match.
// No clock/session is restored and no record, hold, flag or native object changes.
func (i PropertyIntent) ComparePostcondition(policy Policy, observation Observation) error {
	normalized, err := normalizePropertyIntent(i)
	if err != nil {
		return err
	}
	if err = preflight(policy, observation); err != nil {
		return err
	}
	files, err := indexCollectedObservation(observation)
	if err != nil {
		return err
	}
	actual, digest, err := prepareIndexed(policy, observation, normalized.review.Intent, files)
	if err != nil || digest != normalized.policyDigest {
		return ErrDrift
	}
	expected := normalized.Review()
	change := expected.Intent
	affected := make(map[string]bool, len(expected.Affected))
	for _, row := range expected.Affected {
		affected[row.GUID] = true
	}
	for n := range expected.Context {
		f := &expected.Context[n]
		v := valueFor(f.Properties, change.Property)
		if f.GUID == change.FilesystemGUID {
			// A local set preserves any shadowed received column. Quota/refquota
			// affect descendant capacity, but never set descendant property rows.
			v.Effective, v.Source, v.SourceGUID = change.Value, Local, f.GUID
		} else if change.Property == Compression && affected[f.GUID] {
			v.Effective, v.Source, v.SourceGUID = change.Value, Inherited, change.FilesystemGUID
		} else {
			continue
		}
		switch change.Property {
		case Quota:
			f.Properties.Quota = v
		case RefQuota:
			f.Properties.RefQuota = v
		case Compression:
			f.Properties.Compression = v
		}
	}
	expected.ID = ""
	expected.CreatedAt = time.Time{}
	expected.ExpiresAt = time.Time{}
	if !reflect.DeepEqual(actual, expected) {
		return ErrDrift
	}
	return nil
}
