// SPDX-License-Identifier: BSD-3-Clause
// Package observation owns volatile, read-only inventory generations.
package observation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/inventory"
)

var (
	ErrBusy        = errors.New("inventory work already active")
	ErrNotRunning  = errors.New("inventory collector is not running")
	ErrStarted     = errors.New("collector already started; construct a new manager")
	ErrExpired     = errors.New("inventory generation expired; reload required")
	ErrUnavailable = errors.New("inventory has not been collected")
	ErrQuery       = errors.New("invalid inventory query")
	ErrConfig      = errors.New("invalid collector configuration")
)

// Source must honor cancellation and transfer a stable result until Collect's
// caller has copied it. The FreeBSD helper Client satisfies this contract.
type Source interface {
	Collect(context.Context, string) (inventory.Snapshot, error)
}
type Config struct {
	Interval, Timeout, StaleAfter time.Duration
	Limits                        inventory.Limits
	MaxReaders                    int
}

func DefaultConfig() Config {
	return Config{30 * time.Second, 45 * time.Second, 90 * time.Second, inventory.DefaultLimits(), 16}
}
func (c Config) validate() error {
	if c.Interval <= 0 || c.Timeout <= 0 || c.Timeout > 5*time.Minute || c.StaleAfter <= 0 || c.MaxReaders < 1 || c.MaxReaders > 64 || c.Limits.Validate() != nil {
		return ErrConfig
	}
	return nil
}

type Attempt struct {
	ID                    string
	StartedAt, FinishedAt time.Time
	ErrorCode             string
}

// Result exposes only completed attempt metadata and section status to an
// operational observer. No raw source error, identifier or inventory is passed.
type Result struct {
	Attempt Attempt
	Status  [4]inventory.Status
}
type Freshness struct {
	Status                                  inventory.Status
	StartedAt, FinishedAt, LastSuccessfulAt time.Time
	Problems                                []inventory.Problem
	Stale                                   bool
}
type State struct {
	Running, Collecting         bool
	CurrentID, CompleteID       string
	LastAttempt                 Attempt
	LastCompleted               Attempt
	Host, GEOM, Pools, Datasets Freshness
}
type Query struct {
	Generation, Filter, Sort string
	Descending               bool
	Offset, Limit            int
}
type Page[T any] struct {
	Generation    string
	Total, Offset int
	More          bool
	Rows          []T
	Freshness     Freshness
}
type generation struct {
	id       string
	snapshot inventory.Snapshot
	success  [4]time.Time
}
type Manager struct {
	source                       Source
	config                       Config
	now                          func() time.Time
	nonce                        string
	mu                           sync.RWMutex
	started, running, collecting bool
	serial                       uint64
	last                         Attempt
	completed                    Attempt
	current, previous            *generation
	successes                    [4]time.Time
	requests                     chan struct{}
	readers                      chan struct{}
	observer                     func(Result)
}

func New(c Config, s Source) (*Manager, error) {
	return NewObserved(c, s, nil)
}

// NewObserved requires a nonblocking observer; it runs after publication,
// outside the state lock. Runtime event logging supplies a bounded queue.
func NewObserved(c Config, s Source, observer func(Result)) (*Manager, error) {
	if c.validate() != nil || s == nil {
		return nil, ErrConfig
	}
	var seed [16]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return nil, ErrConfig
	}
	return &Manager{source: s, config: c, now: time.Now, nonce: hex.EncodeToString(seed[:]), requests: make(chan struct{}, 1), readers: make(chan struct{}, c.MaxReaders), observer: observer}, nil
}

// Run is the only collector worker. Restart means a new Manager with a new ID
// namespace and no inventory; Run never retries or persists an old generation.
func (m *Manager) Run(ctx context.Context) error {
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()
	return m.run(ctx, ticker.C)
}
func (m *Manager) run(ctx context.Context, ticks <-chan time.Time) error {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return ErrStarted
	}
	if ctx.Err() != nil {
		m.mu.Unlock()
		return ctx.Err()
	}
	m.started = true
	m.running = true
	m.collecting = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.collecting = false
		m.current = nil
		m.previous = nil
		m.successes = [4]time.Time{}
		m.mu.Unlock()
	}()
	m.collect(ctx)
	// Scheduled ticks that arrived while busy are duplicates, not queued work.
	drainTicks(ticks)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticks:
			m.mu.Lock()
			if m.collecting {
				m.mu.Unlock()
				continue
			}
			m.collecting = true
			m.mu.Unlock()
		case <-m.requests:
			// Refresh already reserved collecting=true under the publication lock.
		}
		m.collect(ctx)
		drainTicks(ticks)
	}
}
func drainTicks(ticks <-chan time.Time) {
	// A real ticker retains at most one pending event. Drain only that event;
	// even an extremely short configured interval must not keep us spinning.
	select {
	case <-ticks:
	default:
	}
}

// Refresh reserves at most one immediate operation. Overlapping manual/scheduled
// refreshes fail busy; they never queue another native request.
func (m *Manager) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return ErrNotRunning
	}
	if m.collecting {
		return ErrBusy
	}
	m.collecting = true
	m.requests <- struct{}{}
	return nil
}
func (m *Manager) collect(parent context.Context) {
	m.mu.Lock()
	m.serial++
	id := m.nonce + "-" + strconv.FormatUint(m.serial, 10)
	m.last = Attempt{ID: id, StartedAt: m.now().UTC()}
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(parent, m.config.Timeout)
	defer cancel()
	s, err := m.source.Collect(ctx, id)
	if err == nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		} else {
			s, err = inventory.CloneBounded(s, m.config.Limits)
		}
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	m.mu.Lock()
	defer func() {
		r := Result{Attempt: m.completed}
		if err == nil {
			r.Status = [4]inventory.Status{s.Host.Status, s.GEOM.Status, s.Pools.Status, s.Datasets.Status}
		}
		m.mu.Unlock()
		if m.observer != nil {
			m.observer(r)
		}
	}()
	// Queries can delay this lock after collection/clone has finished. Recheck
	// cancellation at the publication boundary, not only before waiting.
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	m.last.FinishedAt = m.now().UTC()
	m.last.ErrorCode = errorCode(err)
	m.completed = m.last
	if err == nil {
		statuses := []inventory.Status{s.Host.Status, s.GEOM.Status, s.Pools.Status, s.Datasets.Status}
		finished := []time.Time{s.Host.FinishedAt, s.GEOM.FinishedAt, s.Pools.FinishedAt, s.Datasets.FinishedAt}
		for i, status := range statuses {
			if status == inventory.OK {
				m.successes[i] = finished[i]
			}
		}
		next := &generation{id: id, snapshot: s, success: m.successes}
		// Keep the latest and last complete. Without a complete observation yet,
		// retain the previous latest. Publication never combines section arrays.
		retained := m.current
		if !s.Complete() {
			if m.current != nil && m.current.snapshot.Complete() {
				retained = m.current
			} else if m.previous != nil && m.previous.snapshot.Complete() {
				retained = m.previous
			}
		}
		m.previous = retained
		m.current = next
	}
	m.collecting = false
}
func errorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, inventory.ErrLimit), errors.Is(err, helper.ErrLimit):
		return "limit"
	case errors.Is(err, inventory.ErrMalformed):
		return "malformed"
	case errors.Is(err, helper.ErrProtocol):
		return "protocol"
	case errors.Is(err, helper.ErrPermission):
		return "permission"
	case errors.Is(err, helper.ErrConfig):
		return "configuration"
	case errors.Is(err, helper.ErrUnsupported):
		return "unsupported"
	default:
		return "failed"
	}
}
func (m *Manager) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := State{Running: m.running, Collecting: m.collecting, LastAttempt: m.last, LastCompleted: m.completed}
	if m.current == nil {
		return out
	}
	out.CurrentID = m.current.id
	if m.current.snapshot.Complete() {
		out.CompleteID = m.current.id
	} else if m.previous != nil && m.previous.snapshot.Complete() {
		out.CompleteID = m.previous.id
	}
	out.Host = m.freshness(m.current, 0)
	out.GEOM = m.freshness(m.current, 1)
	out.Pools = m.freshness(m.current, 2)
	out.Datasets = m.freshness(m.current, 3)
	return out
}
func (m *Manager) freshness(g *generation, index int) Freshness {
	var f Freshness
	s := g.snapshot
	switch index {
	case 0:
		f = sectionFreshness(s.Host)
	case 1:
		f = sectionFreshness(s.GEOM)
	case 2:
		f = sectionFreshness(s.Pools)
	case 3:
		f = sectionFreshness(s.Datasets)
	}
	f.LastSuccessfulAt = g.success[index]
	f.Stale = g != m.current || m.completed.ErrorCode != "" || f.Status != inventory.OK || f.LastSuccessfulAt.IsZero() || m.now().Sub(f.LastSuccessfulAt) >= m.config.StaleAfter
	return f
}
func sectionFreshness[T any](s inventory.Section[T]) Freshness {
	return Freshness{Status: s.Status, StartedAt: s.StartedAt, FinishedAt: s.FinishedAt, Problems: append([]inventory.Problem(nil), s.Problems...)}
}
func (m *Manager) resolve(id string) (*generation, error) {
	if id == "" {
		if m.current == nil {
			return nil, ErrUnavailable
		}
		return m.current, nil
	}
	if m.current != nil && m.current.id == id {
		return m.current, nil
	}
	if m.previous != nil && m.previous.id == id {
		return m.previous, nil
	}
	return nil, ErrExpired
}

// Overview is small, detached metadata; it never exports the cached snapshot.
type Overview struct {
	Generation                                         string
	StartedAt, FinishedAt                              time.Time
	Complete                                           bool
	Host                                               inventory.Host
	HostState, GEOM, Pools, Datasets                   Freshness
	Classes, Geoms, Providers, PoolCount, DatasetCount int
}

func (m *Manager) Overview(ctx context.Context, id string) (Overview, error) {
	if err := ctx.Err(); err != nil {
		return Overview{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, err := m.resolve(id)
	if err != nil {
		return Overview{}, err
	}
	s := g.snapshot
	h := s.Host.Data
	if h.PhysicalMemory != nil {
		n := *h.PhysicalMemory
		h.PhysicalMemory = &n
	}
	return Overview{Generation: g.id, StartedAt: s.StartedAt, FinishedAt: s.FinishedAt, Complete: s.Complete(), Host: h, HostState: m.freshness(g, 0), GEOM: m.freshness(g, 1), Pools: m.freshness(g, 2), Datasets: m.freshness(g, 3), Classes: len(s.GEOM.Data.Classes), Geoms: len(s.GEOM.Data.Geoms), Providers: len(s.GEOM.Data.Providers), PoolCount: len(s.Pools.Data), DatasetCount: len(s.Datasets.Data)}, nil
}
