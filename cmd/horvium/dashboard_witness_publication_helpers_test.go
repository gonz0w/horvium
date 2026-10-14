// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

type witnessPublication struct {
	ID                                                 string
	Complete                                           bool
	Host, GEOM, Pools, Datasets                        inventory.Status
	Classes, Geoms, Providers, PoolCount, DatasetCount int
}

func witnessPublicationFromOverview(o observation.Overview) witnessPublication {
	return witnessPublication{ID: o.Generation, Complete: o.Complete, Host: o.HostState.Status, GEOM: o.GEOM.Status, Pools: o.Pools.Status, Datasets: o.Datasets.Status, Classes: o.Classes, Geoms: o.Geoms, Providers: o.Providers, PoolCount: o.PoolCount, DatasetCount: o.DatasetCount}
}

type witnessOriginalReader func(string) (inventory.Snapshot, dashboardWitnessRef, error)
type witnessMetadataReader func(context.Context, string) (witnessPublication, error)
type dashboardWitnessRegistry struct {
	mu                 sync.Mutex
	read               witnessOriginalReader
	lookup             witnessMetadataReader
	entries            map[string]*witnessConfirmation
	failed, idle       bool
	failure            error
	operations, active int
}
type witnessConfirmation struct {
	done      chan struct{}
	ref       dashboardWitnessRef
	err       error
	once      sync.Once
	predicate dashboardPredicate
}

func newDashboardWitnessRegistry(r witnessOriginalReader, m witnessMetadataReader) *dashboardWitnessRegistry {
	return &dashboardWitnessRegistry{read: r, lookup: m, entries: map[string]*witnessConfirmation{}}
}
func (r *dashboardWitnessRegistry) confirm(ctx context.Context, id string) (inventory.Snapshot, dashboardWitnessRef, error) {
	deny := func(cause error) (inventory.Snapshot, dashboardWitnessRef, error) {
		r.mu.Lock()
		r.failed = true
		if r.failure == nil {
			r.failure = cause
		}
		failure := r.failure
		r.mu.Unlock()
		return inventory.Snapshot{}, dashboardWitnessRef{}, failure
	}
	if !dashboardGenerationPattern.MatchString(id) || ctx.Err() != nil {
		return deny(witnessFailure("publication-context"))
	}
	r.mu.Lock()
	if r.failed {
		failure := r.failure
		r.mu.Unlock()
		return inventory.Snapshot{}, dashboardWitnessRef{}, witnessFailureFrom(failure, "publication-failed")
	}
	entry, exists := r.entries[id]
	if !exists {
		if r.idle || len(r.entries) >= 2*witnessAttemptCap || r.read == nil || r.lookup == nil {
			r.mu.Unlock()
			return deny(witnessFailure("publication-capacity"))
		}
		entry = &witnessConfirmation{done: make(chan struct{})}
		r.entries[id] = entry
		r.active++
	}
	r.mu.Unlock()
	if exists {
		select {
		case <-entry.done:
		case <-ctx.Done():
			return deny(witnessFailure("publication-context"))
		}
		if entry.err != nil {
			return deny(entry.err)
		}
		s, ref, e := r.read(id)
		if e != nil || ref != entry.ref {
			return deny(witnessFailureFrom(e, "publication-cache"))
		}
		return s, ref, nil
	}
	s, ref, e := r.read(id)
	if e != nil {
		e = witnessFailureFrom(e, "publication-original")
	}
	if e == nil && (ref.CaptureFailure != "" || !ref.Complete || !ref.Delivered || ref.ID != id || !s.Complete() || len(ref.RawSHA256) != 64 || ref.RawSHA256 != ref.ForwardSHA256) {
		e = witnessFailure("publication-eligibility")
	}
	if e == nil {
		r.mu.Lock()
		if r.idle || r.failed {
			e = witnessFailure("publication-idle")
		} else {
			r.operations++
		}
		r.mu.Unlock()
	}
	if e == nil {
		bounded, cancel := context.WithTimeout(ctx, 45*time.Second)
		metadata, me := r.lookup(bounded, id)
		cancel()
		want := witnessPublication{ID: id, Complete: s.Complete(), Host: s.Host.Status, GEOM: s.GEOM.Status, Pools: s.Pools.Status, Datasets: s.Datasets.Status, Classes: len(s.GEOM.Data.Classes), Geoms: len(s.GEOM.Data.Geoms), Providers: len(s.GEOM.Data.Providers), PoolCount: len(s.Pools.Data), DatasetCount: len(s.Datasets.Data)}
		if me != nil {
			e = witnessFailureFrom(me, "publication-metadata")
		} else if metadata != want {
			e = witnessFailure("publication-mismatch")
		}
	}
	if ctx.Err() != nil {
		e = witnessFailure("publication-context")
	}
	r.mu.Lock()
	if e == nil && r.failed {
		e = witnessFailureFrom(r.failure, "publication-failed")
	}
	entry.ref = ref
	entry.err = e
	r.active--
	if e != nil {
		r.failed = true
		if r.failure == nil {
			r.failure = e
		}
	}
	close(entry.done)
	r.mu.Unlock()
	if e != nil {
		return inventory.Snapshot{}, dashboardWitnessRef{}, e
	}
	return s, ref, nil
}
func (r *dashboardWitnessRegistry) controls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.operations
}
func (r *dashboardWitnessRegistry) setIdle(idle bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if idle && r.active != 0 {
		r.failed = true
	}
	r.idle = idle
}
func (r *dashboardWitnessRegistry) predicate(plan []dashboardRequest) dashboardPredicate {
	return func(request dashboardRequest, status int, body []byte) error {
		if request.Index < 0 || request.Index >= len(plan) || request != plan[request.Index] {
			return errDashboardWitness
		}
		root, e := dashboardParseHTML(body)
		if e != nil {
			return e
		}
		workspace, e := root.one("main", "id", "workspace")
		if e != nil {
			return e
		}
		if status == 409 && request.Case == "overview" {
			heading, e := workspace.one("h1", "", "")
			if e != nil || heading.text() != "Overview" || len(workspace.find("script", "", "")) != 0 || len(workspace.find("aside", "class", "detail")) != 0 {
				return errDashboardWitness
			}
			return dashboardExpiryHTML(workspace)
		}
		generation, e := workspace.one("p", "class", "generation")
		if e != nil {
			return e
		}
		code, e := generation.one("code", "", "")
		if e != nil {
			return e
		}
		id := code.rawText()
		s, _, e := r.confirm(context.Background(), id)
		if e != nil {
			return e
		}
		r.mu.Lock()
		entry := r.entries[id]
		r.mu.Unlock()
		entry.once.Do(func() { entry.predicate, _ = newDashboardHTMLPredicate(s, plan) })
		if entry.predicate == nil {
			return errDashboardWitness
		}
		return entry.predicate(request, status, body)
	}
}

// Only metadata crosses into confirmation; Host/accounting values never enter
// expectations. No refresh, retry or alternate-generation fallback is attempted.
func witnessHTTPMetadata(client *http.Client, base string) witnessMetadataReader {
	return witnessHTTPMetadataCapture(client, base, nil)
}
func witnessHTTPMetadataCapture(client *http.Client, base string, original func(string, []byte) error) witnessMetadataReader {
	return func(ctx context.Context, id string) (witnessPublication, error) {
		req, e := http.NewRequestWithContext(ctx, "GET", base+"/api/overview?generation="+url.QueryEscape(id), nil)
		if e != nil {
			return witnessPublication{}, errDashboardWitness
		}
		response, e := client.Do(req)
		if e != nil {
			return witnessPublication{}, errDashboardWitness
		}
		b, re := io.ReadAll(io.LimitReader(response.Body, witnessFrameCap+1))
		ce := response.Body.Close()
		var o observation.Overview
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		_, je := witnessJSONValue(d, 0)
		_, end := d.Token()
		if re != nil || ce != nil || len(b) > witnessFrameCap || response.StatusCode != 200 || je != nil || end != io.EOF || json.Unmarshal(b, &o) != nil {
			return witnessPublication{}, errDashboardWitness
		}
		if original != nil && original(id, b) != nil {
			return witnessPublication{}, witnessFailure("publication-original")
		}
		return witnessPublicationFromOverview(o), nil
	}
}
