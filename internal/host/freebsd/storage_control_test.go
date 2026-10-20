// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

type controlSelectionModel struct {
	checks, freshChecks, closes int
	check                       func(context.Context) error
	closeErr                    error
	freshErr                    error
	onClose                     func()
}

func (s *controlSelectionModel) Targets() (storage.ControlTargets, error) {
	return storage.ControlTargets{JournalDirectory: "/var/db/journal", LedgerDirectory: "/var/db/ledger"}, nil
}
func (s *controlSelectionModel) Check(c context.Context) error {
	s.checks++
	if s.check != nil {
		return s.check(c)
	}
	return c.Err()
}
func (s *controlSelectionModel) CheckFresh(c context.Context) error {
	s.freshChecks++
	return errors.Join(s.freshErr, c.Err())
}
func (s *controlSelectionModel) Close() error {
	s.closes++
	if s.onClose != nil {
		s.onClose()
	}
	return s.closeErr
}

type controlModel struct {
	selection                                                   *controlSelectionModel
	root                                                        atomic.Bool
	registry                                                    *executionOwnerRegistry
	loads, fresh, start, inspects, observes, finishes, releases int
	observe, finish                                             func(context.Context) error
	releaseErr                                                  error
	release                                                     func() error
	inspectState                                                string
	inspectErr                                                  error
	inspectReleased                                             bool
}

func newControlModel() *controlModel {
	m := &controlModel{selection: &controlSelectionModel{}, registry: newExecutionOwnerRegistry(), inspectState: "closed", inspectReleased: true}
	m.root.Store(true)
	return m
}
func (m *controlModel) sources() storageControlSources {
	own := func(c context.Context) controlOwnerAttempt {
		entered := false
		o, e := commissionExecutionOwner(c, m.registry, m.root.Load, func(context.Context) (executionOwnerSource, error) {
			entered = true
			return executionOwnerSource{
				observe: func(c context.Context) (storage.ExecutionDomainsObservation, error) {
					m.observes++
					var e error
					if m.observe != nil {
						e = m.observe(c)
					}
					return storage.ExecutionDomainsObservation{}, e
				},
				finish: func(c context.Context) error {
					m.finishes++
					if m.finish != nil {
						return m.finish(c)
					}
					return c.Err()
				},
				release: func() error {
					m.releases++
					if m.release != nil {
						return m.release()
					}
					return m.releaseErr
				},
			}, nil
		})
		return controlOwnerAttempt{owner: o, entered: entered, released: !entered, err: e}
	}
	return storageControlSources{supported: true, root: m.root.Load,
		load:  func(context.Context, string) (controlSelection, error) { m.loads++; return m.selection, nil },
		fresh: func(c context.Context, _ storage.ControlTargets) controlOwnerAttempt { m.fresh++; return own(c) },
		start: func(c context.Context, _ storage.ControlTargets) controlOwnerAttempt { m.start++; return own(c) },
		inspect: func(context.Context, storage.ControlTargets) controlInspectionAttempt {
			m.inspects++
			return controlInspectionAttempt{state: m.inspectState, released: m.inspectReleased, err: m.inspectErr}
		},
	}
}
func controlArgs(action string) []string {
	return []string{action, "--config", "/etc/horvium/storage-control.json"}
}

const controlClosedLiteral = "{\"format\":\"horvium-storage-control-result\",\"version\":1,\"action\":\"commission-fresh\",\"state\":\"closed\",\"writes_enabled\":false}\n"

func TestControlArgumentsAndPreentry(t *testing.T) {
	for _, action := range []string{"commission-fresh", "start-closed", "inspect"} {
		a, p, e := parseStorageControl(controlArgs(action))
		if e != nil || a != action || p != "/etc/horvium/storage-control.json" {
			t.Fatal("exact invocation rejected")
		}
	}
	bad := [][]string{nil, {"inspect"}, {"ins", "--config", "/etc/config"}, {"inspect", "--config=/etc/config"}, {"inspect", "-c", "/etc/config"}, {"inspect", "--config", "relative"}, {"inspect", "--config", "/etc//config"}, {"inspect", "--config", "/etc/config", "--retry"}, {"inspect", "--config", "/etc/config", "--config", "/etc/config"}, {"inspect", "--config", "/etc/config", "secret"}}
	for _, args := range bad {
		m := newControlModel()
		r := executeStorageControl(context.Background(), args, m.sources())
		if r.exit != 2 || r.class != "invalid-input" || m.loads+m.fresh+m.start+m.inspects != 0 {
			t.Fatalf("invalid invocation reached source: %#v", r)
		}
	}
	for _, kind := range []string{"cancelled", "not-root", "unsupported"} {
		m := newControlModel()
		sources := m.sources()
		c, cancel := context.WithCancel(context.Background())
		switch kind {
		case "cancelled":
			cancel()
		case "not-root":
			m.root.Store(false)
		case "unsupported":
			sources.supported = false
		}
		r := executeStorageControl(c, controlArgs("inspect"), sources)
		cancel()
		if r.exit != 2 || m.loads != 0 || m.inspects != 0 {
			t.Fatal(kind, r)
		}
	}
}

func TestControlFiniteAndBlockedOutput(t *testing.T) {
	m := newControlModel()
	r := executeStorageControl(context.Background(), controlArgs("commission-fresh"), m.sources())
	if r.exit != 0 || r.state != "closed" || m.fresh != 1 || m.start+m.inspects != 0 || m.observes != 1 || m.finishes != 1 || m.releases != 1 || m.selection.closes != 1 || m.selection.freshChecks != 1 {
		t.Fatalf("incomplete finite lifecycle: %#v %#v", r, m)
	}
	entered, unblock, done := make(chan struct{}), make(chan struct{}), make(chan int, 1)
	w := writerFunc(func(data []byte) (int, error) {
		if string(data) != controlClosedLiteral {
			t.Error("wrong literal result", string(data))
		}
		if m.releases != 1 || m.finishes != 1 || m.selection.closes != 1 {
			t.Error("output retained original ownership")
		}
		close(entered)
		<-unblock
		return len(data), nil
	})
	go func() { done <- deliverStorageControl(r, w, io.Discard) }()
	<-entered
	select {
	case <-done:
		t.Fatal("blocked sink reported delivery")
	default:
	}
	close(unblock)
	if <-done != 0 {
		t.Fatal("complete delivery failed")
	}
	// Release never resets the original registry, even for another path attempt.
	second := executeStorageControl(context.Background(), controlArgs("commission-fresh"), m.sources())
	if second.exit != 2 || m.observes != 1 || m.finishes != 1 || m.releases != 1 {
		t.Fatal("second attempt reconstructed authority", second)
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(b []byte) (int, error) { return f(b) }

func TestControlForegroundFreshCleanupBudget(t *testing.T) {
	m := newControlModel()
	active, finishing, releaseFinish := make(chan struct{}), make(chan struct{}), make(chan struct{})
	type marker struct{}
	parent, cancel := context.WithCancel(context.WithValue(context.Background(), marker{}, "foreground"))
	var observed atomic.Bool
	foreground := &controlWaitContext{Context: parent, observed: &observed, ready: active}
	defer cancel()
	m.observe = func(context.Context) error { observed.Store(true); return nil }
	var firstDeadline time.Time
	m.finish = func(c context.Context) error {
		if c.Err() != nil || c.Value(marker{}) != nil {
			t.Error("cleanup reused cancelled foreground context")
		}
		deadline, ok := c.Deadline()
		firstDeadline = deadline
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > controlOperationLimit {
			t.Error("cleanup deadline widened or missing")
		}
		close(finishing)
		<-releaseFinish
		again, _ := c.Deadline()
		if again != firstDeadline || c.Err() != nil {
			t.Error("later cancellation changed cleanup budget")
		}
		return nil
	}
	done := make(chan storageControlOutcome, 1)
	go func() { done <- executeStorageControl(foreground, controlArgs("start-closed"), m.sources()) }()
	<-active
	select {
	case <-done:
		t.Fatal("foreground emitted before cancellation")
	default:
	}
	cancel()
	<-finishing
	cancel()
	close(releaseFinish)
	r := <-done
	if r.exit != 0 || r.state != "closed" || m.start != 1 || m.fresh+m.inspects != 0 || m.finishes != 1 || m.releases != 1 || m.selection.closes != 1 {
		t.Fatalf("foreground cleanup %#v %#v", r, m)
	}
	var out, err bytes.Buffer
	if deliverStorageControl(r, &out, &err) != 0 || err.Len() != 0 || !strings.Contains(out.String(), `"action":"start-closed"`) {
		t.Fatal("wrong final foreground result")
	}
}

// Expose the actual foreground Done boundary after original observation. This
// prevents a timing sleep from cancelling during the separate admission guard.
type controlWaitContext struct {
	context.Context
	observed *atomic.Bool
	ready    chan struct{}
	once     sync.Once
}

func (c *controlWaitContext) Done() <-chan struct{} {
	done := c.Context.Done()
	if c.observed.Load() {
		c.once.Do(func() { close(c.ready) })
	}
	return done
}

func TestControlFailureRetirementAndOutputBarrier(t *testing.T) {
	sentinel := errors.New("secret /operator/config native-raw-sentinel")
	for _, kind := range []string{"observe", "finish", "custody", "root-loss", "release", "selection-release", "timeout"} {
		t.Run(kind, func(t *testing.T) {
			m := newControlModel()
			switch kind {
			case "observe":
				m.observe = func(context.Context) error { return sentinel }
			case "finish":
				m.finish = func(context.Context) error { return sentinel }
			case "timeout":
				m.finish = func(context.Context) error { return context.DeadlineExceeded }
			case "custody":
				m.observe = func(context.Context) error {
					m.selection.check = func(context.Context) error { return storage.ErrDrift }
					return nil
				}
			case "root-loss":
				m.observe = func(context.Context) error { m.root.Store(false); return nil }
			case "release":
				m.releaseErr = sentinel
			case "selection-release":
				m.selection.closeErr = sentinel
			}
			r := executeStorageControl(context.Background(), controlArgs("commission-fresh"), m.sources())
			if r.exit != 3 || r.state != "" || m.releases != 1 || m.selection.closes != 1 {
				t.Fatalf("failed lifetime did not retire/release %#v %#v", r, m)
			}
			if (kind == "observe" || kind == "custody" || kind == "root-loss") && m.finishes != 0 {
				t.Fatal("uncertain observation entered finish")
			}
			var out, err bytes.Buffer
			if deliverStorageControl(r, &out, &err) != 3 || out.Len() != 0 || strings.Contains(err.String(), "sentinel") {
				t.Fatal("failure escaped output boundary")
			}
			uncertain := kind == "release" || kind == "selection-release" || kind == "root-loss"
			if uncertain && err.Len() != 0 {
				t.Fatal("unconfirmed release invoked a sink")
			}
			if !uncertain && err.String() != "recovery-required\n" {
				t.Fatal("confirmed release lost fixed recovery class", err.String())
			}
		})
	}
}

func TestControlInspectionIsolationAndFailureClasses(t *testing.T) {
	for _, kind := range []string{"closed", "recovery-required", "busy", "missing", "uncertain-release"} {
		m := newControlModel()
		switch kind {
		case "recovery-required":
			m.inspectState = kind
		case "busy":
			m.inspectErr = storage.ErrStateLocked
		case "missing":
			m.inspectErr = storage.ErrUnavailable
		case "uncertain-release":
			m.inspectReleased = false
			m.inspectErr = storage.ErrStateUncertain
		}
		r := executeStorageControl(context.Background(), controlArgs("inspect"), m.sources())
		if m.inspects != 1 || m.fresh+m.start+m.observes+m.finishes+m.releases != 0 || m.selection.closes != 1 || m.registry.claimed {
			t.Fatal("inspection reached writer lifecycle", kind)
		}
		var out, err bytes.Buffer
		code := deliverStorageControl(r, &out, &err)
		switch kind {
		case "closed":
			if code != 0 || !strings.Contains(out.String(), `"state":"closed"`) {
				t.Fatal(kind, r)
			}
		case "recovery-required":
			if code != 3 || !strings.Contains(out.String(), `"state":"recovery-required"`) {
				t.Fatal(kind, r)
			}
		case "busy":
			if code != 2 || out.Len() != 0 || err.String() != "busy\n" {
				t.Fatal(kind, r)
			}
		case "missing":
			if code != 2 || out.Len() != 0 || err.String() != "unavailable\n" {
				t.Fatal(kind, r)
			}
		case "uncertain-release":
			if code != 3 || out.Len()+err.Len() != 0 {
				t.Fatal("unconfirmed inspection release invoked sink")
			}
		}
	}
}

func TestControlFactoryFailureAndShortOutput(t *testing.T) {
	for _, entered := range []bool{false, true} {
		for _, released := range []bool{false, true} {
			m := newControlModel()
			s := m.sources()
			s.fresh = func(context.Context, storage.ControlTargets) controlOwnerAttempt {
				return controlOwnerAttempt{entered: entered, released: released, err: errors.New("private-native-output")}
			}
			r := executeStorageControl(context.Background(), controlArgs("commission-fresh"), s)
			if r.deliverable != released || m.selection.closes != 1 || (entered && r.exit != 3) || (!entered && released && r.exit != 2) {
				t.Fatal("factory outcome lost original entry/release", r)
			}
		}
	}
	m := newControlModel()
	r := executeStorageControl(context.Background(), controlArgs("commission-fresh"), m.sources())
	calls := 0
	var err bytes.Buffer
	short := writerFunc(func(b []byte) (int, error) {
		calls++
		if m.releases != 1 || m.selection.closes != 1 {
			t.Error("write preceded release")
		}
		return len(b) - 1, nil
	})
	if deliverStorageControl(r, short, &err) != 3 || calls != 1 || err.String() != "recovery-required\n" {
		t.Fatal("short write succeeded or retried")
	}
}

func TestControlNativeUnavailableBeforeConfig(t *testing.T) {
	if nativeSupported {
		t.Skip("portable unsupported-platform contract")
	}
	var out, err bytes.Buffer
	if runStorageControl(context.Background(), controlArgs("commission-fresh"), &out, &err) != 2 || out.Len() != 0 || err.String() != "unavailable\n" {
		t.Fatal("unsupported platform reached config or authority")
	}
}

func TestControlFreshRefusalHasNoFallback(t *testing.T) {
	for _, role := range []string{"journal", "ledger"} {
		m := newControlModel()
		m.selection.freshErr = storage.ErrDenied
		r := executeStorageControl(context.Background(), controlArgs("commission-fresh"), m.sources())
		if r.exit != 2 || m.fresh+m.start+m.inspects != 0 || m.registry.claimed || m.selection.closes != 1 {
			t.Fatal(role, "existing leaf reached original entry", r)
		}
	}
}

func TestControlFinalReleaseProvenance(t *testing.T) {
	for _, kind := range []string{"owner-custody", "owner-root", "selection-root"} {
		t.Run(kind, func(t *testing.T) {
			m := newControlModel()
			switch kind {
			case "owner-custody":
				m.release = func() error { m.selection.check = func(context.Context) error { return storage.ErrDrift }; return nil }
			case "owner-root":
				m.release = func() error { m.root.Store(false); return nil }
			case "selection-root":
				m.selection.onClose = func() { m.root.Store(false) }
			}
			r := executeStorageControl(context.Background(), controlArgs("commission-fresh"), m.sources())
			var out, err bytes.Buffer
			if r.exit != 3 || r.state != "" || m.releases != 1 || m.selection.closes != 1 || deliverStorageControl(r, &out, &err) != 3 || out.Len() != 0 {
				t.Fatal("release drift retained closed result", r)
			}
			if kind == "owner-custody" {
				if err.String() != "recovery-required\n" {
					t.Fatal("confirmed release lost recovery class")
				}
			} else if err.Len() != 0 {
				t.Fatal("root-uncertain release called a sink")
			}
		})
	}
}
