// SPDX-License-Identifier: BSD-3-Clause
package storage

import "os"

// ReadPropertyJob preserves the full baseline through the held private directory.
// No conversion from legacy lifecycle-only records or authority is supplied.
func (s *StateDirectory) ReadPropertyJob(id string) (PropertyJob, error) {
	if s == nil || os.Geteuid() != 0 {
		return PropertyJob{}, ErrStateDirectory
	}
	if !validJobDigest(id) {
		return PropertyJob{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readPropertyJobLocked(id)
}

func (s *StateDirectory) readPropertyJobLocked(id string) (PropertyJob, error) {
	data, err := s.readStateBytesLocked("job-" + id + ".json")
	if err != nil {
		return PropertyJob{}, err
	}
	p, err := DecodePropertyJob(data)
	if err != nil {
		return PropertyJob{}, err
	}
	if p.job.ID != id {
		return PropertyJob{}, ErrDrift
	}
	return p, nil
}
