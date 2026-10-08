// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func boundExecution(t testing.TB) (*PropertyCoordinator, *handoffFixture, *time.Time, PropertyJob, *PropertyExecution, *PropertyExecutionAuthority) {
	t.Helper()
	c, b, now, j := handoffSetup(t, false)
	e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
	if err != nil || e == nil || len(b.authorities) != 1 || b.authorities[0] == nil {
		t.Fatal("original backend did not receive exactly one authority", err)
	}
	return c, b, now, j, e, b.authorities[0]
}

func TestPropertyExecutionAuthorityBinding(t *testing.T) {
	c, b, _, j := handoffSetup(t, false)
	b.bind = func(a *PropertyExecutionAuthority) error {
		if !c.mu.TryLock() {
			t.Fatal("binder called under coordinator mutex")
		}
		c.mu.Unlock()
		if err := a.CheckLive(context.Background()); !errors.Is(err, ErrDenied) {
			t.Fatal("pending binding can launch", err)
		}
		got, err := a.AdmittedJob()
		if err != nil || !reflect.DeepEqual(got, j) {
			t.Fatal("binding lost whole original admission", err)
		}
		got.job.History[0].Reason = "caller-edited"
		got.intent.review.Context[0].Name = "caller-edited"
		return nil
	}
	e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
	if err != nil || e == nil || len(b.authorities) != 1 {
		t.Fatal("binding not performed", err)
	}
	a := b.authorities[0]
	for range 3 {
		if err := a.CheckLive(context.Background()); err != nil {
			t.Fatal(err)
		}
		got, err := a.AdmittedJob()
		if err != nil || !reflect.DeepEqual(got, j) {
			t.Fatal("admission aliases caller", err)
		}
	}
	if b.publications != 1 || len(b.calls) != 0 {
		t.Fatal("binding performed execution I/O")
	}
	copy := *a
	if err := copy.CheckLive(context.Background()); err != nil {
		t.Fatal("handle copy lost shared identity", err)
	}
	if next, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID); next != nil || !errors.Is(err, ErrDenied) {
		t.Fatal("second binding", err)
	}
}

func TestPropertyExecutionAuthorityBindingCuts(t *testing.T) {
	for _, mode := range []string{"error", "expired", "rollback", "cancelled", "uncertainty", "removed", "changed-ack", "changed-plan", "changed-session", "replaced-entry"} {
		t.Run(mode, func(t *testing.T) {
			c, b, now, j := handoffSetup(t, false)
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			b.bind = func(a *PropertyExecutionAuthority) error {
				if !c.mu.TryLock() {
					t.Fatal("binding holds coordinator mutex")
				}
				defer c.mu.Unlock()
				entry := c.reviews[j.Intent().Review().ID]
				switch mode {
				case "error":
					return ErrUnavailable
				case "expired":
					*now = j.Intent().Review().ExpiresAt
				case "rollback":
					*now = now.Add(-time.Second)
				case "cancelled":
					stop()
				case "uncertainty":
					c.publicationBlocked = true
				case "removed":
					delete(c.reviews, j.Intent().Review().ID)
				case "changed-ack":
					entry.ack.job.History[0].Reason = "different"
				case "changed-plan":
					entry.plan.review.Context[0].Name = "different"
				case "changed-session":
					entry.plan.session = "session-b"
				case "replaced-entry":
					entry = propertyReviewEntry{plan: entry.plan, ack: entry.ack, submission: entry.submission, executionClaimed: true}
				}
				if mode != "removed" {
					c.reviews[j.Intent().Review().ID] = entry
				}
				return nil
			}
			if e, err := c.ClaimPropertyExecution(ctx, "session-a", j.Job().ID); e != nil || err == nil || len(b.authorities) != 1 {
				t.Fatal("binding cut granted worker", err)
			}
			if err := b.authorities[0].CheckLive(context.Background()); err == nil {
				t.Fatal("failed binding retained launch authority")
			}
			if e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID); e != nil || err == nil || len(b.authorities) != 1 {
				t.Fatal("binding failure retried", err)
			}
			if len(b.calls) != 0 || b.publications != 1 {
				t.Fatal("binding failure performed execution or admission I/O")
			}
		})
	}
}

func TestPropertyExecutionAuthorityLiveDenials(t *testing.T) {
	for _, mode := range []string{"nil", "zero", "nil-context", "cancelled", "uncertainty", "expired", "rollback", "clock-unavailable", "removed", "changed-ack", "changed-intent", "changed-plan", "changed-session", "duplicate", "unclaimed", "submission", "replaced-entry"} {
		t.Run(mode, func(t *testing.T) {
			c, b, now, j, e, a := boundExecution(t)
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			entry := c.reviews[j.Intent().Review().ID]
			switch mode {
			case "nil":
				a = nil
			case "zero":
				a = &PropertyExecutionAuthority{}
			case "nil-context":
				ctx = nil
			case "cancelled":
				stop()
			case "uncertainty":
				c.publicationBlocked = true
			case "expired":
				*now = j.Intent().Review().ExpiresAt
			case "rollback":
				*now = now.Add(-time.Second)
			case "clock-unavailable":
				c.clock = nil
			case "removed":
				delete(c.reviews, j.Intent().Review().ID)
			case "changed-ack":
				entry.ack.job.History[0].Reason = "different"
			case "changed-intent":
				entry.ack.intent.capturedAt = entry.ack.intent.capturedAt.Add(time.Millisecond)
			case "changed-plan":
				entry.plan.policyHash[0] ^= 1
			case "changed-session":
				entry.plan.session = "session-b"
			case "duplicate":
				entry.duplicateAdmission = true
			case "unclaimed":
				entry.executionClaimed = false
			case "submission":
				entry.submission = strings.Repeat("b", 64)
			case "replaced-entry":
				entry = propertyReviewEntry{plan: entry.plan, ack: entry.ack, submission: entry.submission, executionClaimed: true}
			}
			if mode != "removed" {
				c.reviews[j.Intent().Review().ID] = entry
			}
			if err := a.CheckLive(ctx); err == nil {
				t.Fatal("invalid original authority passed")
			}
			// An invalid caller handle/context does not revoke the real identity.
			if mode == "nil" || mode == "zero" || mode == "nil-context" || mode == "cancelled" {
				return
			}
			if _, err := e.Run(context.Background(), j.Job().ID); err == nil || executionCount(b.executionFixture, "apply") != 0 {
				t.Fatal("envelope bypassed original authority", err)
			}
		})
	}
}

func TestPropertyExecutionAuthorityFinalLiveChecks(t *testing.T) {
	for _, mode := range []string{"expired", "rollback", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			c, _, now, j, _, a := boundExecution(t)
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			reads := 0
			c.clock = func() time.Time {
				reads++
				if reads == 2 {
					switch mode {
					case "expired":
						return j.Intent().Review().ExpiresAt
					case "rollback":
						return now.Add(-time.Second)
					case "cancelled":
						stop()
					}
				}
				return *now
			}
			if err := a.CheckLive(ctx); err == nil || reads != 2 {
				t.Fatal("final clock/context skipped", err, reads)
			}
			if mode == "rollback" && !c.clockBlocked {
				t.Fatal("rollback did not latch original clock")
			}
		})
	}
}

func TestPropertyExecutionAuthorityRetirement(t *testing.T) {
	for _, mode := range []string{"success", "prepare", "applying", "refresh", "apply", "observe", "observing", "succeeded", "unquiet", "wrong-run-id", "clock"} {
		t.Run(mode, func(t *testing.T) {
			c, b, now, j, e, a := boundExecution(t)
			copy := *a
			id := j.Job().ID
			b.cut = mode
			if mode == "unquiet" {
				b.quiet = false
			}
			if mode == "wrong-run-id" {
				id = strings.Repeat("b", 64)
			}
			if mode == "clock" {
				*now = now.Add(-time.Second)
			}
			b.hook = func(stage string) {
				if stage == "prepare" || stage == "refresh" || stage == "apply" {
					if err := a.CheckLive(context.Background()); err != nil {
						t.Fatal("prelaunch authority lost", stage, err)
					}
				}
				if stage == "observe" || stage == "observing" || stage == "succeeded" {
					if err := copy.CheckLive(context.Background()); !errors.Is(err, ErrDenied) {
						t.Fatal("control return did not retire launch authority", stage, err)
					}
				}
			}
			got, err := e.Run(context.Background(), id)
			if mode == "success" && (err != nil || executionState(got) != JobSucceeded) {
				t.Fatal(err)
			}
			if err := copy.CheckLive(context.Background()); err == nil {
				t.Fatal("completed/failed attempt retained authority")
			}
			if _, err := e.Run(context.Background(), j.Job().ID); !errors.Is(err, ErrDenied) {
				t.Fatal("retired attempt replayed", err)
			}
			if ack, err := a.AdmittedJob(); err != nil || !reflect.DeepEqual(ack, j) {
				t.Fatal("retirement erased admission evidence", err)
			}
			if mode == "success" && c.publicationBlocked {
				t.Fatal("normal retirement fabricated uncertainty")
			}
		})
	}
}

func TestPropertyExecutionAuthorityLateOutcome(t *testing.T) {
	c, b, now, j, e, a := boundExecution(t)
	b.hook = func(stage string) {
		if stage == "apply" {
			if err := a.CheckLive(context.Background()); err != nil {
				t.Fatal(err)
			}
			*now = now.Add(5 * time.Minute)
			if _, err := c.Review(context.Background(), "session-b", j.Job().Intent.Change); err != nil {
				t.Fatal(err)
			}
		}
	}
	got, err := e.Run(context.Background(), j.Job().ID)
	if err != nil || executionState(got) != JobSucceeded || executionCount(b.executionFixture, "apply") != 1 {
		t.Fatal("late exact outcome lost", err)
	}
	if err := a.CheckLive(context.Background()); err == nil {
		t.Fatal("reclaimed plan retained launch authority")
	}
}

func TestPropertyExecutionAuthorityWorkerContext(t *testing.T) {
	c, b, _, j := handoffSetup(t, false)
	request, disconnect := context.WithCancel(context.Background())
	e, err := c.ClaimPropertyExecution(request, "session-a", j.Job().ID)
	if err != nil {
		t.Fatal(err)
	}
	disconnect()
	if len(b.authorities) != 1 {
		t.Fatal("authority not bound")
	}
	if err := b.authorities[0].CheckLive(context.Background()); err != nil {
		t.Fatal("request disconnect revoked independent worker", err)
	}
	got, err := e.Run(context.Background(), j.Job().ID)
	if err != nil || executionState(got) != JobSucceeded {
		t.Fatal("worker inherited request lifetime", err)
	}
}

func TestPropertyExecutionAuthorityConcurrentBinding(t *testing.T) {
	c, b, _, j := handoffSetup(t, false)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	b.bind = func(a *PropertyExecutionAuthority) error { close(entered); <-release; return nil }
	type reply struct {
		e   *PropertyExecution
		err error
	}
	result := make(chan reply, 1)
	go func() {
		e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID)
		result <- reply{e, err}
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("binding not entered")
	}
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			if e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID); e != nil || !errors.Is(err, ErrDenied) {
				t.Error("pending binding duplicated", err)
			}
			if err := b.authorities[0].CheckLive(context.Background()); !errors.Is(err, ErrDenied) {
				t.Error("pending authority usable", err)
			}
		})
	}
	wg.Wait()
	releaseOnce.Do(func() { close(release) })
	r := <-result
	if r.err != nil || r.e == nil || len(b.authorities) != 1 {
		t.Fatal("single binding failed", r.err)
	}
	for range 32 {
		wg.Go(func() {
			if err := b.authorities[0].CheckLive(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
}

func TestPropertyExecutionAuthorityRequiresBinder(t *testing.T) {
	c, b, _, j := handoffSetup(t, false)
	// Admission/execution alone cannot silently skip the authority bridge.
	c.backend = struct {
		PropertyBackend
		PropertyExecutionBackend
	}{b.coordinatorFixture, b.executionFixture}
	if e, err := c.ClaimPropertyExecution(context.Background(), "session-a", j.Job().ID); e != nil || !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing binder accepted", err)
	}
	if len(b.calls) != 0 || b.publications != 1 {
		t.Fatal("missing binder performed I/O")
	}
	for _, a := range []*PropertyExecutionAuthority{nil, {}} {
		if j, err := a.AdmittedJob(); err == nil || !reflect.DeepEqual(j, PropertyJob{}) {
			t.Fatal("empty handle reconstructed admission", err)
		}
	}
}
