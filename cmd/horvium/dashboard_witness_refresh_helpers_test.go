// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Only bounded response data and local times: never session credentials.
type witnessRefreshReply struct {
	Index, Status         int
	StartedNS, FinishedNS int64
	Body                  []byte
}

func (r witnessRefreshReply) busy() bool {
	return r.Status == http.StatusConflict && r.FinishedNS >= r.StartedNS && r.StartedNS > 0 && bytes.Equal(bytes.TrimSpace(r.Body), []byte(`{"code":"busy","message":"Inventory work already active"}`))
}
func (r witnessRefreshReply) accepted() bool {
	return r.Status == http.StatusAccepted && r.FinishedNS >= r.StartedNS && r.StartedNS > 0 && bytes.Equal(bytes.TrimSpace(r.Body), []byte(`{"code":"refresh_requested"}`))
}

// Exactly one authenticated operator request; no status/transport retries.
func witnessRefreshOnce(client *http.Client, base, csrf string, index int) (witnessRefreshReply, error) {
	r := witnessRefreshReply{Index: index, StartedNS: time.Now().UnixNano()}
	response, err := client.PostForm(base+"/api/refresh", url.Values{"csrf": {csrf}})
	if err == nil {
		r.Status = response.StatusCode
		r.Body, err = io.ReadAll(io.LimitReader(response.Body, 65537))
		closeErr := response.Body.Close()
		if err == nil {
			err = closeErr
		}
		if len(r.Body) > 65536 {
			r.Body = r.Body[:65536]
			err = errDashboardWitness
		}
	}
	r.FinishedNS = time.Now().UnixNano()
	if err != nil {
		return r, errDashboardWitness
	}
	return r, nil
}

// A fixed three-way start barrier. Drain all terminal replies even on failure.
func witnessRefreshOverlap(client *http.Client, base, csrf string) ([]witnessRefreshReply, error) {
	type result struct {
		reply witnessRefreshReply
		err   error
	}
	start := make(chan struct{})
	results := make(chan result, 3)
	for i := 1; i <= 3; i++ {
		go func(index int) {
			<-start
			r, e := witnessRefreshOnce(client, base, csrf, index)
			results <- result{r, e}
		}(i)
	}
	close(start)
	replies := make([]witnessRefreshReply, 3)
	var err error
	for range 3 {
		r := <-results
		replies[r.reply.Index-1] = r.reply
		if r.err != nil {
			err = errDashboardWitness
		}
	}
	return replies, err
}
