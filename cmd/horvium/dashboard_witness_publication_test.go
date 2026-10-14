// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

func TestDashboardWitnessPublicationIsSeparateAndCoalesced(t *testing.T) {
	store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
	if e != nil {
		t.Fatal(e)
	}
	defer store.close()
	ref, e := store.capture(bytes.NewReader(witnessFrames(witnessGolden())), witnessTestID, "native-small", "management-1", "helper-1")
	if e != nil {
		t.Fatal(e)
	}
	var calls atomic.Int32
	read := func(id string) (inventory.Snapshot, dashboardWitnessRef, error) {
		s, e := store.original(ref)
		return s, *ref, e
	}
	lookup := func(ctx context.Context, id string) (witnessPublication, error) {
		calls.Add(1)
		return witnessPublication{ID: id, Complete: true, Host: "ok", GEOM: "ok", Pools: "ok", Datasets: "ok", PoolCount: 1, DatasetCount: 1}, nil
	}
	registry := newDashboardWitnessRegistry(read, lookup)
	if _, _, e = registry.confirm(context.Background(), witnessTestID); e == nil || calls.Load() != 0 {
		t.Fatal("complete source alone became publication")
	}
	// A failed candidate cannot resume. Confirm the delivered original in a new
	// bounded registry, while multiple callers share one real metadata operation.
	if store.forward(ref, &bytes.Buffer{}) != nil {
		t.Fatal("deliver")
	}
	registry = newDashboardWitnessRegistry(read, lookup)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, r, e := registry.confirm(context.Background(), witnessTestID)
			if e != nil || s.Datasets.Data[0].Used != 9007199254740993 || r.ID != witnessTestID {
				t.Error("confirmed original", e)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 || registry.controls() != 1 {
		t.Fatal("per-sample metadata calls")
	}
	registry.setIdle(true)
	if _, _, e = registry.confirm(context.Background(), witnessTestID); e != nil {
		t.Fatal("immutable confirmed lookup during idle", e)
	}
	if _, _, e = registry.confirm(context.Background(), strings.Replace(witnessTestID, "-1", "-2", 1)); e == nil || calls.Load() != 1 {
		t.Fatal("HTTP during idle or missing lineage accepted")
	}
}

func TestDashboardWitnessMetadataMismatchAndFailureArePermanent(t *testing.T) {
	for _, change := range []string{"expired", "id", "status", "count", "cancelled"} {
		t.Run(change, func(t *testing.T) {
			s, _ := dashboardDecodeWitness(bytes.NewReader(witnessFrames(witnessGolden())), witnessTestID)
			calls := 0
			reg := newDashboardWitnessRegistry(func(id string) (inventory.Snapshot, dashboardWitnessRef, error) {
				return s, dashboardWitnessRef{ID: id, Complete: true, Delivered: true, RawSHA256: strings.Repeat("a", 64), ForwardSHA256: strings.Repeat("a", 64)}, nil
			}, func(ctx context.Context, id string) (witnessPublication, error) {
				calls++
				p := witnessPublication{ID: id, Complete: true, Host: "ok", GEOM: "ok", Pools: "ok", Datasets: "ok", PoolCount: 1, DatasetCount: 1}
				switch change {
				case "expired":
					return p, errors.New("expired")
				case "id":
					p.ID += "0"
				case "status":
					p.Pools = "unavailable"
				case "count":
					p.DatasetCount++
				case "cancelled":
					return p, context.Canceled
				}
				return p, nil
			})
			if _, _, e := reg.confirm(context.Background(), witnessTestID); e == nil {
				t.Fatal("uncertain publication accepted")
			}
			if _, _, e := reg.confirm(context.Background(), witnessTestID); e == nil || calls != 1 {
				t.Fatal("failed publication retried")
			}
		})
	}
}

func TestDashboardWitnessChangedValuesSelectOriginalAndRejectMutation(t *testing.T) {
	original := dashboardRichSnapshot()
	h, m, c := dashboardHTMLFixture(t, original)
	dashboardHTMLWait(t, func() bool { o, e := m.Overview(context.Background(), ""); return e == nil && o.Complete })
	plan, _ := dashboardPlan(len(original.Datasets.Data))
	lookup := func(ctx context.Context, id string) (witnessPublication, error) {
		o, e := m.Overview(ctx, id)
		return witnessPublicationFromOverview(o), e
	}
	read := func(id string) (inventory.Snapshot, dashboardWitnessRef, error) {
		return original, dashboardWitnessRef{ID: id, Complete: true, Delivered: true, RawSHA256: strings.Repeat("b", 64), ForwardSHA256: strings.Repeat("b", 64)}, nil
	}
	reg := newDashboardWitnessRegistry(read, lookup)
	predicate := reg.predicate(plan)
	request := plan[1]
	response := dashboardHTMLRequest(h, c, "GET", request.Path, "")
	body := response.Body.Bytes()
	if e := predicate(request, 200, body); e != nil {
		t.Fatal("real generation original", e)
	}
	// The exact original drives expectations, even when the new values would
	// match another complete snapshot. No API/accounting values supply a baseline.
	changed := bytes.Replace(body, []byte(strconv.FormatUint(original.Pools.Data[0].Allocated, 10)+" B"), []byte("999 B"), 1)
	if bytes.Equal(changed, body) {
		t.Fatal("mutation not applied")
	}
	if e := predicate(request, 200, changed); e == nil {
		t.Fatal("corrupt exact cell accepted")
	}
	o, _ := m.Overview(context.Background(), "")
	wrong := bytes.ReplaceAll(body, []byte(o.Generation), []byte(witnessTestID))
	if e := predicate(request, 200, wrong); e == nil {
		t.Fatal("same numbers with wrong lineage accepted")
	}
}
