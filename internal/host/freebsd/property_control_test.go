// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func controlFixture(t testing.TB) (*propertyControl, storage.PropertyIntent, *storage.PropertyJob, *storage.Policy, *storage.Observation, *time.Time, *int) {
	t.Helper()
	s := policySource(func(string) (storage.Policy, error) {
		p := collectionPolicy()
		p.Scopes[0].Properties = []storage.Property{storage.Quota, storage.RefQuota, storage.Compression}
		return p, nil
	})
	p, o, err := s.Collect(context.Background(), "100")
	if err != nil {
		t.Fatal(err)
	}
	j := sourcePropertyJob(t, p, o)
	j, err = storage.TransitionPropertyJob(j, storage.JobApplying, j.Job().History[0].At+1, "launch-intent", "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(int64(j.Job().History[1].At)).Add(time.Second)
	calls := 0
	c := &propertyControl{root: func() bool { return true }, clock: func() time.Time { return now }, authorize: func(context.Context, storage.PropertyIntent) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
		return j, p, o, nil
	}, run: func(_ context.Context, intent storage.PropertyIntent) error {
		args, err := propertySetArguments(intent)
		if err != nil {
			return err
		}
		calls++
		if !reflect.DeepEqual(args, []string{"set", "quota=2097152", "tank"}) {
			t.Fatal("arbitrary native arguments", args)
		}
		return nil
	}}
	return c, j.Intent(), &j, &p, &o, &now, &calls
}

func TestPropertyControlAllowlistAndBinding(t *testing.T) {
	c, i, _, _, o, _, calls := controlFixture(t)
	before, err := storage.EncodePropertyIntent(i)
	if err != nil {
		t.Fatal(err)
	}
	quiet, err := c.apply(context.Background(), i)
	if err != nil || !quiet || *calls != 1 || o.Eligible {
		t.Fatal("missing fixed control or changed eligibility", err)
	}
	after, _ := storage.EncodePropertyIntent(i)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("intent mutated")
	}
	quiet, err = c.apply(context.Background(), i)
	if !errors.Is(err, storage.ErrDenied) || quiet || *calls != 1 {
		t.Fatal("control replay", err)
	}
}

func TestPropertyControlDenials(t *testing.T) {
	for _, mode := range []string{"root", "authorize", "queued", "foreign-intent", "policy", "native", "incomplete", "expired", "rollback", "cancelled", "future-history", "missing-run", "missing-authorizer"} {
		t.Run(mode, func(t *testing.T) {
			c, i, j, p, o, now, calls := controlFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			original := c.authorize
			switch mode {
			case "root":
				c.root = func() bool { return false }
			case "authorize":
				c.authorize = func(context.Context, storage.PropertyIntent) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
					return storage.PropertyJob{}, storage.Policy{}, storage.Observation{}, ErrUnavailable
				}
			case "queued":
				*j = sourcePropertyJob(t, *p, *o)
			case "foreign-intent":
				i = storage.PropertyIntent{}
			case "policy":
				p.Revision++
			case "native":
				o.Filesystems[0].Properties.Quota.Effective = "1"
			case "incomplete":
				o.Complete = false
			case "expired":
				*now = now.Add(5 * time.Minute)
			case "rollback", "cancelled", "future-history":
				c.authorize = func(ctx context.Context, i storage.PropertyIntent) (storage.PropertyJob, storage.Policy, storage.Observation, error) {
					if mode == "cancelled" {
						cancel()
					} else if mode == "rollback" {
						*now = now.Add(-time.Second)
					} else {
						*now = time.UnixMilli(int64(j.Job().History[1].At) - 1)
					}
					return original(ctx, i)
				}
			case "missing-run":
				c.run = nil
			case "missing-authorizer":
				c.authorize = nil
			}
			quiet, err := c.apply(ctx, i)
			if err == nil || quiet || *calls != 0 {
				t.Fatal("unsafe control launch", err)
			}
		})
	}
}

func TestPropertyControlNativeFailureIsUnverified(t *testing.T) {
	for _, failure := range []error{ErrExit, ErrUnavailable, ErrPermission, ErrLimit, context.Canceled, context.DeadlineExceeded} {
		c, i, _, _, _, _, calls := controlFixture(t)
		c.run = func(context.Context, storage.PropertyIntent) error { (*calls)++; return failure }
		quiet, err := c.apply(context.Background(), i)
		if !errors.Is(err, failure) || quiet || *calls != 1 {
			t.Fatal("error inferred quiescence", err)
		}
		if _, err = c.apply(context.Background(), i); !errors.Is(err, storage.ErrDenied) || *calls != 1 {
			t.Fatal("failed control retried", err)
		}
	}
}

func TestPropertyControlConcurrentSingleAttempt(t *testing.T) {
	c, i, _, _, _, _, calls := controlFixture(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for range 32 {
		wg.Go(func() {
			quiet, err := c.apply(context.Background(), i)
			mu.Lock()
			defer mu.Unlock()
			if err == nil && quiet {
				success++
			} else if !errors.Is(err, storage.ErrDenied) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if success != 1 || *calls != 1 {
		t.Fatal("concurrent control duplicate")
	}
}

func TestPropertyControlArguments(t *testing.T) {
	for _, property := range []storage.Property{storage.Quota, storage.RefQuota, storage.Compression} {
		for _, value := range []string{"1", "18446744073709551615", "none", "off", "lz4", "gzip-9", "0", "-1", "$(touch /tmp/no)", "inherit", "zstd-99"} {
			s := policySource(func(string) (storage.Policy, error) {
				p := collectionPolicy()
				p.Scopes[0].Properties = []storage.Property{property}
				return p, nil
			})
			p, o, err := s.Collect(context.Background(), "100")
			if err != nil {
				t.Fatal(err)
			}
			o.Eligible = true
			at := time.Now()
			plan, err := storage.Build(at, "control-test", p, o, storage.Intent{FilesystemGUID: "101", Property: property, Value: value})
			if err != nil {
				continue
			}
			i, err := plan.CapturePropertyIntent(at, "control-test", p, o)
			if err != nil {
				t.Fatal(err)
			}
			args, err := propertySetArguments(i)
			if err != nil || !reflect.DeepEqual(args, []string{"set", string(property) + "=" + value, "tank"}) {
				t.Fatal("selection changed", err)
			}
			args[0] = "destroy"
			again, _ := propertySetArguments(i)
			if again[0] != "set" {
				t.Fatal("argument alias")
			}
		}
	}
	if args, err := propertySetArguments(storage.PropertyIntent{}); err == nil || len(args) != 0 {
		t.Fatal("zero intent supplied native arguments")
	}
}

func FuzzPropertyControlSelection(f *testing.F) {
	f.Add("2097152", uint8(0))
	f.Add("none", uint8(1))
	f.Add("lz4", uint8(2))
	f.Add("$(bad)", uint8(0))
	f.Fuzz(func(t *testing.T, value string, kind uint8) {
		if len(value) > 4096 {
			return
		}
		property := []storage.Property{storage.Quota, storage.RefQuota, storage.Compression}[kind%3]
		s := policySource(func(string) (storage.Policy, error) {
			p := collectionPolicy()
			p.Scopes[0].Properties = []storage.Property{property}
			return p, nil
		})
		p, o, err := s.Collect(context.Background(), "100")
		if err != nil {
			t.Fatal(err)
		}
		o.Eligible = true
		at := time.Now()
		plan, err := storage.Build(at, "control-fuzz", p, o, storage.Intent{FilesystemGUID: "101", Property: property, Value: value})
		if err != nil {
			return
		}
		i, err := plan.CapturePropertyIntent(at, "control-fuzz", p, o)
		if err != nil {
			return
		}
		args, err := propertySetArguments(i)
		if err != nil || len(args) != 3 || args[0] != "set" || args[1] != string(property)+"="+value || args[2] != "tank" || strings.ContainsAny(args[1], "\r\n\x00") {
			t.Fatal("argument widening", err)
		}
	})
}
