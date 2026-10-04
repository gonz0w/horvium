// SPDX-License-Identifier: BSD-3-Clause
package observation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/inventory"
)

type sourceFunc func(context.Context, string) (inventory.Snapshot, error)

func (f sourceFunc) Collect(c context.Context, id string) (inventory.Snapshot, error) {
	return f(c, id)
}
func sample(at time.Time, n int) inventory.Snapshot {
	memory := uint64(16 << 30)
	s := inventory.Snapshot{StartedAt: at, FinishedAt: at}
	s.Host = inventory.Section[inventory.Host]{Status: inventory.OK, StartedAt: at, FinishedAt: at, Data: inventory.Host{InstalledKernel: "15.1-RELEASE", RunningKernel: "15.1-RELEASE", Userland: "15.1-RELEASE", Architecture: "amd64", PhysicalMemory: &memory, ZFSUserland: "zfs-2.4.2", ZFSKernel: "zfs-kmod-2.4.2"}}
	s.GEOM = inventory.Section[inventory.GEOM]{Status: inventory.OK, StartedAt: at, FinishedAt: at}
	s.Pools = inventory.Section[[]inventory.Pool]{Status: inventory.OK, StartedAt: at, FinishedAt: at, Data: []inventory.Pool{{Name: "fixture", GUID: "1", Size: 1024, Free: 1024, Health: "ONLINE"}}}
	s.Datasets = inventory.Section[[]inventory.Dataset]{Status: inventory.OK, StartedAt: at, FinishedAt: at}
	for i := 0; i < n; i++ {
		s.Datasets.Data = append(s.Datasets.Data, inventory.Dataset{Name: fmt.Sprintf("fixture/data-%05d", i), GUID: fmt.Sprint(i + 1), Used: uint64(n - i)})
	}
	return s
}
func running(t *testing.T, m *Manager) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Error(err)
			}
		case <-time.After(2 * time.Second):
			t.Error("collector did not stop")
		}
	})
	return cancel, done
}
func await(t *testing.T, predicate func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !predicate() {
		if time.Now().After(deadline) {
			t.Fatal("expected state did not arrive")
		}
		time.Sleep(time.Millisecond)
	}
}
func quietConfig() Config { c := DefaultConfig(); c.Interval = time.Hour; return c }
func TestStartupAndRefreshAreSingleFlight(t *testing.T) {
	entered := make(chan string, 2)
	release := make(chan struct{})
	var calls atomic.Int32
	m, err := New(quietConfig(), sourceFunc(func(ctx context.Context, id string) (inventory.Snapshot, error) {
		calls.Add(1)
		entered <- id
		select {
		case <-release:
			return sample(time.Now(), 3), nil
		case <-ctx.Done():
			return inventory.Snapshot{}, ctx.Err()
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	running(t, m)
	select {
	case id := <-entered:
		if id == "" {
			t.Fatal("missing request ID")
		}
	case <-time.After(time.Second):
		t.Fatal("no startup collection")
	}
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			if err := m.Refresh(); !errors.Is(err, ErrBusy) {
				t.Errorf("concurrent refresh: %v", err)
			}
		})
	}
	wg.Wait()
	if _, err := m.Datasets(context.Background(), Query{}); !errors.Is(err, ErrUnavailable) {
		t.Fatal("initial inventory fabricated")
	}
	close(release)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	if calls.Load() != 1 {
		t.Fatal("duplicate refresh queued")
	}
	if err := m.Refresh(); err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return calls.Load() == 2 && !m.State().Collecting })
	if m.State().CurrentID == m.State().LastAttempt.ID && m.State().LastAttempt.ID == "" {
		t.Fatal("missing generation")
	}
}
func TestFailurePreservesGenerationAndDoesNotLeakErrorText(t *testing.T) {
	var fail atomic.Bool
	m, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) {
		if fail.Load() {
			return sample(time.Now(), 99), fmt.Errorf("operator-sensitive-dump: %w", helper.ErrProtocol)
		}
		return sample(time.Now(), 2), nil
	}))
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	id := m.State().CurrentID
	fail.Store(true)
	if err := m.Refresh(); err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return m.State().LastAttempt.ErrorCode != "" })
	state := m.State()
	if state.CurrentID != id || state.CompleteID != id || state.LastAttempt.ErrorCode != "protocol" || !state.Datasets.Stale {
		t.Fatal("failure replaced data, hid error or appeared fresh")
	}
	page, err := m.Datasets(context.Background(), Query{})
	if err != nil || len(page.Rows) != 2 || page.Generation != id || !page.Freshness.Stale {
		t.Fatal("fallback rows missing or mixed")
	}
}
func TestGenerationPagingExpiryAndLastCompleteRetention(t *testing.T) {
	var number atomic.Int32
	m, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) {
		n := number.Load()
		s := sample(time.Now(), 3)
		if n > 0 {
			s.Datasets.Data[0].Name = fmt.Sprintf("fixture/generation-%d", n)
		}
		if n == 1 || n == 2 {
			s.Pools.Status = inventory.Unavailable
			s.Pools.Data = nil
			s.Pools.Problems = []inventory.Problem{{Source: "pools", Code: "failed"}}
		}
		return s, nil
	}))
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	first := m.State().CurrentID
	refresh := func(n int32) string {
		t.Helper()
		previous := m.State().CurrentID
		number.Store(n)
		if err := m.Refresh(); err != nil {
			t.Fatal(err)
		}
		await(t, func() bool { return m.State().CurrentID != previous && !m.State().Collecting })
		return m.State().CurrentID
	}
	second := refresh(1)
	third := refresh(2)
	if m.State().CompleteID != first {
		t.Fatal("last complete lost after partial observations")
	}
	if _, err := m.Datasets(context.Background(), Query{Generation: second}); !errors.Is(err, ErrExpired) {
		t.Fatal("third retained generation")
	}
	page, err := m.Datasets(context.Background(), Query{Generation: first, Limit: 1})
	if err != nil || page.Generation != first || page.Rows[0].Name != "fixture/data-00000" {
		t.Fatal("pinned page mixed generations")
	}
	refresh(3)
	if _, err := m.Datasets(context.Background(), Query{Generation: first}); !errors.Is(err, ErrExpired) {
		t.Fatal("old generation did not expire")
	}
	if _, err := m.Datasets(context.Background(), Query{Generation: third}); err != nil {
		t.Fatal("previous latest was not retained")
	}
}
func TestImmutableOwnedGenerations(t *testing.T) {
	s := sample(time.Now(), 3)
	m, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) { return s, nil }))
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	s.Datasets.Data[0].Name = "changed by source"
	page, err := m.Datasets(context.Background(), Query{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Rows[0].Name != "fixture/data-00000" {
		t.Fatal("source owns cached data")
	}
	page.Rows[0].Name = "changed by reader"
	again, _ := m.Datasets(context.Background(), Query{Limit: 1})
	if again.Rows[0].Name != "fixture/data-00000" {
		t.Fatal("reader mutated generation")
	}
}
func TestNumericSortFilterAndQueryBounds(t *testing.T) {
	m, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) { return sample(time.Now(), 12), nil }))
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	p, err := m.Datasets(context.Background(), Query{Sort: "guid", Offset: 1, Limit: 2})
	if err != nil || p.Total != 12 || p.Rows[0].GUID != "2" || p.Rows[1].GUID != "3" || !p.More {
		t.Fatal("GUID sorted lexically or page bounds incorrect")
	}
	p, err = m.Datasets(context.Background(), Query{Filter: "data-0001", Sort: "used", Descending: true, Limit: 500})
	if err != nil || p.Total != 2 || p.Rows[0].GUID != "11" {
		t.Fatal("filter/numeric ordering incorrect")
	}
	for _, q := range []Query{{Limit: 501}, {Limit: -1}, {Offset: -1}, {Sort: "command"}, {Filter: "\x00"}} {
		if _, err := m.Datasets(context.Background(), q); !errors.Is(err, ErrQuery) {
			t.Fatalf("query accepted: %v", err)
		}
	}
	p, err = m.Datasets(context.Background(), Query{Offset: int(^uint(0) >> 1)})
	if err != nil || len(p.Rows) != 0 || p.More {
		t.Fatal("offset overflow or invalid empty page")
	}
}

func TestSectionFreshnessDoesNotBorrowAnotherSectionsTimestamp(t *testing.T) {
	base := time.Now().UTC()
	var clock atomic.Int64
	clock.Store(base.UnixNano())
	var partial atomic.Bool
	now := func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	m, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) {
		s := sample(now(), 1)
		if partial.Load() {
			s.Datasets.Status = inventory.Unavailable
			s.Datasets.Data = nil
			s.Datasets.Problems = []inventory.Problem{{Source: "datasets", Code: "permission"}}
		}
		return s, nil
	}))
	m.now = now
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	clock.Store(base.Add(89 * time.Second).UnixNano())
	if m.State().Datasets.Stale {
		t.Fatal("fresh section aged too early")
	}
	clock.Store(base.Add(90 * time.Second).UnixNano())
	if !m.State().Datasets.Stale {
		t.Fatal("90-second boundary not stale")
	}
	partial.Store(true)
	if err := m.Refresh(); err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return !m.State().Collecting && m.State().Datasets.Status == inventory.Unavailable })
	state := m.State()
	if state.Host.Stale || !state.Datasets.Stale || !state.Datasets.LastSuccessfulAt.Equal(base) || !state.Host.LastSuccessfulAt.Equal(base.Add(90*time.Second)) {
		t.Fatal("unrelated section refresh changed failed-section age")
	}
	p, err := m.Datasets(context.Background(), Query{})
	if err != nil || p.Total != 0 || p.Freshness.Status != inventory.Unavailable || p.Freshness.Problems[0].Code != "permission" {
		t.Fatal("unavailable list presented as successful zero")
	}
	state.Datasets.Problems[0].Code = "reader changed it"
	if m.State().Datasets.Problems[0].Code != "permission" {
		t.Fatal("metadata shared mutable slice")
	}
}

func TestScheduledTicksDuringWorkAreDiscarded(t *testing.T) {
	ticks := make(chan time.Time, 1)
	entered := make(chan struct{}, 2)
	release := make(chan struct{}, 2)
	var calls atomic.Int32
	m, _ := New(quietConfig(), sourceFunc(func(ctx context.Context, _ string) (inventory.Snapshot, error) {
		calls.Add(1)
		entered <- struct{}{}
		select {
		case <-release:
			return sample(time.Now(), 1), nil
		case <-ctx.Done():
			return inventory.Snapshot{}, ctx.Err()
		}
	}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.run(ctx, ticks) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("scheduled worker did not stop")
		}
	})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("startup did not enter")
	}
	ticks <- time.Now()
	release <- struct{}{}
	await(t, func() bool { return !m.State().Collecting })
	select {
	case <-entered:
		t.Fatal("busy scheduled tick queued another collection")
	case <-time.After(25 * time.Millisecond):
	}
	ticks <- time.Now()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("next interval did not collect")
	}
	release <- struct{}{}
	await(t, func() bool { return !m.State().Collecting })
	if calls.Load() != 2 {
		t.Fatal("incorrect periodic work count")
	}
}

func TestCollectionDeadlineAndShutdownClearVolatileState(t *testing.T) {
	var stall atomic.Bool
	m, _ := New(quietConfig(), sourceFunc(func(ctx context.Context, _ string) (inventory.Snapshot, error) {
		if stall.Load() {
			<-ctx.Done()
			return sample(time.Now(), 3), nil
		}
		return sample(time.Now(), 1), nil
	}))
	m.config.Timeout = 25 * time.Millisecond
	cancel, done := running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	id := m.State().CurrentID
	stall.Store(true)
	if err := m.Refresh(); err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return m.State().LastAttempt.ErrorCode == "timeout" })
	if m.State().CurrentID != id {
		t.Fatal("late result published after deadline")
	}
	cancel()
	// running's cleanup consumes done; wait on public state to avoid double receives.
	_ = done
	await(t, func() bool { return !m.State().Running })
	if m.State().CurrentID != "" || m.State().CompleteID != "" {
		t.Fatal("shutdown retained volatile inventory")
	}
	if err := m.Refresh(); !errors.Is(err, ErrNotRunning) {
		t.Fatal("stopped collector refreshed")
	}
	next, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) { return sample(time.Now(), 1), nil }))
	running(t, next)
	await(t, func() bool { return next.State().CurrentID != "" })
	if next.State().CurrentID == id {
		t.Fatal("restart reused a generation ID")
	}
	if _, err := next.Datasets(context.Background(), Query{Generation: id}); !errors.Is(err, ErrExpired) {
		t.Fatal("old process cursor accepted")
	}
}

func TestBudgetsRejectPublicationAndCachedReadersDoNotCollect(t *testing.T) {
	var large atomic.Bool
	var calls atomic.Int32
	cfg := quietConfig()
	cfg.Limits.MaxRecords = 8
	m, _ := New(cfg, sourceFunc(func(context.Context, string) (inventory.Snapshot, error) {
		calls.Add(1)
		if large.Load() {
			return sample(time.Now(), 10), nil
		}
		return sample(time.Now(), 3), nil
	}))
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	id := m.State().CurrentID
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 100 {
				p, err := m.Datasets(context.Background(), Query{Generation: id, Limit: 2})
				if err != nil || p.Generation != id || len(p.Rows) != 2 {
					t.Error("cached read failed")
				}
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("queries spawned collection")
	}
	large.Store(true)
	if err := m.Refresh(); err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return m.State().LastAttempt.ErrorCode == "limit" })
	if m.State().CurrentID != id {
		t.Fatal("over-budget snapshot published")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Datasets(ctx, Query{}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled query did work")
	}
}

func TestQueryAdmissionAndConfiguration(t *testing.T) {
	cfg := quietConfig()
	cfg.MaxReaders = 1
	m, _ := New(cfg, sourceFunc(func(context.Context, string) (inventory.Snapshot, error) { return sample(time.Now(), 1), nil }))
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	m.readers <- struct{}{}
	if _, err := m.Datasets(context.Background(), Query{}); !errors.Is(err, ErrBusy) {
		t.Fatal("query admission not bounded")
	}
	<-m.readers
	for _, change := range []func(*Config){func(c *Config) { c.Interval = 0 }, func(c *Config) { c.Timeout = 0 }, func(c *Config) { c.StaleAfter = 0 }, func(c *Config) { c.Limits.MaxBytes = 0 }, func(c *Config) { c.MaxReaders = 65 }} {
		c := cfg
		change(&c)
		if _, err := New(c, m.source); !errors.Is(err, ErrConfig) {
			t.Fatal("unsafe collector configuration accepted")
		}
	}
	if _, err := New(cfg, nil); !errors.Is(err, ErrConfig) {
		t.Fatal("nil source accepted")
	}
	if err := m.Run(context.Background()); !errors.Is(err, ErrStarted) {
		t.Fatal("second worker permitted")
	}
}

func TestRetryKeepsPreviousFailureVisibleUntilSuccess(t *testing.T) {
	var mode atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	m, _ := New(quietConfig(), sourceFunc(func(ctx context.Context, _ string) (inventory.Snapshot, error) {
		switch mode.Load() {
		case 1:
			return inventory.Snapshot{}, helper.ErrProtocol
		case 2:
			entered <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
				return inventory.Snapshot{}, ctx.Err()
			}
		}
		return sample(time.Now(), 1), nil
	}))
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	mode.Store(1)
	_ = m.Refresh()
	await(t, func() bool { return m.State().LastAttempt.ErrorCode == "protocol" })
	mode.Store(2)
	_ = m.Refresh()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("retry not started")
	}
	if !m.State().Datasets.Stale {
		t.Fatal("retry erased previous failure before obtaining new data")
	}
	close(release)
	await(t, func() bool { return !m.State().Collecting })
	if m.State().Datasets.Stale {
		t.Fatal("successful recovery did not clear transport error")
	}
}

func TestHundredThousandRecordBoundary(t *testing.T) {
	var exhausted atomic.Bool
	m, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) {
		count := 99999
		if exhausted.Load() {
			count++
		}
		return sample(time.Now(), count), nil
	}))
	running(t, m)
	// The total cap includes the one pool as well as dataset rows.
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	id := m.State().CurrentID
	p, err := m.Datasets(context.Background(), Query{Generation: id, Offset: 99998, Limit: 500})
	if err != nil || p.Total != 99999 || len(p.Rows) != 1 || p.More {
		t.Fatal("record boundary truncated or overflowed")
	}
	exhausted.Store(true)
	if err := m.Refresh(); err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return m.State().LastCompleted.ErrorCode == "limit" && !m.State().Collecting })
	if m.State().CurrentID != id {
		t.Fatal("over-cap generation replaced prior context")
	}
}

func TestProviderAndOverviewResponsesAreDetached(t *testing.T) {
	s := sample(time.Now(), 0)
	s.GEOM.Data = inventory.GEOM{Classes: []inventory.Class{{ID: "c", Name: "DISK"}}, Geoms: []inventory.Geom{{ID: "g", ClassID: "c", Name: "ada0"}}, Providers: []inventory.Provider{{ID: "p", GeomID: "g", Name: "ada0", Class: "DISK", Kind: inventory.DiskCandidate, SectorSize: 512, Identity: inventory.IdentityUnique, Identifiers: []inventory.Identifier{{Kind: "serial", Value: "fixture", Source: "geom.config.ident"}}}}}
	m, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) { return s, nil }))
	running(t, m)
	await(t, func() bool { return m.State().CurrentID != "" && !m.State().Collecting })
	p, err := m.Providers(context.Background(), Query{Sort: "size"})
	if err != nil || len(p.Rows) != 1 {
		t.Fatal(err)
	}
	p.Rows[0].Identifiers[0].Value = "reader changed"
	overview, err := m.Overview(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	*overview.Host.PhysicalMemory = 0
	next, _ := m.Providers(context.Background(), Query{})
	view, _ := m.Overview(context.Background(), "")
	if next.Rows[0].Identifiers[0].Value != "fixture" || *view.Host.PhysicalMemory == 0 {
		t.Fatal("response retained mutable cache storage")
	}
	if _, err := m.Pools(context.Background(), Query{Sort: "free"}); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkCachedDatasetPage(b *testing.B) {
	for _, count := range []int{100, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			s := sample(time.Now(), count)
			m, _ := New(quietConfig(), sourceFunc(func(context.Context, string) (inventory.Snapshot, error) { return s, nil }))
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- m.Run(ctx) }()
			defer func() { cancel(); <-done }()
			deadline := time.Now().Add(10 * time.Second)
			for m.State().CurrentID == "" {
				if time.Now().After(deadline) {
					b.Fatal("benchmark collection failed")
				}
				time.Sleep(time.Millisecond)
			}
			id := m.State().CurrentID
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := m.Datasets(ctx, Query{Generation: id, Limit: 100}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
