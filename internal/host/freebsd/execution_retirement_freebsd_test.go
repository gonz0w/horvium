// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/gonz0w/horvium/internal/storage"
)

func nativeIdleRetirementFixture(t *testing.T, ctx context.Context, journalParent, ledgerParent string, check func(string, bool)) {
	t.Helper()
	j, l := filepath.Join(journalParent, "native-idle-retirement"), filepath.Join(ledgerParent, "native-idle-retirement")
	a, e := activateNativeFreshExecutionJournal(ctx, j, l)
	if e != nil {
		t.Fatal("owned idle activation", e)
	}
	defer a.Close()
	before, initial := readActivationFixture(t, j)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	check("retirement_cancelled_no_effect", a.FinishIdle(cancelled) != nil)
	after, _ := readActivationFixture(t, j)
	check("retirement_cancel_preserves_active", bytes.Equal(before, after))
	check("retirement_fixed_native_finish", a.FinishIdle(ctx) == nil)
	_, closed := readActivationFixture(t, j)
	check("retirement_exact_closed_history", closed.Version == initial.Version && closed.Generation == initial.Generation+1 && closed.Owner.State == "closed" && closed.Owner.ID == initial.Owner.ID && closed.Owner.Boot == initial.Owner.Boot && closed.Binding == initial.Binding && closed.Census == initial.Census && !closed.Metadata.Pending && !closed.Native.Pending && !closed.Disarmed)
	check("retirement_journal_lock_retained", nativeActivationLockHeld(t, j))
	check("retirement_ledger_lock_retained", nativeActivationLockHeld(t, l))
	got, e := a.Observe(ctx)
	check("retirement_observation_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
	check("retirement_repeat_denied", a.FinishIdle(ctx) != nil)
	bytesBefore, _ := readActivationFixture(t, j)
	check("retirement_release", a.Close() == nil && a.Close() == nil)
	check("retirement_both_locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
	q, e := activateNativeFreshExecutionJournal(ctx, j, l)
	after, _ = readActivationFixture(t, j)
	check("retirement_closed_no_recommission", q == nil && e != nil && bytes.Equal(bytesBefore, after))
}
