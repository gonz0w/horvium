// SPDX-License-Identifier: BSD-3-Clause
package storage

import "os"

// ReadJob reads one canonical ID-selected private record through the held
// directory. It returns lifecycle metadata, never an admission/recovery result
// or storage authority. Missing/unsafe/damaged files return no partial job.
func (s *StateDirectory) ReadJob(id string) (Job, error) {
	if s == nil || os.Geteuid() != 0 {
		return Job{}, ErrStateDirectory
	}
	if !validJobDigest(id) {
		return Job{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readJobLocked(id)
}

// Caller holds mu and has checked effective UID and the canonical selector.
func (s *StateDirectory) readJobLocked(id string) (Job, error) {
	data, err := s.readStateBytesLocked("job-" + id + ".json")
	if err != nil {
		return Job{}, err
	}
	j, err := DecodeJobRecord(data)
	if err != nil {
		return Job{}, err
	}
	if j.ID != id {
		return Job{}, ErrDrift
	}
	return j, nil
}
