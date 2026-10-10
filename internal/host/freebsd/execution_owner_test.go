// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func TestCommissionExecutionOwnerLifecycle(t *testing.T) {
	ctx := context.Background()
	r := newExecutionOwnerRegistry()
	roots := true
	created, observed, finished, released := 0, 0, 0, 0
	source := executionOwnerSource{observe: func(context.Context) (storage.ExecutionDomainsObservation, error) {
		observed++
		return storage.ExecutionDomainsObservation{}, nil
	}, finish: func(context.Context) error { finished++; return nil }, release: func() error { released++; return nil }}
	factory := func(context.Context) (executionOwnerSource, error) { created++; return source, nil }
	root := func() bool { return roots }
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for _, c := range []context.Context{nil, cancelled} {
		if o, e := commissionExecutionOwner(c, r, root, factory); o != nil || e == nil || created != 0 || r.claimed {
			t.Fatal("invalid context consumed commissioning")
		}
	}
	roots = false
	if o, e := commissionExecutionOwner(ctx, r, root, factory); o != nil || e == nil || created != 0 || r.claimed {
		t.Fatal("nonroot commissioning")
	}
	roots = true
	for _, q := range []*executionOwnerRegistry{nil, {}, {original: r}} {
		if o, e := commissionExecutionOwner(ctx, q, root, factory); o != nil || e == nil || created != 0 {
			t.Fatal("foreign registry")
		}
	}
	if o, e := commissionExecutionOwner(ctx, r, nil, factory); o != nil || e == nil || r.claimed {
		t.Fatal("missing root")
	}
	if o, e := commissionExecutionOwner(ctx, r, root, nil); o != nil || e == nil || r.claimed {
		t.Fatal("missing factory")
	}
	o, e := commissionExecutionOwner(ctx, r, root, factory)
	if e != nil || o == nil || created != 1 || r.owner != o {
		t.Fatal("original commissioning", e)
	}
	copied := &commissionedExecutionOwner{original: o, registry: r, source: source, root: root}
	wire, e := json.Marshal(o)
	var decoded commissionedExecutionOwner
	if e != nil || string(wire) != "{}" || json.Unmarshal(wire, &decoded) != nil {
		t.Fatal("serialized owner")
	}
	for _, q := range []*commissionedExecutionOwner{nil, {}, copied, &decoded} {
		if _, e := q.Observe(ctx); e == nil || q.FinishIdle(ctx) == nil || q.Close() == nil || observed != 0 || finished != 0 || released != 0 {
			t.Fatal("foreign owner reached original")
		}
	}
	for _, c := range []context.Context{nil, cancelled} {
		if _, e := o.Observe(c); e == nil || o.FinishIdle(c) == nil || observed != 0 || finished != 0 {
			t.Fatal("preentry context reached producer")
		}
	}
	if _, e = o.Observe(ctx); e != nil || observed != 1 {
		t.Fatal("original observation", e)
	}
	if q, e := commissionExecutionOwner(ctx, r, root, factory); q != nil || e == nil || created != 1 {
		t.Fatal("second commissioning reached producer")
	}
	if o.FinishIdle(ctx) != nil || finished != 1 {
		t.Fatal("idle finish")
	}
	if _, e = o.Observe(ctx); e == nil || o.FinishIdle(ctx) == nil || observed != 1 || finished != 1 {
		t.Fatal("finished owner reused")
	}
	if o.Close() != nil || o.Close() != nil || released != 1 {
		t.Fatal("release")
	}
	if q, e := commissionExecutionOwner(ctx, r, root, factory); q != nil || e == nil || created != 1 {
		t.Fatal("closed owner reset registration")
	}
}

func TestCommissionExecutionOwnerFailures(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"factory", "partial", "cancel", "root-after", "observation", "finish", "release", "root-live", "root-close"} {
		t.Run(mode, func(t *testing.T) {
			r := newExecutionOwnerRegistry()
			rooted := true
			made, observed, finished, released := 0, 0, 0, 0
			c, cancel := context.WithCancel(ctx)
			defer cancel()
			source := executionOwnerSource{observe: func(context.Context) (storage.ExecutionDomainsObservation, error) {
				observed++
				if mode == "observation" {
					return storage.ExecutionDomainsObservation{}, storage.ErrDrift
				}
				return storage.ExecutionDomainsObservation{}, nil
			}, finish: func(context.Context) error {
				finished++
				if mode == "finish" {
					return storage.ErrStateUncertain
				}
				return nil
			}, release: func() error {
				released++
				if mode == "release" {
					return storage.ErrUnavailable
				}
				return nil
			}}
			factory := func(context.Context) (executionOwnerSource, error) {
				made++
				switch mode {
				case "factory":
					return source, storage.ErrUnavailable
				case "partial":
					return executionOwnerSource{release: source.release}, nil
				case "cancel":
					cancel()
				case "root-after":
					rooted = false
				}
				return source, nil
			}
			root := func() bool { return rooted }
			o, e := commissionExecutionOwner(c, r, root, factory)
			if mode == "factory" || mode == "partial" || mode == "cancel" || mode == "root-after" {
				if o != nil || e == nil || released != 1 || !r.claimed {
					t.Fatal("failed factory released registration", e)
				}
			} else {
				if e != nil || o == nil {
					t.Fatal(e)
				}
				switch mode {
				case "observation":
					if _, e = o.Observe(ctx); e == nil || observed != 1 {
						t.Fatal("observation failure")
					}
				case "finish":
					if o.FinishIdle(ctx) == nil || finished != 1 {
						t.Fatal("finish failure")
					}
				case "root-live":
					rooted = false
					if _, e = o.Observe(ctx); e == nil || observed != 0 {
						t.Fatal("lost root reached source")
					}
					rooted = true
				case "release":
					if o.FinishIdle(ctx) != nil {
						t.Fatal("finish")
					}
				case "root-close":
					rooted = false
				}
				if mode != "release" && mode != "root-close" {
					if _, e = o.Observe(ctx); e == nil || o.FinishIdle(ctx) == nil {
						t.Fatal("failed owner revived")
					}
				}
				first := o.Close()
				second := o.Close()
				if !errors.Is(second, first) || released != 1 {
					t.Fatal("release repeated", first, second)
				}
				if (mode == "release" || mode == "root-close") && first == nil {
					t.Fatal("lost release/root reported success")
				}
			}
			if q, e := commissionExecutionOwner(ctx, r, func() bool { return true }, factory); q != nil || e == nil || made != 1 {
				t.Fatal("failed registration reset")
			}
		})
	}
}

func TestCommissionExecutionOwnerContention(t *testing.T) {
	ctx := context.Background()
	r := newExecutionOwnerRegistry()
	entered, resume := make(chan struct{}), make(chan struct{})
	var created atomic.Int32
	factory := func(context.Context) (executionOwnerSource, error) {
		created.Add(1)
		close(entered)
		<-resume
		return executionOwnerSource{observe: func(context.Context) (storage.ExecutionDomainsObservation, error) {
			return storage.ExecutionDomainsObservation{}, nil
		}, finish: func(context.Context) error { return nil }, release: func() error { return nil }}, nil
	}
	result := make(chan *commissionedExecutionOwner, 1)
	go func() { o, _ := commissionExecutionOwner(ctx, r, func() bool { return true }, factory); result <- o }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("factory not entered")
	}
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if o, e := commissionExecutionOwner(ctx, r, func() bool { return true }, factory); o != nil || e == nil {
				t.Error("competing commissioning")
			}
		}()
	}
	wg.Wait()
	close(resume)
	o := <-result
	if o == nil || created.Load() != 1 || o.Close() != nil {
		t.Fatal("original factory/release")
	}
}
func TestCommissionExecutionOwnerExclusion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r := newExecutionOwnerRegistry()
	entered, resume := make(chan struct{}), make(chan struct{})
	source := executionOwnerSource{observe: func(context.Context) (storage.ExecutionDomainsObservation, error) {
		t.Error("finished observation reached source")
		return storage.ExecutionDomainsObservation{}, nil
	}, finish: func(context.Context) error {
		close(entered)
		select {
		case <-resume:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, release: func() error { return nil }}
	o, e := commissionExecutionOwner(ctx, r, func() bool { return true }, func(context.Context) (executionOwnerSource, error) { return source, nil })
	if e != nil {
		t.Fatal(e)
	}
	finished := make(chan error, 1)
	go func() { finished <- o.FinishIdle(ctx) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("finish not entered")
	}
	closed := make(chan error, 1)
	go func() { closed <- o.Close() }()
	observed := make(chan error, 1)
	go func() { _, e := o.Observe(ctx); observed <- e }()
	early := false
	select {
	case <-closed:
		early = true
	case <-time.After(30 * time.Millisecond):
	}
	close(resume)
	if early {
		t.Fatal("close overlapped finish")
	}
	if <-finished != nil || <-closed != nil || <-observed == nil {
		t.Fatal("finish/observe/release ordering")
	}
}
