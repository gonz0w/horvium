// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash"
	"os"
)

// PublishApplyingPropertyJob durably acknowledges only this original claim's
// exact queued-to-applying successor. It owns no worker and invokes no native
// storage command. Metadata is not launch authority: the complete original
// backend must independently prove session/clock, eligibility, executor/recovery
// and global/pool serialization. Copies share one consumed publication attempt.
// Errors return zero, including a visible but unacknowledged applying record.
func (c *PropertyStateClaim) PublishApplyingPropertyJob(next PropertyJob) (PropertyJob, error) {
	return c.publishApplyingPropertyJobBounded(next, defaultStateCatalogLimits, nil)
}

// Reduced bounds and phase wrapping are private disposable fixture seams.
func (c *PropertyStateClaim) publishApplyingPropertyJobBounded(next PropertyJob, limits stateCatalogLimits, wrap func(ownershipPublication) ownershipPublication) (PropertyJob, error) {
	return c.publishClaimPropertyJobBounded(next, limits, wrap, true)
}

// The two original-claim publishers share the same strict transaction. Only
// validated metadata edges and the private post-sync acknowledgement differ.
func (c *PropertyStateClaim) publishClaimPropertyJobBounded(next PropertyJob, limits stateCatalogLimits, wrap func(ownershipPublication) ownershipPublication, applying bool) (PropertyJob, error) {
	zero := func(err error) (PropertyJob, error) { return PropertyJob{}, err }
	if os.Geteuid() != 0 {
		return zero(ErrStateDirectory)
	}
	next, err := BindPropertyJob(next.job, next.intent)
	if err != nil {
		return zero(err)
	}
	data, err := EncodePropertyJob(next)
	if err != nil {
		return zero(err)
	}
	if !limits.valid() {
		return zero(ErrLimit)
	}
	if c == nil || c.state == nil || c.identity == nil || next.job.ID != c.identity.id {
		return zero(ErrDenied)
	}
	s := c.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.propertyOrigin != s || s.propertyAdmissions.active != c.identity {
		return zero(ErrDenied)
	}
	if s.publicationBlocked {
		return zero(ErrStateUncertain)
	}
	if applying && c.identity.applyingAttempted || !applying && c.identity.successorAttempted {
		return zero(ErrDenied)
	}
	if _, err = s.ownershipDirectoryLocked(); err != nil {
		return zero(err)
	}
	var baseline, expectedAfter [32]byte
	var predecessor PropertyJob
	// Reuse the strict bounded namespace scanner and publisher, preserving all
	// actual presentation bytes outside the exact owned destination. Only the
	// transaction's verified temporary may be excluded during staging.
	inspect := func(temporary string) (StateCatalog, error) {
		if s.propertyOrigin != s || s.propertyAdmissions.active != c.identity {
			return StateCatalog{}, ErrDenied
		}
		b, e := newReservationBuilder(limits)
		if e != nil {
			return StateCatalog{}, e
		}
		before, after := sha256.New(), sha256.New()
		for _, h := range []hash.Hash{before, after} {
			h.Write([]byte("horvium:property-assessment-census:v1\n"))
		}
		frame := func(h hash.Hash, name string, content []byte) {
			var size [8]byte
			binary.BigEndian.PutUint64(size[:], uint64(len(name)))
			h.Write(size[:])
			h.Write([]byte(name))
			binary.BigEndian.PutUint64(size[:], uint64(len(content)))
			h.Write(size[:])
			h.Write(content)
		}
		catalog, e := s.inspectStateRecordsLocked(limits, nil, temporary, func(entry stateRecordName, content []byte) error {
			if entry.jobID != "" {
				if e := b.add(content); e != nil {
					return e
				}
			}
			frame(before, entry.name, content)
			if entry.jobID == next.job.ID {
				content = data
			}
			frame(after, entry.name, content)
			return nil
		})
		if e != nil {
			return StateCatalog{}, e
		}
		view := b.finish()
		r, exists := view.records[next.job.ID]
		if !exists || r.property.job.ID == "" {
			return StateCatalog{}, ErrDenied
		}
		owned := false
		for _, p := range catalog.Ownership {
			owned = owned || p.PoolGUID == r.job.Intent.PoolGUID
		}
		if !owned {
			return StateCatalog{}, ErrDenied
		}
		var validation error
		if applying {
			validation = s.propertyAdmissions.validateApplying(c.identity, view, r.property, next)
		} else {
			validation = s.propertyAdmissions.validateSuccessor(c.identity, view, r.property, next)
		}
		if validation != nil {
			return StateCatalog{}, validation
		}
		var digest, projected [32]byte
		copy(digest[:], before.Sum(nil))
		copy(projected[:], after.Sum(nil))
		if baseline == ([32]byte{}) {
			baseline, expectedAfter, predecessor = digest, projected, clonePropertyJob(r.property)
		}
		if baseline != digest || expectedAfter != projected {
			return StateCatalog{}, ErrDrift
		}
		return catalog, nil
	}
	if _, err = inspect(""); err != nil {
		if errors.Is(err, ErrStateUncertain) {
			s.publicationBlocked = true
		}
		return zero(err)
	}
	// Consume before staging, under the original mutex. A failed attempt cannot
	// be retried through a copied claim, even if owned cleanup succeeds.
	if applying {
		c.identity.applyingAttempted = true
	} else {
		c.identity.successorAttempted = true
	}
	revision := uint64(len(predecessor.job.History))
	err = s.publishStateInspectedLocked("job-"+next.job.ID+".json", ".job-", data, revision,
		func(catalog StateCatalog) error {
			return allowJobPublication(catalog, next.job, revision, uint64(len(data)), limits)
		},
		func() error {
			current, e := s.readPropertyJobLocked(next.job.ID)
			if e != nil {
				return e
			}
			actual, e := EncodePropertyJob(current)
			if e != nil {
				return e
			}
			expected, e := EncodePropertyJob(predecessor)
			if e != nil || !bytes.Equal(actual, expected) {
				return ErrDrift
			}
			return ValidatePropertyJobSuccessor(current, next)
		}, inspect, wrap)
	if err != nil {
		return zero(err)
	}
	var ack PropertyJob
	var digest [32]byte
	if applying {
		ack, digest, err = c.inspectPropertyJobLocked()
	} else {
		ack, digest, err = s.inspectPropertyJobLocked(c.identity.id, func(v JobReservations, current PropertyJob) error {
			return s.propertyAdmissions.checkAcknowledged(c.identity, v, current, sha256.Sum256(data), revision+1)
		})
	}
	actual, encodingErr := EncodePropertyJob(ack)
	if err != nil || encodingErr != nil || digest != expectedAfter || !bytes.Equal(actual, data) {
		s.publicationBlocked = true
		return zero(ErrStateUncertain)
	}
	// Populate lineage ONLY after exact post-sync record and whole-census readback.
	// This private receipt never grants native outcome attribution or release.
	c.identity.acknowledgedReceipt = sha256.Sum256(data)
	c.identity.acknowledgedRevision = revision + 1
	c.identity.successorAttempted = false
	return ack, nil
}
