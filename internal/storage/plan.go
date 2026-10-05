// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"
)

// Build creates a review only. Its inputs must be owned by the future helper
// adapter during this call. An eligible boolean is not evidence from a client.
func Build(at time.Time, session string, policy Policy, observation Observation, intent Intent) (Plan, error) {
	if at.IsZero() || !token(session) || !validIntent(intent) {
		return Plan{}, ErrInvalid
	}
	review, digest, err := prepare(policy, observation, intent)
	if err != nil {
		return Plan{}, err
	}
	var id [32]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Plan{}, ErrUnavailable
	}
	review.ID = hex.EncodeToString(id[:])
	review.CreatedAt = at
	review.ExpiresAt = at.Add(5 * time.Minute)
	// Include generated fields in the exact serialized review bound. This encoding
	// is an internal sizing check, not a persisted schema or transport contract.
	data, err := json.Marshal(review)
	if err != nil {
		return Plan{}, ErrInvalid
	}
	if len(data) > maxPlanBytes {
		return Plan{}, ErrLimit
	}
	return Plan{review: review, session: session, policyHash: digest}, nil
}

func (p Plan) Review() Review {
	r := p.review
	r.Context = append([]Filesystem(nil), r.Context...)
	r.Affected = append([]Resource(nil), r.Affected...)
	r.Versions = append([]DatasetIdentity(nil), r.Versions...)
	r.States = append([]FilesystemState(nil), r.States...)
	return r
}

// Validate must receive an independently fresh, complete observation, not the
// review's cache. It only checks preconditions; it cannot claim, reserve, submit
// or launch. Later durable admission must perform its own final native checks.
func (p Plan) Validate(at time.Time, session string, policy Policy, observation Observation) error {
	if p.review.ID == "" {
		return ErrInvalid
	}
	if at.Before(p.review.CreatedAt) || !at.Before(p.review.ExpiresAt) {
		return ErrExpired
	}
	if session != p.session {
		return ErrDenied
	}
	review, digest, err := prepare(policy, observation, p.review.Intent)
	if err != nil || digest != p.policyHash {
		return ErrDrift
	}
	expected := p.review
	expected.ID = ""
	expected.CreatedAt = time.Time{}
	expected.ExpiresAt = time.Time{}
	if !reflect.DeepEqual(review, expected) {
		return ErrDrift
	}
	return nil
}

func prepare(p Policy, o Observation, i Intent) (Review, [32]byte, error) {
	var zero [32]byte
	if err := preflight(p, o); err != nil {
		return Review{}, zero, err
	}
	files, err := indexObservation(o)
	if err != nil {
		return Review{}, zero, err
	}
	scopes, digest, err := indexPolicy(p, o, files)
	if err != nil {
		return Review{}, zero, err
	}
	target, ok := files[i.FilesystemGUID]
	if !ok || !permits(scopes[target.GUID], i.Property) {
		return Review{}, zero, ErrDenied
	}
	review := Review{PolicyRevision: p.Revision, PoolGUID: o.PoolGUID, PoolName: o.PoolName, Intent: i, Pool: o.Pool}
	ancestors := map[string]bool{}
	for parent := target.ParentGUID; parent != ""; parent = files[parent].ParentGUID {
		ancestors[parent] = true
	}
	// Reserve generous header/time/intent overhead before appending context. Every
	// selected row is counted before retaining it; impact is never silently paged.
	poolBytes, err := json.Marshal(o.Pool)
	if err != nil {
		return Review{}, zero, ErrInvalid
	}
	remaining := maxPlanBytes - 4096 - len(poolBytes)
	if remaining < 0 {
		return Review{}, zero, ErrLimit
	}
	context := make(map[string]bool)
	for _, f := range o.Filesystems {
		descendant := strings.HasPrefix(f.Name, target.Name+"/")
		if f.GUID != target.GUID && !ancestors[f.GUID] && !descendant {
			continue
		}
		affected := f.GUID == target.GUID
		if descendant {
			switch i.Property {
			case Quota:
				affected = true
			case Compression:
				affected = true
				for child := f; child.GUID != target.GUID; child = files[child.ParentGUID] {
					source := child.Properties.Compression.Source
					if source == Local || source == Received {
						affected = false
						break
					}
				}
			}
		}
		if affected && !permits(scopes[f.GUID], i.Property) {
			return Review{}, zero, ErrDenied
		}
		encoded, err := json.Marshal(f)
		if err != nil {
			return Review{}, zero, ErrInvalid
		}
		// Resource repeats name/GUID; conservatively count it even for unaffected rows.
		cost := len(encoded) + len(f.Name) + len(f.GUID) + 128
		if cost > remaining {
			return Review{}, zero, ErrLimit
		}
		remaining -= cost
		context[f.GUID] = true
		review.Context = append(review.Context, f)
		if affected {
			review.Affected = append(review.Affected, Resource{f.GUID, f.Name})
		}
	}
	for _, s := range o.States {
		if !context[s.GUID] {
			continue
		}
		encoded, err := json.Marshal(s)
		if err != nil {
			return Review{}, zero, ErrInvalid
		}
		cost := len(encoded) + 64
		if cost > remaining {
			return Review{}, zero, ErrLimit
		}
		remaining -= cost
		review.States = append(review.States, s)
	}
	sort.Slice(review.States, func(i, j int) bool { return review.States[i].Name < review.States[j].Name })
	for _, id := range o.Identities {
		if (id.Type != SnapshotType && id.Type != BookmarkType) || !context[id.OwnerGUID] {
			continue
		}
		encoded, err := json.Marshal(id)
		if err != nil {
			return Review{}, zero, ErrInvalid
		}
		cost := len(encoded) + 64
		if cost > remaining {
			return Review{}, zero, ErrLimit
		}
		remaining -= cost
		review.Versions = append(review.Versions, id)
	}
	sort.Slice(review.Versions, func(i, j int) bool { return review.Versions[i].Name < review.Versions[j].Name })
	sort.Slice(review.Context, func(i, j int) bool { return review.Context[i].Name < review.Context[j].Name })
	sort.Slice(review.Affected, func(i, j int) bool { return review.Affected[i].Name < review.Affected[j].Name })
	return review, digest, nil
}
