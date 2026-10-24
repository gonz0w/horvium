// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type ownerEvents struct {
	sync.Mutex
	values []string
}

func (e *ownerEvents) add(s string) { e.Lock(); defer e.Unlock(); e.values = append(e.values, s) }
func (e *ownerEvents) snapshot() []string {
	e.Lock()
	defer e.Unlock()
	return append([]string(nil), e.values...)
}
func fakeOwnedCalls(events *ownerEvents) commandCalls {
	return commandCalls{
		observe: func(context.Context) error { events.add("observed-unreaped"); return nil },
		stop:    func() error { events.add("stop"); return nil },
		wait:    func() error { events.add("sole-wait"); return nil },
		proof:   func(error, time.Time) error { events.add("read-only-proof"); return nil },
	}
}
func awaitSlot(t *testing.T, slot *commandSlot, empty bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		slot.mu.Lock()
		ready := (slot.owner == nil) == empty
		slot.mu.Unlock()
		if ready {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("owner slot transition did not complete")
}
func TestExecutorOwnedFakeSignalsOnlyBeforeSoleWait(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { events.add("born-in-retained-slot"); return fakeOwnedCalls(events), nil })
	if err != nil {
		t.Fatal(err)
	}
	got := events.snapshot()
	want := []string{"born-in-retained-slot", "observed-unreaped", "stop", "sole-wait", "read-only-proof"}
	if len(got) != len(want) {
		t.Fatal(got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatal(got)
		}
	}
	awaitSlot(t, &slot, true)
}
func TestExecutorOwnedFakeUncertainDrainFencesNextProbeUntilActualProof(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	calls := fakeOwnedCalls(events)
	calls.wait = func() error { events.add("sole-wait"); <-release; return nil }
	err := ownedCommand(context.Background(), &slot, 20*time.Millisecond, func() (commandCalls, error) { return calls, nil })
	if !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	var births atomic.Int32
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { births.Add(1); return commandCalls{}, nil }); !errors.Is(err, ErrCleanupDebt) || births.Load() != 0 {
		t.Fatal("unknown drain admitted another birth", err)
	}
	unblock()
	awaitSlot(t, &slot, true)
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { births.Add(1); return fakeOwnedCalls(&ownerEvents{}), nil }); err != nil || births.Load() != 1 {
		t.Fatal("actual clean proof did not release slot", err)
	}
	got := events.snapshot()
	if len(got) != 4 || got[0] != "observed-unreaped" || got[1] != "stop" || got[2] != "sole-wait" || got[3] != "read-only-proof" {
		t.Fatal(got)
	}
}
func TestExecutorOwnedFakeObserverFailureCannotSkipRetirement(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	observation := errors.New("observer denied")
	calls := fakeOwnedCalls(events)
	calls.observe = func(context.Context) error { return observation }
	err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { return calls, nil })
	if !errors.Is(err, observation) {
		t.Fatal(err)
	}
	got := events.snapshot()
	if len(got) != 3 || got[0] != "stop" || got[1] != "sole-wait" || got[2] != "read-only-proof" {
		t.Fatal(got)
	}
	awaitSlot(t, &slot, true) // Error retained; actual independent retirement is known.
}
func TestExecutorOwnedFakeLostDrainProofIsStickyAndNeverSignalsAgain(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	calls := fakeOwnedCalls(events)
	calls.proof = func(error, time.Time) error { events.add("uncertain-drain"); return ErrCleanupDebt }
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { return calls, nil }); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	before := len(events.snapshot())
	births := 0
	for range 3 {
		if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { births++; return calls, nil }); !errors.Is(err, ErrCleanupDebt) {
			t.Fatal(err)
		}
	}
	if births != 0 || len(events.snapshot()) != before {
		t.Fatal("debt retried or signalled a reused identity")
	}
	slot.mu.Lock()
	retained := slot.owner != nil && slot.owner.start != nil
	slot.mu.Unlock()
	if !retained {
		t.Fatal("original owner discarded")
	}
}
func TestExecutorOwnedFakeBlockedCloserDoesNotSkipOtherClosersOrSoleWait(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	calls := fakeOwnedCalls(events)
	other := make(chan struct{})
	waited := make(chan struct{})
	calls.close = []func() error{func() error { <-release; return nil }, func() error { close(other); return nil }}
	calls.wait = func() error { close(waited); return nil }
	if err := ownedCommand(context.Background(), &slot, 20*time.Millisecond, func() (commandCalls, error) { return calls, nil }); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	for _, done := range []<-chan struct{}{other, waited} {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("blocked close suppressed another finalizer")
		}
	}
	unblock()
	awaitSlot(t, &slot, true)
}
func TestExecutorOwnedFakeBlockedStopCannotRaceWaitOrPermitAnotherProbe(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	calls := fakeOwnedCalls(events)
	var waited atomic.Bool
	calls.stop = func() error { <-release; events.add("stop-finished"); return nil }
	calls.wait = func() error { waited.Store(true); return nil }
	if err := ownedCommand(context.Background(), &slot, 20*time.Millisecond, func() (commandCalls, error) { return calls, nil }); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	if waited.Load() {
		t.Fatal("Wait consumed leader while positive operation was pending")
	}
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { t.Error("new birth while stop pending"); return commandCalls{}, nil }); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	unblock()
	awaitSlot(t, &slot, true)
	if !waited.Load() {
		t.Fatal("sole wait missing")
	}
}
func TestExecutorOwnedFakeFailedBirthClosesEveryPartialResource(t *testing.T) {
	var slot commandSlot
	var count atomic.Int32
	failure := os.ErrNotExist
	err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) {
		return commandCalls{close: []func() error{func() error { count.Add(1); return nil }, func() error { count.Add(1); return nil }}}, failure
	})
	if !errors.Is(err, failure) || count.Load() != 2 {
		t.Fatal(err, count.Load())
	}
	awaitSlot(t, &slot, true)
}
func TestExecutorOwnedFakeCollectionDebtPreventsLaterReadCalls(t *testing.T) {
	var calls atomic.Int32
	source := testSource(func(context.Context, probe, int) (readOutput, error) {
		calls.Add(1)
		return readOutput{}, errors.Join(ErrExit, ErrCleanupDebt)
	})
	snapshot := source.Collect(context.Background())
	if calls.Load() != 1 || snapshot.Complete() {
		t.Fatal("collection continued after debt", calls.Load())
	}
}
func TestExecutorOwnedFakeReserveNeverRenewsForQueuedCalls(t *testing.T) {
	var slot commandSlot
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	calls := fakeOwnedCalls(&ownerEvents{})
	calls.wait = func() error { <-release; return nil }
	started := time.Now()
	err := ownedCommand(context.Background(), &slot, 20*time.Millisecond, func() (commandCalls, error) { return calls, nil })
	if !errors.Is(err, ErrCleanupDebt) || time.Since(started) > time.Second {
		t.Fatal(err)
	}
	for range 3 {
		begin := time.Now()
		err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { t.Error("new start"); return commandCalls{}, nil })
		if !errors.Is(err, ErrCleanupDebt) || time.Since(begin) > 100*time.Millisecond {
			t.Fatal("pending fence refreshed cleanup", err)
		}
	}
	unblock()
	awaitSlot(t, &slot, true)
}

func TestExecutorOwnedFakeHungObserverCannotSuppressStopWaitOrFence(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	ctx, cancel := context.WithCancel(context.Background())
	calls := fakeOwnedCalls(events)
	calls.observe = func(context.Context) error { <-release; return context.Canceled }
	calls.stop = func() error { events.add("stop"); return nil }
	calls.wait = func() error { events.add("sole-wait"); return nil }
	cancel()
	// Begin a live work context, then cancel only after the born owner exists.
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	err := ownedCommand(ctx, &slot, 20*time.Millisecond, func() (commandCalls, error) { cancel(); return calls, nil })
	if !errors.Is(err, ErrCleanupDebt) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	got := events.snapshot()
	if len(got) != 2 || got[0] != "stop" || got[1] != "sole-wait" {
		t.Fatal(got)
	}
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { t.Error("hung observer permitted new probe"); return commandCalls{}, nil }); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	unblock()
	awaitSlot(t, &slot, true)
}
func TestExecutorOwnedFakeLateBirthRemainsOwnedAfterCallerReserve(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	birthRelease := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(birthRelease) }) }
	t.Cleanup(unblock)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := ownedCommand(ctx, &slot, 20*time.Millisecond, func() (commandCalls, error) { <-birthRelease; events.add("born"); return fakeOwnedCalls(events), nil })
	if !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) {
		t.Error("uncertain Start permitted another birth")
		return commandCalls{}, nil
	}); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	unblock()
	awaitSlot(t, &slot, true)
	got := events.snapshot()
	stop, wait := -1, -1
	for i, e := range got {
		if e == "stop" {
			stop = i
		}
		if e == "sole-wait" {
			wait = i
		}
	}
	if len(got) == 0 || got[0] != "born" || stop < 1 || wait <= stop {
		t.Fatal(got)
	}
}
func TestExecutorOwnedFakePrimaryStopAndWaitFailuresAllSurvive(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	observer := errors.New("observer denied")
	stop := errors.New("retirement failed")
	wait := errors.New("wait unknown")
	calls := fakeOwnedCalls(events)
	calls.observe = func(context.Context) error { return observer }
	calls.stop = func() error { return stop }
	calls.wait = func() error { return wait }
	calls.proof = func(error, time.Time) error { return ErrCleanupDebt }
	err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { return calls, nil })
	for _, want := range []error{observer, stop, wait, ErrCleanupDebt} {
		if !errors.Is(err, want) {
			t.Fatal("lost failure", want, err)
		}
	}
}

func TestExecutorOwnedFakeReaderFailureSurvivesOtherClosedReader(t *testing.T) {
	var slot commandSlot
	readerFailure := errors.New("capture failed independently")
	closed, failed := make(chan error, 1), make(chan error, 1)
	closed <- os.ErrClosed
	failed <- readerFailure
	calls := fakeOwnedCalls(&ownerEvents{})
	calls.readers = []<-chan error{closed, failed}
	err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { return calls, nil })
	if !errors.Is(err, readerFailure) {
		t.Fatal("closed reader hid independent capture failure", err)
	}
	awaitSlot(t, &slot, true)
}

func TestExecutorOwnedFakeForcedCloseCannotPublishIncompleteOutput(t *testing.T) {
	reader := make(chan error, 1)
	calls := fakeOwnedCalls(&ownerEvents{})
	calls.readers = []<-chan error{reader}
	calls.close = []func() error{func() error { reader <- os.ErrClosed; return nil }}
	observed := make(chan error, 1)
	var lost atomic.Int32
	deadline := time.Now().Add(20 * time.Millisecond)
	result := retireCommand(calls, observed, true, &lost, nil, deadline)
	if !errors.Is(result.err, errOutputIncomplete) || !errors.Is(result.err, context.DeadlineExceeded) {
		t.Fatal("forced close accepted partial output", result.err)
	}
	if !result.clean {
		t.Fatal("capture failure should not erase actual independent cleanup proof")
	}
}

func TestExecutorOwnedFakeLateProofCannotTurnDeadlineIntoSuccess(t *testing.T) {
	calls := fakeOwnedCalls(&ownerEvents{})
	var lost atomic.Int32
	result := retireCommand(calls, make(chan error), true, &lost, nil, time.Now().Add(-time.Millisecond))
	if !errors.Is(result.err, context.DeadlineExceeded) {
		t.Fatal("expired cleanup returned success", result.err)
	}
}

func TestExecutorOwnedFakeLostReservationForbidsSignalAndSecondWait(t *testing.T) {
	var slot commandSlot
	events := &ownerEvents{}
	calls := fakeOwnedCalls(events)
	calls.observe = func(context.Context) error { return errReservationLost }
	err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { return calls, nil })
	if !errors.Is(err, errReservationLost) || !errors.Is(err, ErrCleanupDebt) || len(events.snapshot()) != 0 {
		t.Fatal("lost reservation allowed stop/wait/proof", err, events.snapshot())
	}
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) {
		t.Error("lost reservation admitted another birth")
		return calls, nil
	}); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
}

func TestExecutorOwnedFakeLateObserverAfterOurWaitIsJoined(t *testing.T) {
	var slot commandSlot
	waited := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := fakeOwnedCalls(&ownerEvents{})
	calls.observe = func(context.Context) error { <-waited; return errReservationLost }
	calls.wait = func() error { close(waited); return nil }
	err := ownedCommand(ctx, &slot, time.Second, func() (commandCalls, error) { cancel(); return calls, nil })
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrCleanupDebt) {
		t.Fatal("our sole wait was mistaken for foreign consumption", err)
	}
	awaitSlot(t, &slot, true)
}

func TestExecutorOwnedFakeLossDuringAdmittedStopForbidsWait(t *testing.T) {
	var reservation atomic.Int32
	var stops, waits atomic.Int32
	calls := fakeOwnedCalls(&ownerEvents{})
	calls.stop = func() error {
		stops.Add(1)
		if !reservation.CompareAndSwap(reservationStopping, reservationLost) {
			t.Error("positive operation lacked atomic admission")
		}
		return nil
	}
	calls.wait = func() error { waits.Add(1); return nil }
	result := retireCommand(calls, make(chan error), true, &reservation, context.Canceled, time.Now().Add(time.Second))
	if !errors.Is(result.err, ErrCleanupDebt) || result.clean || stops.Load() != 1 || waits.Load() != 0 {
		t.Fatal("lost reservation permitted a later wait", result, stops.Load(), waits.Load())
	}
}

func TestExecutorOwnedFakeOuterDrainCannotCompeteWithPendingWait(t *testing.T) {
	var slot commandSlot
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	calls := fakeOwnedCalls(&ownerEvents{})
	calls.wait = func() error { <-release; return nil }
	if err := ownedCommand(context.Background(), &slot, 20*time.Millisecond, func() (commandCalls, error) { return calls, nil }); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	var drains atomic.Int32
	if err := slot.drainIdle(func() error { drains.Add(1); return nil }); !errors.Is(err, ErrCleanupDebt) || drains.Load() != 0 {
		t.Fatal("outer reaper competed with retained wait", err)
	}
	unblock()
	awaitSlot(t, &slot, true)
	if err := slot.drainIdle(func() error { drains.Add(1); return nil }); err != nil || drains.Load() != 1 {
		t.Fatal("actual completed joins did not permit outer drain", err)
	}
}

func TestExecutorOwnedFakeOuterDrainFencesAdmissionWithinCallerDeadline(t *testing.T) {
	var slot commandSlot
	release, started, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	go func() { done <- slot.drainIdle(func() error { close(started); <-release; return nil }) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("drain marker not established")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	begin := time.Now()
	err := ownedCommand(ctx, &slot, time.Second, func() (commandCalls, error) {
		t.Error("outer reaper drain admitted a new child")
		return commandCalls{}, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(begin) > time.Second {
		t.Fatal("drain mutex defeated caller deadline", err)
	}
	unblock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("outer drain did not finish")
	}
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) { return fakeOwnedCalls(&ownerEvents{}), nil }); err != nil {
		t.Fatal("drain marker was not released", err)
	}
}

func TestExecutorOwnedFakeGuardStatusOneIsAmbiguous(t *testing.T) {
	for _, tc := range []struct {
		exited bool
		code   int
		known  bool
	}{
		{true, 0, true}, {true, 1, false}, {true, 124, true}, {true, 126, true}, {true, 127, true},
		{false, -1, false}, {false, 0, false}, {true, -1, false}, {true, 256, false},
	} {
		if got := nativeGuardDrainStatus(tc.exited, tc.code); got != tc.known {
			t.Fatal(tc, got)
		}
	}
}

func TestExecutorOwnedFakeOuterDrainFailureKeepsAdmissionFenced(t *testing.T) {
	var slot commandSlot
	failure := errors.New("outer reaper status is unknown")
	if err := slot.drainIdle(func() error { return failure }); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if err := ownedCommand(context.Background(), &slot, time.Second, func() (commandCalls, error) {
		t.Error("failed outer drain admitted another child")
		return commandCalls{}, nil
	}); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
	if err := slot.drainIdle(func() error { t.Error("failed drain retried"); return nil }); !errors.Is(err, ErrCleanupDebt) {
		t.Fatal(err)
	}
}
