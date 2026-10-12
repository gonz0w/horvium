// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"testing"
	"time"
)

func TestPolicyEnvelopeAbsoluteCaller(t *testing.T) {
	start := time.Now()
	c, stop := newPolicyEnvelopeCaller(start)
	defer stop()
	if !policyEnvelopeCallerUnchanged(c, start) || c.Err() != nil {
		t.Fatal("one absolute 180-second caller")
	}
	renewed, cancel := newPolicyEnvelopeCaller(start.Add(time.Second))
	defer cancel()
	if policyEnvelopeCallerUnchanged(renewed, start) {
		t.Fatal("renewed caller must be rejected")
	}
	old, cancelOld := context.WithDeadline(context.Background(), start.Add(120*time.Second))
	defer cancelOld()
	if policyEnvelopeCallerUnchanged(old, start) {
		t.Fatal("old 120-second profile cannot satisfy new caller")
	}
}

func TestPolicyEnvelopeParentClamp(t *testing.T) {
	// Synthetic phase entry after 40 seconds; no slow sleep or native claim.
	start := time.Now().Add(-40 * time.Second)
	old, cancelOld := context.WithDeadline(context.Background(), start.Add(120*time.Second))
	defer cancelOld()
	oldActivation, stopOld := context.WithTimeout(old, 90*time.Second)
	defer stopOld()
	oldEnd, _ := old.Deadline()
	clipped, _ := oldActivation.Deadline()
	if !clipped.Equal(oldEnd) {
		t.Fatal("earlier parent must clamp activation")
	}
	extended, cancel := newPolicyEnvelopeCaller(start)
	defer cancel()
	activation, stop := context.WithTimeout(extended, 90*time.Second)
	defer stop()
	callerEnd, _ := extended.Deadline()
	ownEnd, _ := activation.Deadline()
	if !ownEnd.Before(callerEnd) {
		t.Fatal("90-second product deadline remains independent")
	}
	if !policyEnvelopeCallerUnchanged(extended, start) {
		t.Fatal("activation renewed original caller")
	}
}

// Fixture-only absolute context. Original start is retained for drift checks.
func newPolicyEnvelopeCaller(start time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), start.Add(180*time.Second))
}
func policyEnvelopeCallerUnchanged(c context.Context, start time.Time) bool {
	d, ok := c.Deadline()
	return ok && d.Equal(start.Add(180*time.Second))
}
