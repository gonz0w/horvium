// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"
)

// Test-only authority from a born command whose complete ancestry is unreaped.
// Metadata observers cannot construct it. The creator's stop-before-Wait gate
// and the controller's original ancestor Wait stay closed until seal returns.
// A seal serializes with an in-flight signal and permanently drops the callback.
type fixtureReservation struct {
	mu       sync.Mutex
	end      time.Time
	positive func(int) error
}

func (r *fixtureReservation) signal(signal int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.positive == nil || !time.Now().Before(r.end) || (signal != 9 && signal != 17 && signal != 19) {
		return errors.New("fixture signal authority unavailable")
	}
	return r.positive(signal)
}
func (r *fixtureReservation) seal() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.positive = nil
}

// On a broken handshake the native abandon callback exits the creator rather
// than releasing its consuming Wait. A returning fake callback still cannot
// invoke stop. The original controller retains ancestry until its own seal.
func fixturePublishBirth(response io.Writer, pid int, closeBirth func() error, abandon func()) error {
	fail := func() error { abandon(); return ErrCleanupDebt }
	if pid <= 1 {
		return fail()
	}
	var frame [8]byte
	binary.BigEndian.PutUint64(frame[:], uint64(pid))
	if n, e := response.Write(frame[:]); e != nil || n != len(frame) {
		return fail()
	}
	if closeBirth() != nil {
		return fail()
	}
	return nil
}
func fixtureStopAfterSeal(request io.Reader, stop func() error, abandon func()) error {
	fail := func() error { abandon(); return ErrCleanupDebt }
	if stop == nil {
		return fail()
	}
	var seal [1]byte
	if _, e := io.ReadFull(request, seal[:]); e != nil || seal[0] != 'S' {
		return fail()
	}
	return stop()
}
func fixtureBirthPID(frame [8]byte) uint64 { return binary.BigEndian.Uint64(frame[:]) }
