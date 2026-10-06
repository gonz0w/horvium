// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sort"
	"time"
)

// PropertyIntent is an immutable reviewed-property baseline, not a restored plan,
// admitted job, native eligibility proof or execution capability. Only validated
// capture/decoding can populate its private state; getters return owned copies.
type PropertyIntent struct {
	review                      Review
	capturedAt                  time.Time
	policyDigest, sessionDigest [32]byte
}

// CapturePropertyIntent revalidates the existing volatile plan/session against
// fresh trusted inputs before capturing its complete current review model. It
// claims no plan, writes no file and reserves no resource. Raw session credentials
// are not retained: the digest is scoped to this plan and cannot restore a session.
func (p Plan) CapturePropertyIntent(at time.Time, session string, policy Policy, observation Observation) (PropertyIntent, error) {
	if err := p.Validate(at, session, policy, observation); err != nil {
		return PropertyIntent{}, err
	}
	result := PropertyIntent{review: p.Review(), capturedAt: at.Round(0).UTC(), policyDigest: p.policyHash}
	result.review.CreatedAt = result.review.CreatedAt.Round(0).UTC()
	result.review.ExpiresAt = result.review.ExpiresAt.Round(0).UTC()
	result.sessionDigest = sha256.Sum256([]byte("horvium:property-intent-session:v1\n" + p.review.ID + "\n" + session))
	normalized, err := normalizePropertyIntent(result)
	if err != nil {
		return PropertyIntent{}, err
	}
	if _, err = encodePropertyIntent(normalized); err != nil {
		return PropertyIntent{}, err
	}
	return normalized, nil
}

func (i PropertyIntent) Review() Review { return (Plan{review: i.review}).Review() }

// CompareBaseline detects ordinary policy/review drift against a fresh trusted
// observation. It performs no authorization, session/expiry admission, quiescence,
// native postcondition verification or recovery; decoded labels cannot supply those.
func (i PropertyIntent) CompareBaseline(policy Policy, observation Observation) error {
	normalized, err := normalizePropertyIntent(i)
	if err != nil {
		return err
	}
	review, digest, err := prepare(policy, observation, normalized.review.Intent)
	if err != nil || digest != normalized.policyDigest {
		return ErrDrift
	}
	expected := normalized.review
	expected.ID = ""
	expected.CreatedAt = time.Time{}
	expected.ExpiresAt = time.Time{}
	if !reflect.DeepEqual(review, expected) {
		return ErrDrift
	}
	return nil
}

func canonicalIntentTime(at time.Time) bool {
	year := at.UTC().Year()
	return !at.IsZero() && year >= 1970 && year <= 9999 && at.After(time.Unix(0, 0))
}

func normalizePropertyIntent(i PropertyIntent) (PropertyIntent, error) {
	if err := propertyIntentBudget(i); err != nil {
		return PropertyIntent{}, err
	}
	r := i.Review()
	if !validJobDigest(r.ID) || r.PolicyRevision == 0 || !validIntent(r.Intent) || !canonicalIntentTime(r.CreatedAt) || !canonicalIntentTime(r.ExpiresAt) || !canonicalIntentTime(i.capturedAt) || r.ExpiresAt.Sub(r.CreatedAt) != 5*time.Minute || i.capturedAt.Before(r.CreatedAt) || !i.capturedAt.Before(r.ExpiresAt) {
		return PropertyIntent{}, ErrStateRecord
	}
	// Reconstruct only a consistency-check input. These synthetic completeness/
	// eligibility flags and derived affected scopes are never native/root authority.
	// Real capture/CompareBaseline require their own independently trusted inputs.
	o := Observation{Complete: true, Eligible: true, PoolGUID: r.PoolGUID, PoolName: r.PoolName, Filesystems: r.Context, States: r.States, Pool: r.Pool, Features: r.Features, Topology: r.Topology}
	files := make(map[string]Filesystem, len(r.Context))
	for _, f := range r.Context {
		files[f.GUID] = f
		o.Identities = append(o.Identities, DatasetIdentity{GUID: f.GUID, Name: f.Name, Type: FilesystemType})
	}
	o.Identities = append(o.Identities, r.Versions...)
	p := Policy{Revision: r.PolicyRevision, PoolGUID: r.PoolGUID, PoolName: r.PoolName}
	for _, a := range r.Affected {
		f, exists := files[a.GUID]
		if !exists || f.Name != a.Name {
			return PropertyIntent{}, ErrStateRecord
		}
		p.Scopes = append(p.Scopes, Scope{GUID: f.GUID, Name: f.Name, ParentGUID: f.ParentGUID, Properties: []Property{r.Intent.Property}})
	}
	expected, _, err := prepare(p, o, r.Intent)
	if err != nil {
		return PropertyIntent{}, ErrStateRecord
	}
	r.Features.Values = append([]PoolFeature(nil), r.Features.Values...)
	sort.Slice(r.Features.Values, func(a, b int) bool { return r.Features.Values[a].Name < r.Features.Values[b].Name })
	r.Topology = normalizedTopology(r.Topology)
	sort.Slice(r.Context, func(a, b int) bool { return r.Context[a].Name < r.Context[b].Name })
	sort.Slice(r.States, func(a, b int) bool { return r.States[a].Name < r.States[b].Name })
	sort.Slice(r.Affected, func(a, b int) bool { return r.Affected[a].Name < r.Affected[b].Name })
	sort.Slice(r.Versions, func(a, b int) bool { return r.Versions[a].Name < r.Versions[b].Name })
	if len(r.Versions) == 0 {
		r.Versions = nil
	}
	canonical := r
	canonical.ID = ""
	canonical.CreatedAt = time.Time{}
	canonical.ExpiresAt = time.Time{}
	if !reflect.DeepEqual(expected, canonical) {
		return PropertyIntent{}, ErrStateRecord
	}
	r.CreatedAt = r.CreatedAt.Round(0).UTC()
	r.ExpiresAt = r.ExpiresAt.Round(0).UTC()
	i.review = r
	i.capturedAt = i.capturedAt.Round(0).UTC()
	return i, nil
}

// Bound caller-owned rows/strings before allocation, copying, indexing or JSON.
// This conservative limit may refuse an input even when its JSON alone would fit.
func propertyIntentBudget(i PropertyIntent) error {
	r := i.review
	if len(r.Context) > maxRecords || len(r.States) > maxRecords || len(r.Versions) > maxRecords || len(r.Affected) > maxRecords || len(r.Features.Values) > maxFeatures || len(r.Topology.Nodes) > maxVdevs {
		return ErrLimit
	}
	remaining := maxStateRecordBytes - 1024
	debit := func(fields ...string) bool {
		remaining -= 128
		for _, s := range fields {
			if len(s) > maxFieldBytes {
				return false
			}
			remaining -= len(s)
		}
		return remaining >= 0
	}
	if !debit(r.ID, r.PoolGUID, r.PoolName, r.Intent.FilesystemGUID, string(r.Intent.Property), r.Intent.Value) || !debit(r.Pool.GUID, r.Pool.LoadGUID, r.Pool.Name, string(r.Pool.GUIDSource), string(r.Pool.LoadGUIDSource)) {
		return ErrLimit
	}
	for _, v := range poolValues(r.Pool) {
		if !debit(v.Effective, string(v.Source)) {
			return ErrLimit
		}
	}
	if !debit(r.Features.PoolGUID, r.Features.PoolName, r.Features.Format) || !debit(r.Topology.PoolGUID, r.Topology.PoolName, r.Topology.Health, r.Topology.Scan, r.Topology.Diagnostics) {
		return ErrLimit
	}
	for _, v := range r.Features.Values {
		if !debit(v.Name, v.State, string(v.Source)) {
			return ErrLimit
		}
	}
	for _, v := range r.Topology.Nodes {
		if !debit(v.GUID, v.ParentGUID, v.Address, v.Kind, v.Health, v.Annotation) {
			return ErrLimit
		}
	}
	value := func(v Value) bool { return debit(v.Effective, string(v.Source), v.SourceGUID, v.ReceivedValue) }
	for _, f := range r.Context {
		if !debit(f.GUID, f.Name, f.ParentGUID) || !value(f.Properties.Quota) || !value(f.Properties.RefQuota) || !value(f.Properties.Compression) {
			return ErrLimit
		}
	}
	for _, s := range r.States {
		if !debit(s.GUID, s.Name, s.EncryptionRootGUID) {
			return ErrLimit
		}
		for _, v := range stateValues(s) {
			if !value(v) {
				return ErrLimit
			}
		}
	}
	for _, v := range r.Versions {
		if !debit(v.GUID, v.Name, string(v.Type), v.OwnerGUID) {
			return ErrLimit
		}
	}
	for _, a := range r.Affected {
		if !debit(a.GUID, a.Name) {
			return ErrLimit
		}
	}
	return nil
}

func propertyDigestString(d [32]byte) string { return hex.EncodeToString(d[:]) }
