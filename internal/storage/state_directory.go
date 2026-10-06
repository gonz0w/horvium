// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"os"
	"sync"
)

var (
	ErrStateDirectory = errors.New("storage state directory unavailable or unsafe")
	ErrStateLocked    = errors.New("storage state writer already active")
)

// StateDirectory holds a descriptor-anchored directory and its stable advisory
// writer lock. It is not a record store, enrollment, native executor or ZFS fence.
// Do not copy it; close the owned handle when its lifetime ends. No runtime wiring
// creates or opens state directories yet. Only the FreeBSD implementation opens.
type StateDirectory struct {
	mu                sync.Mutex
	directory, writer *os.File
}

// Close releases owned descriptors once, including for a zero handle. It never
// unlinks the lock file or directory; persistent lock identity survives restart.
func (s *StateDirectory) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	if s.writer != nil {
		if s.writer.Close() != nil {
			err = ErrStateDirectory
		}
		s.writer = nil
	}
	if s.directory != nil {
		if s.directory.Close() != nil {
			err = ErrStateDirectory
		}
		s.directory = nil
	}
	return err
}
