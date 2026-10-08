// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
)

// InspectPropertyJob reads one property job from a complete validated mixed
// census through this held root lease. The digest binds every ordered record
// name and actual presentation bytes, not just the selected job. It detects
// ordinary between-read drift; it supplies no authorization, atomic root fence,
// executor/quiescence/restoration proof, resolution or reservation release.
// Every failure returns zero job and digest. Read-only system UFS softupdates
// remains permitted; publication capability is a separate unchanged gate.
func (s *StateDirectory) InspectPropertyJob(id string) (PropertyJob, [32]byte, error) {
	if s == nil || os.Geteuid() != 0 {
		return PropertyJob{}, [32]byte{}, ErrStateDirectory
	}
	if err := ValidatePropertyJobSelection(id); err != nil {
		return PropertyJob{}, [32]byte{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inspectPropertyJobLocked(id, nil)
}

// Caller validated kernel UID/selector and holds s.mu. An original claim may
// add its exact volatile lineage/hold check to the same complete census. The
// retained diagnostic uses no claim and preserves its read-only semantics.
func (s *StateDirectory) inspectPropertyJobLocked(id string, check func(JobReservations, PropertyJob) error) (PropertyJob, [32]byte, error) {
	b, err := newReservationBuilder(defaultStateCatalogLimits)
	if err != nil {
		return PropertyJob{}, [32]byte{}, err
	}
	h := sha256.New()
	h.Write([]byte("horvium:property-assessment-census:v1\n"))
	catalog, err := s.inspectStateRecordsLocked(defaultStateCatalogLimits, nil, "", func(entry stateRecordName, data []byte) error {
		if entry.jobID != "" {
			if err := b.add(data); err != nil {
				return err
			}
		}
		// Length framing prevents ambiguous concatenations. The existing census
		// sorts all names, bounds bytes, validates each object/codec and repeats
		// final namespace/object checks before anything can escape.
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(entry.name)))
		h.Write(size[:])
		h.Write([]byte(entry.name))
		binary.BigEndian.PutUint64(size[:], uint64(len(data)))
		h.Write(size[:])
		h.Write(data)
		return nil
	})
	if err != nil {
		return PropertyJob{}, [32]byte{}, err
	}
	view := b.finish()
	current, exists := view.records[id]
	if !exists {
		return PropertyJob{}, [32]byte{}, ErrDenied
	}
	if current.property.job.ID == "" {
		return PropertyJob{}, [32]byte{}, ErrStateRecord
	}
	owned := false
	for _, p := range catalog.Ownership {
		owned = owned || p.PoolGUID == current.job.Intent.PoolGUID
	}
	if !owned {
		return PropertyJob{}, [32]byte{}, ErrDenied
	}
	if check != nil {
		if err := check(view, current.property); err != nil {
			return PropertyJob{}, [32]byte{}, err
		}
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return PropertyJob{job: current.property.Job(), intent: current.property.Intent()}, digest, nil
}
