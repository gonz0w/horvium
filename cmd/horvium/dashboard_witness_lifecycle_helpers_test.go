// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

type witnessRequestCounter struct {
	transport http.RoundTripper
	requests  atomic.Int64
}

func (c *witnessRequestCounter) RoundTrip(r *http.Request) (*http.Response, error) {
	c.requests.Add(1)
	return c.transport.RoundTrip(r)
}

type witnessConfirmationReply struct {
	Index                 int
	ID, RawSHA256         string
	StartedNS, FinishedNS int64
	Denied                bool
}

// Three simultaneous callers share the registry, never retry and all finish.
func witnessConfirmConcurrent(ctx context.Context, registry *dashboardWitnessRegistry, id string) ([]witnessConfirmationReply, error) {
	start := make(chan struct{})
	done := make(chan witnessConfirmationReply, 3)
	for i := 1; i <= 3; i++ {
		go func(index int) {
			<-start
			r := witnessConfirmationReply{Index: index, StartedNS: time.Now().UnixNano()}
			snapshot, ref, e := registry.confirm(ctx, id)
			r.FinishedNS = time.Now().UnixNano()
			r.Denied = e != nil || !snapshot.Complete()
			if !r.Denied {
				r.ID = ref.ID
				r.RawSHA256 = ref.RawSHA256
			}
			done <- r
		}(i)
	}
	close(start)
	replies := make([]witnessConfirmationReply, 3)
	var err error
	for range 3 {
		r := <-done
		replies[r.Index-1] = r
		if r.Denied {
			err = errDashboardWitness
		}
	}
	return replies, err
}
