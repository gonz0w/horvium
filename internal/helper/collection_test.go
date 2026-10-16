// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// A deadline and its socket timer can expire before the cancellation callback
// publishes Err. Make that ordering deterministic without sleeping or retries.
type unpublishedDeadline struct {
	context.Context
	deadline time.Time
}

func (c unpublishedDeadline) Deadline() (time.Time, bool) { return c.deadline, true }

func TestCollectionExpiredSocketDoesNotDependOnContextCallback(t *testing.T) {
	client, _ := connected(t)
	deadline := time.Now().Add(-time.Second)
	ctx := unpublishedDeadline{context.Background(), deadline}
	if ctx.Err() != nil || client.SetReadDeadline(deadline) != nil {
		t.Fatal("delayed callback/socket deadline fixture")
	}
	got, err := readCollection(ctx, client, "r", DefaultLimits())
	if !errors.Is(err, context.DeadlineExceeded) || !reflect.DeepEqual(got, inventory.Snapshot{}) {
		t.Fatalf("expired original window classified as %v", err)
	}
}
func TestCollectionReplyAfterDeadlineCannotPublish(t *testing.T) {
	var wire bytes.Buffer
	if writeSnapshot(&wire, "r", observation(), DefaultLimits()) != nil {
		t.Fatal("complete source fixture")
	}
	ctx := unpublishedDeadline{context.Background(), time.Now().Add(-time.Second)}
	got, err := readCollection(ctx, &wire, "r", DefaultLimits())
	if !errors.Is(err, context.DeadlineExceeded) || !reflect.DeepEqual(got, inventory.Snapshot{}) {
		t.Fatal("late reply accepted before callback", err)
	}
}
func TestCollectionKeepsOnTimeResultsAndExplicitCancellation(t *testing.T) {
	for _, label := range []string{"complete", "malformed", "cancelled", "no-deadline"} {
		t.Run(label, func(t *testing.T) {
			var wire bytes.Buffer
			if writeSnapshot(&wire, "r", observation(), DefaultLimits()) != nil {
				t.Fatal("source fixture")
			}
			var ctx context.Context = unpublishedDeadline{context.Background(), time.Now().Add(time.Hour)}
			if label == "malformed" {
				wire.Reset()
				wire.Write(frame(`{}`))
			}
			if label == "cancelled" {
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			if label == "no-deadline" {
				ctx = context.Background()
			}
			got, err := readCollection(ctx, &wire, "r", DefaultLimits())
			switch label {
			case "malformed":
				if !errors.Is(err, ErrProtocol) || !reflect.DeepEqual(got, inventory.Snapshot{}) {
					t.Fatal("early malformed reply lost protocol failure", err)
				}
			case "cancelled":
				if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(got, inventory.Snapshot{}) {
					t.Fatal("explicit cancellation changed", err)
				}
			default:
				if err != nil || !reflect.DeepEqual(got, observation()) {
					t.Fatal("on-time reply changed", err)
				}
			}
		})
	}
}
