// SPDX-License-Identifier: BSD-3-Clause
package observation

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
	"testing"
	"time"
)

func TestObserverReportsPublishedOutcomeWithoutSourceText(t *testing.T) {
	events := make(chan Result, 2)
	m, e := NewObserved(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) {
		return inventory.Snapshot{}, errors.New("secret raw source text")
	}), func(r Result) { events <- r })
	if e != nil {
		t.Fatal(e)
	}
	running(t, m)
	select {
	case r := <-events:
		if r.Attempt.ErrorCode != "failed" || r.Attempt.ID == "" || r.Attempt.FinishedAt.IsZero() {
			t.Fatal("invalid event")
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}
