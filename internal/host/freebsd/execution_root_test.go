// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/gonz0w/horvium/internal/storage"
)

// Supplied identities exercise only the private original-owner contract. They
// do not qualify real kernel UID transitions or native descriptor custody.
func TestExecutionRootIdentityAdmission(t *testing.T) {
	for _, ids := range [][2]int{{0, 0}, {65534, 0}, {0, 65534}, {65534, 65534}, {-1, 0}, {0, -1}} {
		r := newExecutionOwnerRegistry()
		created, released := 0, 0
		factory := func(context.Context) (executionOwnerSource, error) {
			created++
			return executionOwnerSource{
				observe: func(context.Context) (storage.ExecutionDomainsObservation, error) {
					return storage.ExecutionDomainsObservation{}, nil
				},
				finish:  func(context.Context) error { return nil },
				release: func() error { released++; return nil },
			}, nil
		}
		root := func() bool { return executionRootIDs(ids[0], ids[1]) }
		o, err := commissionExecutionOwner(context.Background(), r, root, factory)
		if ids == [2]int{0, 0} {
			if err != nil || o == nil || created != 1 || !r.claimed || o.Close() != nil || released != 1 {
				t.Fatal("actual-root pair failed the original owner model")
			}
		} else if o != nil || !errors.Is(err, ErrPermission) || created != 0 || released != 0 || r.claimed {
			t.Fatalf("wrong real/effective pair reached construction: %v", ids)
		}
	}
}

func TestExecutionRootIdentityLossRetiresOriginal(t *testing.T) {
	for _, stage := range []string{"factory", "before-observe", "observe", "before-finish", "finish", "before-close", "close"} {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			registry := newExecutionOwnerRegistry()
			var real atomic.Int64
			root := func() bool { return executionRootIDs(int(real.Load()), 0) }
			created, observed, finished, released := 0, 0, 0, 0
			factory := func(context.Context) (executionOwnerSource, error) {
				created++
				if stage == "factory" {
					real.Store(65534)
				}
				return executionOwnerSource{
					observe: func(context.Context) (storage.ExecutionDomainsObservation, error) {
						observed++
						if stage == "observe" {
							real.Store(65534)
						}
						return storage.ExecutionDomainsObservation{}, nil
					},
					finish: func(context.Context) error {
						finished++
						if stage == "finish" {
							real.Store(65534)
						}
						return nil
					},
					release: func() error {
						released++
						if stage == "close" {
							real.Store(65534)
						}
						return nil
					},
				}, nil
			}
			owner, err := commissionExecutionOwner(ctx, registry, root, factory)
			if stage == "factory" {
				if owner != nil || !errors.Is(err, ErrPermission) || released != 1 {
					t.Fatal("construction root loss admitted an owner or skipped release")
				}
			} else {
				if err != nil || owner == nil {
					t.Fatal("original construction failed")
				}
				if stage == "before-observe" || stage == "before-finish" || stage == "before-close" {
					real.Store(65534)
				}
				switch stage {
				case "before-observe", "observe":
					if _, err = owner.Observe(ctx); !errors.Is(err, ErrPermission) || finished != 0 || stage == "before-observe" && observed != 0 || stage == "observe" && observed != 1 {
						t.Fatal("observation ignored real UID loss")
					}
				case "before-finish", "finish":
					if err = owner.FinishIdle(ctx); !errors.Is(err, ErrPermission) || stage == "before-finish" && finished != 0 || stage == "finish" && finished != 1 {
						t.Fatal("finish ignored real UID loss")
					}
				}
				beforeObserved, beforeFinished := observed, finished
				real.Store(0)
				if stage != "before-close" && stage != "close" {
					if _, err = owner.Observe(ctx); err == nil || owner.FinishIdle(ctx) == nil || observed != beforeObserved || finished != beforeFinished {
						t.Fatal("restored UID revived retired authority")
					}
				} else if stage == "before-close" {
					real.Store(65534)
				}
				closeErr := owner.Close()
				if (stage == "before-close" || stage == "close") && !errors.Is(closeErr, ErrPermission) {
					t.Fatal("release root loss reported clean release")
				}
				real.Store(0)
				if second := owner.Close(); !errors.Is(second, closeErr) || released != 1 {
					t.Fatal("release repeated or changed its original result")
				}
			}
			real.Store(0)
			if owner, err := commissionExecutionOwner(ctx, registry, root, factory); owner != nil || err == nil || created != 1 || !registry.claimed || released != 1 {
				t.Fatal("root restoration reset single-use registration")
			}
		})
	}
}
