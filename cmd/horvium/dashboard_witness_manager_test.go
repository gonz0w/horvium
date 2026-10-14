// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

// The fake producer declares its model independently from the literal framed
// originals. Management numeric APIs never build or refresh these expectations.
type witnessManagerSource struct {
	mu                    sync.Mutex
	store                 *dashboardWitnessStore
	refs                  map[string]*dashboardWitnessRef
	changed, modelFailure bool
	pause                 bool
	captured              chan string
}

func (s *witnessManagerSource) Collect(ctx context.Context, id string) (inventory.Snapshot, error) {
	s.mu.Lock()
	changed, fail, pause := s.changed, s.modelFailure, s.pause
	s.mu.Unlock()
	raw := bytes.ReplaceAll(witnessFrames(witnessGolden()), []byte(witnessTestID), []byte(id))
	if changed {
		raw = bytes.ReplaceAll(raw, []byte(`"allocated_bytes":9007199254740993`), []byte(`"allocated_bytes":9007199254740994`))
		raw = bytes.ReplaceAll(raw, []byte(`"used_bytes":9007199254740993`), []byte(`"used_bytes":9007199254740994`))
	}
	// Replacement JSON values here have equal width, keeping literal frame bounds.
	ref, e := s.store.capture(bytes.NewReader(raw), id, "native-small", "manager-actor", "helper-actor")
	if e != nil {
		return inventory.Snapshot{}, e
	}
	if e = s.store.forward(ref, &bytes.Buffer{}); e != nil {
		return inventory.Snapshot{}, e
	}
	s.mu.Lock()
	s.refs[id] = ref
	s.mu.Unlock()
	start, _ := time.Parse(time.RFC3339, "2026-10-13T00:00:00Z")
	finish := start.Add(time.Second)
	mem := uint64(1)
	v := uint64(9007199254740993)
	if changed {
		v++
	}
	snapshot := inventory.Snapshot{StartedAt: start, FinishedAt: finish, Host: inventory.Section[inventory.Host]{Status: inventory.OK, StartedAt: start, FinishedAt: finish, Data: inventory.Host{InstalledKernel: "fixture", RunningKernel: "fixture", Userland: "fixture", Architecture: "amd64", PhysicalMemory: &mem, ZFSUserland: "fixture", ZFSKernel: "fixture"}}, GEOM: inventory.Section[inventory.GEOM]{Status: inventory.OK, StartedAt: start, FinishedAt: finish}, Pools: inventory.Section[[]inventory.Pool]{Status: inventory.OK, StartedAt: start, FinishedAt: finish, Data: []inventory.Pool{{Name: "fixture", GUID: "18446744073709551615", Size: ^uint64(0), Allocated: v, Free: 18437736874454810622, Health: "ONLINE"}}}, Datasets: inventory.Section[[]inventory.Dataset]{Status: inventory.OK, StartedAt: start, FinishedAt: finish, Data: []inventory.Dataset{{Name: "fixture/data", GUID: "42", Used: v, Available: ^uint64(0), Referenced: 1}}}}
	if fail {
		snapshot.Datasets.Data = append(snapshot.Datasets.Data, snapshot.Datasets.Data[0])
	}
	if s.captured != nil {
		s.captured <- id
	}
	if pause {
		<-ctx.Done()
	}
	return snapshot, nil
}
func (s *witnessManagerSource) original(id string) (inventory.Snapshot, dashboardWitnessRef, error) {
	s.mu.Lock()
	ref := s.refs[id]
	s.mu.Unlock()
	if ref == nil {
		return inventory.Snapshot{}, dashboardWitnessRef{}, errDashboardWitness
	}
	snapshot, e := s.store.original(ref)
	return snapshot, *ref, e
}
func newWitnessManagerSource(t *testing.T) *witnessManagerSource {
	t.Helper()
	store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { store.close() })
	return &witnessManagerSource{store: store, refs: map[string]*dashboardWitnessRef{}}
}
func TestDashboardWitnessRealChangedPublicationsAndModelRejection(t *testing.T) {
	source := newWitnessManagerSource(t)
	manager, e := observation.New(observation.DefaultConfig(), source)
	if e != nil {
		t.Fatal(e)
	}
	h, m, c := dashboardHTMLManagerFixture(t, manager)
	plan, _ := dashboardPlan(1)
	lookup := func(ctx context.Context, id string) (witnessPublication, error) {
		o, e := m.Overview(ctx, id)
		return witnessPublicationFromOverview(o), e
	}
	reg := newDashboardWitnessRegistry(source.original, lookup)
	predicate := reg.predicate(plan)
	first, _ := m.Overview(context.Background(), "")
	body := dashboardHTMLRequest(h, c, "GET", plan[1].Path, "").Body.Bytes()
	if predicate(plan[1], 200, body) != nil {
		t.Fatal("first literal original")
	}
	source.mu.Lock()
	source.changed = true
	source.mu.Unlock()
	if m.Refresh() != nil {
		t.Fatal("refresh")
	}
	dashboardHTMLWait(t, func() bool { o, _ := m.Overview(context.Background(), ""); return o.Generation != first.Generation })
	second, _ := m.Overview(context.Background(), "")
	next := dashboardHTMLRequest(h, c, "GET", plan[1].Path, "").Body.Bytes()
	if bytes.Equal(body, next) || predicate(plan[1], 200, next) != nil || !bytes.Contains(next, []byte("9007199254740994 B")) {
		t.Fatal("changed generation did not use its exact independent original")
	}
	corrupt := bytes.ReplaceAll(next, []byte("9007199254740994 B"), []byte("9007199254740993 B"))
	if predicate(plan[1], 200, corrupt) == nil {
		t.Fatal("old numbers silently rebaselined new generation")
	}
	forged := bytes.ReplaceAll(body, []byte(first.Generation), []byte(second.Generation))
	if predicate(plan[1], 200, forged) == nil {
		t.Fatal("equal old values with new lineage accepted")
	}
	source.mu.Lock()
	source.modelFailure = true
	source.mu.Unlock()
	previous := m.State().LastCompleted.ID
	if m.Refresh() != nil {
		t.Fatal("rejecting refresh")
	}
	dashboardHTMLWait(t, func() bool { s := m.State(); return s.LastCompleted.ID != previous && !s.Collecting })
	failed := m.State().LastCompleted.ID
	s, ref, e := source.original(failed)
	if e != nil || !s.Complete() || !ref.Delivered || m.State().CurrentID != second.Generation {
		t.Fatal("complete original did not preserve failed publication boundary", e)
	}
	if _, _, e := reg.confirm(context.Background(), failed); e == nil {
		t.Fatal("real clone/model rejection treated as publication")
	}
}
func TestDashboardWitnessRealCancellationCannotPublishCompleteOriginal(t *testing.T) {
	source := newWitnessManagerSource(t)
	source.pause = true
	source.captured = make(chan string, 1)
	m, e := observation.New(observation.DefaultConfig(), source)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()
	var id string
	select {
	case id = <-source.captured:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("original capture")
	}
	cancel()
	<-done
	s, ref, e := source.original(id)
	if e != nil || !s.Complete() || !ref.Delivered || m.State().CurrentID != "" {
		t.Fatal("cancellation boundary", e)
	}
	reg := newDashboardWitnessRegistry(source.original, func(ctx context.Context, id string) (witnessPublication, error) {
		o, e := m.Overview(ctx, id)
		return witnessPublicationFromOverview(o), e
	})
	if _, _, e := reg.confirm(context.Background(), id); e == nil {
		t.Fatal("cancelled original became publication")
	}
}
