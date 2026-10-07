// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"reflect"
	"testing"
)

func TestPropertyStoreCompositionRequiresTrustedCheckAndFinalPolicy(t *testing.T) {
	for _, mode := range []string{"pass", "denied", "uncertain", "missing-check", "policy-drift", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			changed, called := false, false
			rows := safetyCaptures()
			safety := &PropertySafetySource{policy: policySource(func(string) (storage.Policy, error) {
				p := safetyPolicy()
				if changed {
					p.Revision++
				}
				return p, nil
			}), providers: safetyProvider(), system: useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
				return testOutput(rows[p]), nil
			})}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			source := &PropertyStateSource{safety: safety, check: func(p storage.Policy, u storage.PoolSystemUse) error {
				called = true
				if p.PoolGUID != "100" || len(u.System().Mounts) == 0 {
					t.Fatal("unbound inputs")
				}
				switch mode {
				case "denied":
					return storage.ErrDenied
				case "uncertain":
					return storage.ErrStateUncertain
				case "policy-drift":
					changed = true
				case "canceled":
					cancel()
				}
				return nil
			}}
			if mode == "missing-check" {
				source.check = nil
			}
			p, o, err := source.Collect(ctx, "100")
			if mode == "pass" {
				if err != nil || !called || !o.Complete || o.Eligible || p.PoolGUID != "100" {
					t.Fatal("missing check/authority", err)
				}
			} else {
				if err == nil || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
					t.Fatal("failed check retained state", err)
				}
				if mode == "policy-drift" && !errors.Is(err, storage.ErrDrift) {
					t.Fatal(err)
				}
			}
		})
	}
}
