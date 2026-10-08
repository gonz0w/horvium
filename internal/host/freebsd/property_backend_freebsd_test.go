// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// Nested in the same declared independent metadata/data fixture. The callback
// explicitly supplies TEST-ONLY peer/session/recovery/global clearance, not an
// installed authenticator or recovered-executor proof. All native methods and
// fixed control execute unchanged; the wrapper independently witnesses applying.
func nativePropertyBackend(t *testing.T, ctx context.Context, anchor string, policy storage.Policy, phase string) {
	t.Run("backend-integration", func(t *testing.T) {
		command := func(c context.Context, args ...string) (string, error) {
			out, err := execute(c, "/sbin/zfs", args, 4096)
			return string(out.data), err
		}
		original, err := command(ctx, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
		if err != nil || strings.TrimSpace(original) != "0\tdefault" {
			t.Fatal("declared default quota", original, err)
		}
		restore := func(c context.Context) error {
			guid, err := command(c, "get", "-H", "-p", "-o", "value", "guid", policy.PoolName)
			if err != nil || strings.TrimSpace(guid) != policy.Scopes[0].GUID {
				return errors.New("owned backend restoration GUID mismatch")
			}
			if _, err = command(c, "inherit", "-S", "quota", policy.PoolName); err != nil {
				return err
			}
			actual, err := command(c, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
			if err != nil || actual != original {
				return errors.New("owned backend quota source restoration failed")
			}
			return nil
		}
		changed := false
		defer func() {
			if changed {
				clean, stop := context.WithTimeout(context.Background(), 15*time.Second)
				defer stop()
				if err := restore(clean); err != nil {
					t.Error(err)
				}
			}
		}()
		setup := func(suffix string) (*storage.StateDirectory, string, []byte) {
			t.Helper()
			path := filepath.Join(anchor, "backend-"+suffix)
			if os.Mkdir(path, 0700) != nil {
				t.Fatal("owned backend directory")
			}
			state, err := storage.OpenStateDirectory(path)
			if err != nil {
				t.Fatal(err)
			}
			data, err := storage.EncodeOwnershipRecord(policy)
			if err != nil || os.WriteFile(filepath.Join(path, "ownership-"+policy.PoolGUID+".json"), data, 0600) != nil {
				t.Fatal("owned backend policy", err)
			}
			t.Cleanup(func() {
				if err := state.Close(); err != nil {
					t.Error("owned backend close", err)
				}
				if err := os.RemoveAll(path); err != nil {
					t.Error("owned backend directory removal", err)
				}
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Error("owned backend directory remains")
				}
			})
			return state, path, data
		}
		selection := storage.Intent{FilesystemGUID: policy.Scopes[0].GUID, Property: storage.Quota, Value: "2097152"}
		if phase == "success-and-denials" {
			state, path, policyBytes := setup("success")
			allowed := false
			backend, err := newPropertyBackend(state, policy.PoolGUID, func(c context.Context) error {
				if err := c.Err(); err != nil {
					return err
				}
				if !allowed {
					return storage.ErrDenied
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			coordinator := storage.NewPropertyCoordinator(backend)
			t.Run("clearance-denied", func(t *testing.T) {
				if _, err := coordinator.Review(ctx, "native-backend-fixture", selection); !errors.Is(err, storage.ErrDenied) {
					t.Fatal("absent trusted clearance authorized review", err)
				}
				entries, err := os.ReadDir(path)
				if err != nil || len(entries) != 2 {
					t.Fatal("denied review published metadata", err)
				}
			})
			allowed = true
			review, err := coordinator.Review(ctx, "native-backend-fixture", selection)
			if err != nil {
				t.Fatal(err)
			}
			queued, dup, err := coordinator.Submit(ctx, "native-backend-fixture", review.ID, strings.Repeat("7", 64))
			if err != nil || dup {
				t.Fatal("native backend admission", err)
			}
			jobPath := filepath.Join(path, "job-"+queued.Job().ID+".json")
			lock, err := os.Stat(filepath.Join(path, "writer.lock"))
			if err != nil {
				t.Fatal(err)
			}
			t.Run("original-admission", func(t *testing.T) {
				raw, err := os.ReadFile(jobPath)
				expected, _ := storage.EncodePropertyJob(queued)
				if err != nil || !bytes.Equal(raw, expected) || backendState(queued) != storage.JobQueued {
					t.Fatal("exact actual queued metadata", err)
				}
				ack, dup, err := coordinator.Submit(ctx, "native-backend-fixture", review.ID, strings.Repeat("7", 64))
				if err != nil || !dup || !reflect.DeepEqual(ack, queued) {
					t.Fatal("duplicate confirm lost original admission", err)
				}
			})
			worker, err := coordinator.ClaimPropertyExecution(ctx, "native-backend-fixture", queued.Job().ID)
			if err != nil {
				t.Fatal(err)
			}
			factory := backend.source.control
			witnessed, calls := false, 0
			backend.source.control = func(j storage.PropertyJob, live func(context.Context) error) (*propertyControl, error) {
				control, err := factory(j, live)
				if err != nil {
					return nil, err
				}
				native := control.run
				control.run = func(c context.Context, i storage.PropertyIntent) error {
					raw, err := os.ReadFile(jobPath)
					expected, _ := storage.EncodePropertyJob(j)
					if err != nil || !bytes.Equal(raw, expected) || backendState(j) != storage.JobApplying || len(j.Job().History) != 2 {
						return errors.New("actual applying metadata missing before effect")
					}
					if err = backend.authority.CheckLive(c); err != nil {
						return err
					}
					witnessed = true
					calls++
					return native(c, i) // Actual unchanged fixed guarded native operation.
				}
				return control, nil
			}
			changed = true // Arm exact native restoration before any control can execute.
			var outcome storage.PropertyJob
			t.Run("durable-success", func(t *testing.T) {
				var err error
				outcome, err = worker.Run(ctx, queued.Job().ID)
				if err != nil || backendState(outcome) != storage.JobSucceeded || calls != 1 {
					t.Fatal("actual private backend execution", err)
				}
				raw, err := os.ReadFile(jobPath)
				expected, _ := storage.EncodePropertyJob(outcome)
				if err != nil || !bytes.Equal(raw, expected) {
					t.Fatal("exact actual durable outcome", err)
				}
				h := outcome.Job().History
				if len(h) != 4 || h[2].State != storage.JobObserving || h[2].ObservationDigest == "" || h[2].ObservationDigest != h[3].ObservationDigest || !backend.matched {
					t.Fatal("outcome lacks actual observation witness")
				}
			})
			t.Run("applying-before-effect", func(t *testing.T) {
				if !witnessed || calls != 1 {
					t.Fatal("native effect lacks applying/live witness")
				}
			})
			t.Run("independent-postcondition", func(t *testing.T) {
				s, err := NewPropertySafetySource(state)
				if err != nil {
					t.Fatal(err)
				}
				p, o, err := s.Collect(ctx, policy.PoolGUID)
				if err != nil || o.Eligible || outcome.Intent().ComparePostcondition(p, o) != nil {
					t.Fatal("independent actual native effect", err)
				}
				actual, err := command(ctx, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
				if err != nil || strings.TrimSpace(actual) != "2097152\tlocal" {
					t.Fatal("exact native quota/source", actual, err)
				}
			})
			t.Run("no-replay", func(t *testing.T) {
				if _, err := worker.Run(ctx, queued.Job().ID); !errors.Is(err, storage.ErrDenied) {
					t.Fatal("worker replay", err)
				}
				if q, err := backend.ApplyProperty(ctx, queued.Intent()); q || err == nil {
					t.Fatal("backend replay", err)
				}
				if e, err := coordinator.ClaimPropertyExecution(ctx, "native-backend-fixture", queued.Job().ID); e != nil || err == nil {
					t.Fatal("second handoff", err)
				}
				if calls != 1 {
					t.Fatal("native control repeated")
				}
				if err := backend.authority.CheckLive(ctx); err == nil {
					t.Fatal("completed authority remains live")
				}
			})
			t.Run("retained-hold", func(t *testing.T) {
				source, err := NewPropertyStateSource(state)
				if err != nil {
					t.Fatal(err)
				}
				p, o, err := source.Collect(ctx, policy.PoolGUID)
				if err == nil || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
					t.Fatal("terminal label released native hold", err)
				}
			})
			t.Run("preserved-metadata", func(t *testing.T) {
				actual, err := os.ReadFile(filepath.Join(path, "ownership-"+policy.PoolGUID+".json"))
				if err != nil || !bytes.Equal(actual, policyBytes) {
					t.Fatal("backend changed policy", err)
				}
				final, err := os.Stat(filepath.Join(path, "writer.lock"))
				if err != nil || !os.SameFile(lock, final) {
					t.Fatal("backend replaced lock", err)
				}
				entries, err := os.ReadDir(path)
				if err != nil || len(entries) != 3 {
					t.Fatal("backend left temporary metadata", err)
				}
			})
			if err := restore(ctx); err != nil {
				t.Fatal(err)
			}
			changed = false
			for _, mode := range []string{"revoked-clearance", "closed-lease"} {
				t.Run(mode, func(t *testing.T) {
					state, path, _ := setup(mode)
					allowed := true
					b, err := newPropertyBackend(state, policy.PoolGUID, func(c context.Context) error {
						if !allowed {
							return storage.ErrDenied
						}
						return c.Err()
					})
					if err != nil {
						t.Fatal(err)
					}
					c := storage.NewPropertyCoordinator(b)
					r, err := c.Review(ctx, "native-backend-cut", selection)
					if err != nil {
						t.Fatal(err)
					}
					q, dup, err := c.Submit(ctx, "native-backend-cut", r.ID, strings.Repeat("8", 64))
					if err != nil || dup {
						t.Fatal(err)
					}
					e, err := c.ClaimPropertyExecution(ctx, "native-backend-cut", q.Job().ID)
					if err != nil {
						t.Fatal(err)
					}
					factory := b.source.control
					b.source.control = func(j storage.PropertyJob, live func(context.Context) error) (*propertyControl, error) {
						control, err := factory(j, live)
						if err != nil {
							return nil, err
						}
						if mode == "revoked-clearance" {
							allowed = false
						} else if err = state.Close(); err != nil {
							return nil, err
						}
						return control, nil
					}
					got, err := e.Run(ctx, q.Job().ID)
					if err == nil {
						t.Fatal("native prelaunch cut reported success")
					}
					want := storage.JobReconcileRequired
					if mode == "closed-lease" {
						want = storage.JobApplying
					}
					if backendState(got) != want || b.controlAttempted {
						t.Fatal("prelaunch cut lost honest last ack", backendState(got), err)
					}
					raw, err := os.ReadFile(filepath.Join(path, "job-"+q.Job().ID+".json"))
					expected, _ := storage.EncodePropertyJob(got)
					if err != nil || !bytes.Equal(raw, expected) {
						t.Fatal("cut lost actual retained metadata", err)
					}
					actual, err := command(ctx, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
					if err != nil || actual != original {
						t.Fatal("denied backend changed quota", actual, err)
					}
				})
			}
		}
		if phase == "gaps" {
			for _, boundary := range []string{"applying", "succeeded"} {
				t.Run("lost-"+boundary+"-ack", func(t *testing.T) {
					state, path, _ := setup("lost-" + boundary)
					b, err := newPropertyBackend(state, policy.PoolGUID, func(c context.Context) error { return c.Err() })
					if err != nil {
						t.Fatal(err)
					}
					c := storage.NewPropertyCoordinator(b)
					r, err := c.Review(ctx, "native-backend-gap", selection)
					if err != nil {
						t.Fatal(err)
					}
					q, dup, err := c.Submit(ctx, "native-backend-gap", r.ID, strings.Repeat("9", 64))
					if err != nil || dup {
						t.Fatal(err)
					}
					e, err := c.ClaimPropertyExecution(ctx, "native-backend-gap", q.Job().ID)
					if err != nil {
						t.Fatal(err)
					}
					var visible storage.PropertyJob
					wrap := func(publish func(storage.PropertyJob) (storage.PropertyJob, error)) func(storage.PropertyJob) (storage.PropertyJob, error) {
						return func(next storage.PropertyJob) (storage.PropertyJob, error) {
							ack, err := publish(next)
							if err != nil {
								return ack, err
							}
							if string(backendState(next)) == boundary {
								visible = ack
								return storage.PropertyJob{}, ErrUnavailable
							}
							return ack, nil
						}
					}
					b.source.applying = wrap(b.source.applying)
					b.source.successor = wrap(b.source.successor)
					changed = true
					got, err := e.Run(ctx, q.Job().ID)
					if !errors.Is(err, storage.ErrStateUncertain) || !b.blocked {
						t.Fatal("actual acknowledgement gap reported success", err)
					}
					want := storage.JobQueued
					if boundary == "succeeded" {
						want = storage.JobObserving
					}
					if backendState(got) != want || string(backendState(visible)) != boundary {
						t.Fatal("gap invented/discarded acknowledged history", backendState(got), backendState(visible))
					}
					raw, err := os.ReadFile(filepath.Join(path, "job-"+q.Job().ID+".json"))
					expected, _ := storage.EncodePropertyJob(visible)
					if err != nil || !bytes.Equal(raw, expected) {
						t.Fatal("actual visible metadata missing after gap", err)
					}
					if _, err := e.Run(ctx, q.Job().ID); !errors.Is(err, storage.ErrDenied) {
						t.Fatal("gap replayed worker", err)
					}
					if quiet, err := b.ApplyProperty(ctx, q.Intent()); quiet || !errors.Is(err, storage.ErrStateUncertain) {
						t.Fatal("gap replayed backend", err)
					}
					actual, err := command(ctx, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
					if err != nil {
						t.Fatal(err)
					}
					if boundary == "applying" && (actual != original || b.controlAttempted) {
						t.Fatal("unacknowledged applying launched native effect")
					}
					if boundary == "succeeded" && (strings.TrimSpace(actual) != "2097152\tlocal" || !b.quiet || !b.matched) {
						t.Fatal("actual effect/independent facts disappeared after success gap")
					}
					if err := restore(ctx); err != nil {
						t.Fatal(err)
					}
					changed = false
				})
			}
		}
	})
}
