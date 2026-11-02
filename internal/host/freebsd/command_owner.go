// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var ErrCleanupDebt = errors.New("native probe retirement is uncertain")

var errReservationLost = errors.New("original child reservation was lost")
var errOutputIncomplete = errors.New("command output did not drain before cleanup deadline")

const (
	reservationHeld int32 = iota
	reservationLost
	reservationStopping
	reservationWaiting
)

const commandCleanupReserve = 250 * time.Millisecond

// The slot is private to this process's fixed-command executor. It is neither a
// durable native-write permit nor a replacement for the M2 execution journal.
// Keep the actual owner, including its original handles, until retirement is
// known. A timeout never resets the slot or authorizes another probe.
type commandSlot struct {
	mu        sync.Mutex
	owner     *commandOwner
	draining  bool
	drainDebt bool
	changed   chan struct{}
}

type commandOwner struct {
	uncertain bool // guarded by the slot mutex
	done      chan commandResult
	retiring  chan struct{}
	retire    sync.Once
	deadline  time.Time
	start     func() (commandCalls, error) // retains the original command/handles
}

type commandResult struct {
	err   error
	clean bool
}

type commandCalls struct {
	observe func(context.Context) error // non-consuming exit observation only
	stop    func() error                // final positive operation, before Wait
	wait    func() error                // exactly one actual wait
	close   []func() error
	readers []<-chan error
	proof   func(error, time.Time) error // read-only, after Wait
}

func (s *commandSlot) wakeLocked() {
	if s.changed != nil {
		close(s.changed)
	}
	s.changed = make(chan struct{})
}

func (s *commandSlot) acquire(ctx context.Context) (*commandOwner, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		s.mu.Lock()
		if s.owner == nil && !s.draining && !s.drainDebt {
			o := &commandOwner{done: make(chan commandResult, 1), retiring: make(chan struct{})}
			s.owner = o
			s.wakeLocked()
			s.mu.Unlock()
			return o, nil
		}
		if s.drainDebt || (s.owner != nil && s.owner.uncertain) {
			s.mu.Unlock()
			return nil, ErrCleanupDebt
		}
		changed := s.changed
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

// The isolated native supervisor may reap adopted children only after the
// executor has joined its original child and handles. Mark admission under the
// mutex, then do OS work outside it so queued callers keep their own deadlines.
func (s *commandSlot) drainIdle(drain func() error) (result error) {
	s.mu.Lock()
	if s.owner != nil || s.draining || s.drainDebt {
		s.mu.Unlock()
		return ErrCleanupDebt
	}
	s.draining = true
	s.wakeLocked()
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.draining = false
		s.drainDebt = result != nil
		s.wakeLocked()
		s.mu.Unlock()
	}()
	result = drain()
	return result
}

func (s *commandSlot) uncertain(o *commandOwner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner == o && !o.uncertain {
		o.uncertain = true
		s.wakeLocked()
	}
}

func (s *commandSlot) finish(o *commandOwner, result commandResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if result.clean {
		s.owner = nil
	} else {
		o.uncertain = true
	}
	s.wakeLocked()
	o.done <- result
}

func ignoredClose(err error) error {
	if errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
}

// All finalizers are started before joining any of them. In particular an
// observer/closer cannot suppress stop or Wait. Stop and Wait share one goroutine
// so a blocked positive operation cannot race a reap; the slot retains it.
func retireCommand(calls commandCalls, observed <-chan error, observedJoined bool, reservation *atomic.Int32, primary error, deadline time.Time) commandResult {
	readDone := make(chan struct{})
	readResult := make(chan error, 1)
	go func() {
		var err error
		for _, done := range calls.readers {
			err = errors.Join(err, ignoredClose(<-done))
		}
		readResult <- err
		close(readDone)
	}()
	drain := func(end time.Time) bool {
		select {
		case <-readDone:
			return true
		default:
		}
		timer := time.NewTimer(max(0, time.Until(end)))
		defer timer.Stop()
		select {
		case <-readDone:
			return true
		case <-timer.C:
			return false
		}
	}
	closed := make([]<-chan error, 0, len(calls.close))
	for _, closePipe := range calls.close {
		done := make(chan error, 1)
		go func(closePipe func() error) { drain(deadline); done <- ignoredClose(closePipe()) }(closePipe)
		closed = append(closed, done)
	}
	type waited struct {
		stop, wait, capture error
		lost                bool
	}
	joined := make(chan waited, 1)
	go func() {
		var capture error
		if primary == nil && reservation.Load() != reservationLost {
			// Natural root exit must not publish output whose writer survives it.
			// Share the original reserve, leaving its latter half for stop/join.
			if !drain(time.Now().Add(max(0, time.Until(deadline)/2))) {
				capture = errOutputIncomplete
			}
		}
		if !reservation.CompareAndSwap(reservationHeld, reservationStopping) {
			joined <- waited{capture: capture, lost: true}
			return
		}
		stopErr := calls.stop()
		if errors.Is(stopErr, os.ErrProcessDone) {
			stopErr = nil
		}
		if !drain(deadline) {
			capture = errors.Join(capture, errOutputIncomplete)
		}
		if !reservation.CompareAndSwap(reservationStopping, reservationWaiting) {
			joined <- waited{stop: stopErr, capture: capture, lost: true}
			return
		}
		joined <- waited{stop: stopErr, wait: calls.wait(), capture: capture}
	}()
	// This owner may outlive its caller's finite cleanup reserve. It performs no
	// further positive operation, never renews work and retains every pending join.
	result := <-joined
	cleanup := result.stop
	for _, done := range closed {
		cleanup = errors.Join(cleanup, <-done)
	}
	primary = errors.Join(primary, result.capture, <-readResult)
	if !observedJoined {
		// Cancellation of this read-only observer must be joined too. It never
		// grants a new signal capability after Wait has consumed the leader.
		observation := <-observed
		if observation != nil && !errors.Is(observation, context.Canceled) && !errors.Is(observation, context.DeadlineExceeded) {
			primary = errors.Join(primary, observation)
		}
	}
	var proof error
	if result.lost || reservation.Load() == reservationLost {
		proof = errors.Join(errReservationLost, ErrCleanupDebt)
	} else {
		proof = calls.proof(result.wait, deadline)
	}
	clean := cleanup == nil && proof == nil && !errors.Is(primary, ErrCleanupDebt)
	err := errors.Join(primary, result.wait, cleanup, proof)
	if !time.Now().Before(deadline) {
		// Late actual proof can release admission, but cannot convert this
		// expired invocation into successful output publication.
		err = errors.Join(err, context.DeadlineExceeded)
	}
	if !clean {
		err = errors.Join(err, ErrCleanupDebt)
	}
	return commandResult{err: err, clean: clean}
}

// start registers its born handles in this already-retained owner before any
// fallible post-birth work. It must return finalizers even on partial setup error.
func ownedCommand(ctx context.Context, slot *commandSlot, reserve time.Duration, start func() (commandCalls, error)) error {
	owner, err := slot.acquire(ctx)
	if err != nil {
		return err
	}
	owner.start = start
	beginRetirement := func() time.Time {
		owner.retire.Do(func() {
			owner.deadline = time.Now().Add(reserve)
			if end, ok := ctx.Deadline(); ok && end.Add(reserve).Before(owner.deadline) {
				owner.deadline = end.Add(reserve)
			}
			close(owner.retiring)
		})
		return owner.deadline
	}
	go func() {
		calls, startErr := owner.start()
		if calls.wait == nil { // No child was born; still join every opened pipe.
			cleanup := error(nil)
			closed := make([]<-chan error, 0, len(calls.close))
			for _, closePipe := range calls.close {
				done := make(chan error, 1)
				go func(closePipe func() error) { done <- ignoredClose(closePipe()) }(closePipe)
				closed = append(closed, done)
			}
			for _, done := range closed {
				cleanup = errors.Join(cleanup, <-done)
			}
			result := commandResult{err: errors.Join(startErr, cleanup), clean: cleanup == nil}
			if !result.clean {
				result.err = errors.Join(result.err, ErrCleanupDebt)
			}
			slot.finish(owner, result)
			return
		}
		watchCtx, stopWatch := context.WithCancel(context.Background())
		observed := make(chan error, 1)
		var reservation atomic.Int32
		go func() {
			err := calls.observe(watchCtx)
			if errors.Is(err, errReservationLost) {
				for {
					state := reservation.Load()
					if state == reservationWaiting {
						// Our one Wait has claimed consumption after cancellation;
						// a late WNOWAIT observer may now see its expected ECHILD.
						err = context.Canceled
						break
					}
					if state == reservationLost || reservation.CompareAndSwap(state, reservationLost) {
						break
					}
				}
			}
			observed <- err
		}()
		primary, observedJoined := startErr, false
		if primary == nil {
			select {
			case err := <-observed:
				primary, observedJoined = err, true
			case <-ctx.Done():
				primary = ctx.Err()
			}
		}
		deadline := beginRetirement()
		stopWatch()
		slot.finish(owner, retireCommand(calls, observed, observedJoined, &reservation, primary, deadline))
	}()
	select {
	case result := <-owner.done:
		return result.err
	case <-ctx.Done():
	case <-owner.retiring:
	}
	// One reserve, never refreshed by failed finalizers, observer errors or a
	// second caller. Late completion can release the fence only with actual proof.
	timer := time.NewTimer(max(0, time.Until(beginRetirement())))
	defer timer.Stop()
	select {
	case result := <-owner.done:
		return result.err
	case <-timer.C:
		slot.uncertain(owner)
		return errors.Join(ctx.Err(), ErrCleanupDebt)
	}
}
