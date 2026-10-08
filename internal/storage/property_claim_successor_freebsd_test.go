// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// Actual metadata operations only on the submission parent's supported owned
// anchor. Intent/policy are synthetic; no native effects or eligibility follow.
func nativePropertyClaimSuccessor(t *testing.T, anchor string) {
	t.Run("claim-successor", func(t *testing.T) {
		for _, mode := range []string{"observing-success", "applying-reconcile", "observing-reconcile", "copy", "concurrent", "no-ack", "lost-applying", "raw-advance", "foreign-claim", "copied-handle", "closed", "reopened", "queued", "skipped-observing", "reconcile-replay", "terminal-replay", "intent-drift", "history-drift", "durable-drift", "other-pending", "same-terminal", "other-terminal", "missing-policy", "orphan", "late-policy", "late-presentation", "late-pending", "capacity", "file-sync", "rename", "cleanup", "directory-sync", "readback-drift", "readback-record", "succeeded-rename", "succeeded-directory-sync", "succeeded-readback-drift", "reconcile-directory-sync"} {
			t.Run(mode, func(t *testing.T) {
				path := filepath.Join(anchor, "successor-"+mode)
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
				s, err := OpenStateDirectory(path)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if s.Close() != nil || os.RemoveAll(path) != nil {
						t.Error("owned successor teardown")
					}
					if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
						t.Error("owned successor path remains")
					}
				}()
				policy := enrolledPolicy()
				policyPath := filepath.Join(path, "ownership-100.json")
				policyData, err := EncodeOwnershipRecord(policy)
				if err != nil || os.WriteFile(policyPath, policyData, 0600) != nil {
					t.Fatal("owned policy", err)
				}
				queued := nativePropertyJobFixture(t)
				if _, dup, e := s.PublishPropertySubmission(queued); e != nil || dup {
					t.Fatal("owned admission", e)
				}
				c, err := s.ClaimPropertyState(queued.job.ID)
				if err != nil {
					t.Fatal(err)
				}
				original := c.identity
				advance := func(p PropertyJob, state JobState) PropertyJob {
					t.Helper()
					reason, digest := "native-observation", strings.Repeat("d", 64)
					if state == JobApplying {
						reason, digest = "launch-intent", ""
					} else if state == JobSucceeded {
						reason = "verified"
					} else if state == JobReconcileRequired {
						reason = "executor-unverified"
						digest = p.job.History[len(p.job.History)-1].ObservationDigest
					}
					next, e := TransitionPropertyJob(p, state, p.job.History[len(p.job.History)-1].At+1, reason, digest)
					if e != nil {
						t.Fatal("owned transition", e)
					}
					return next
				}
				current := advance(queued, JobApplying)
				if mode == "no-ack" {
					if err = s.PublishPropertyJob(current, 1); err != nil {
						t.Fatal(err)
					}
				} else if mode == "lost-applying" {
					got, e := c.publishApplyingPropertyJobBounded(current, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
						tx.syncDirectory = func() error { return errors.New("owned applying acknowledgement cut") }
						return tx
					})
					if !errors.Is(e, ErrStateUncertain) || !reflect.DeepEqual(got, PropertyJob{}) || original.acknowledgedRevision != 0 {
						t.Fatal("lost applying fabricated lineage", e)
					}
				} else {
					got, e := c.PublishApplyingPropertyJob(current)
					if e != nil || !reflect.DeepEqual(got, current) {
						t.Fatal("owned applying", e)
					}
				}
				if mode == "raw-advance" || mode == "observing-reconcile" || mode == "terminal-replay" || strings.HasPrefix(mode, "succeeded-") {
					current = advance(current, JobObserving)
					if mode == "raw-advance" {
						err = s.PublishPropertyJob(current, 2)
					} else {
						_, err = c.PublishPropertyJobSuccessor(current)
					}
					if err != nil {
						t.Fatal("owned observation setup", err)
					}
				}
				if mode == "terminal-replay" || mode == "reconcile-replay" {
					state := JobSucceeded
					if mode == "reconcile-replay" {
						state = JobReconcileRequired
					}
					current = advance(current, state)
					if _, err = c.PublishPropertyJobSuccessor(current); err != nil {
						t.Fatal("owned retained outcome", err)
					}
				}
				next := clonePropertyJob(current)
				if mode != "terminal-replay" && mode != "reconcile-replay" {
					state := JobObserving
					if mode == "applying-reconcile" || mode == "observing-reconcile" || mode == "reconcile-directory-sync" {
						state = JobReconcileRequired
					} else if mode == "raw-advance" || strings.HasPrefix(mode, "succeeded-") {
						state = JobSucceeded
					}
					next = advance(current, state)
				}
				target := filepath.Join(path, "job-"+queued.job.ID+".json")
				before, err := os.ReadFile(target)
				lockBefore, lockErr := os.Stat(filepath.Join(path, "writer.lock"))
				if err != nil || lockErr != nil {
					t.Fatal("owned predecessor", err, lockErr)
				}
				receipt, revision := original.acknowledgedReceipt, original.acknowledgedRevision
				writeOther := func(kind string) {
					t.Helper()
					pool := "999"
					if kind == "same-terminal" {
						pool = "100"
					}
					other := freshPropertySubmission(t, 510, pool)
					if strings.HasSuffix(kind, "terminal") {
						other, err = TransitionPropertyJob(other, JobCancelled, other.job.History[0].At+1, "operator-cancelled", "")
					}
					data, e := EncodePropertyJob(other)
					if err != nil || e != nil || os.WriteFile(filepath.Join(path, "job-"+other.job.ID+".json"), data, 0600) != nil {
						t.Fatal("owned other hold", e)
					}
				}
				switch mode {
				case "copy":
					copied := *c
					c = &copied
				case "foreign-claim":
					copied := *c.identity
					c = &PropertyStateClaim{state: s, identity: &copied}
				case "copied-handle":
					c = &PropertyStateClaim{state: &StateDirectory{directory: s.directory, writer: s.writer, propertyOrigin: s.propertyOrigin, propertyAdmissions: s.propertyAdmissions}, identity: original}
				case "closed", "reopened":
					if err = s.Close(); err != nil {
						t.Fatal(err)
					}
					if mode == "reopened" {
						s, err = OpenStateDirectory(path)
						if err != nil {
							t.Fatal(err)
						}
						c = &PropertyStateClaim{state: s, identity: original}
					}
				case "queued":
					next = queued
				case "skipped-observing":
					next.job.History[len(next.job.History)-1].State = JobSucceeded
					next.job.History[len(next.job.History)-1].Reason = "verified"
				case "intent-drift":
					next.intent.sessionDigest[0] ^= 1
				case "history-drift":
					next.job.History[0].At++
				case "durable-drift":
					changed := clonePropertyJob(current)
					changed.job.History[1].At++
					data, e := EncodePropertyJob(changed)
					if e != nil || os.WriteFile(target, data, 0600) != nil {
						t.Fatal("owned current drift", e)
					}
					before = data
				case "other-pending", "same-terminal", "other-terminal":
					writeOther(mode)
				case "missing-policy":
					if err = os.Remove(policyPath); err != nil {
						t.Fatal(err)
					}
				case "orphan":
					if err = os.WriteFile(filepath.Join(path, ".job-"+strings.Repeat("c", 32)+".tmp"), []byte("owned orphan"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				positive := mode == "observing-success" || mode == "applying-reconcile" || mode == "observing-reconcile" || mode == "copy" || mode == "concurrent" || mode == "other-terminal"
				if mode == "concurrent" {
					var wg sync.WaitGroup
					successes := make(chan bool, 16)
					for range 16 {
						wg.Go(func() {
							copied := *c
							got, e := copied.PublishPropertyJobSuccessor(next)
							if e != nil && !errors.Is(e, ErrDenied) {
								t.Error("unexpected concurrent denial", e)
							}
							successes <- e == nil && reflect.DeepEqual(got, next)
						})
					}
					wg.Wait()
					close(successes)
					count := 0
					for ok := range successes {
						if ok {
							count++
						}
					}
					if count != 1 {
						t.Fatal("duplicate successor acknowledgements", count)
					}
				} else {
					limits := defaultStateCatalogLimits
					if mode == "capacity" {
						limits.bytes = uint64(len(before) + len(policyData))
					}
					fault := strings.TrimPrefix(mode, "succeeded-")
					fault = strings.TrimPrefix(fault, "reconcile-")
					got, e := c.publishClaimPropertyJobBounded(next, limits, func(tx ownershipPublication) ownershipPublication {
						switch fault {
						case "file-sync":
							tx.syncFile = func() error { return errors.New("owned sync cut") }
						case "rename":
							tx.rename = func() error { return errors.New("owned rename cut") }
						case "cleanup":
							cleanup := tx.cleanup
							tx.write = func() error { return errors.New("owned write cut") }
							tx.cleanup = func() error { _ = cleanup(); return errors.New("owned cleanup uncertainty") }
						case "directory-sync":
							tx.syncDirectory = func() error { return errors.New("owned directory cut") }
						case "late-policy", "late-presentation", "late-pending":
							verify := tx.verify
							tx.verify = func() error {
								if fault == "late-policy" {
									policy.Revision++
									data, e := EncodeOwnershipRecord(policy)
									if e != nil || os.WriteFile(policyPath, data, 0600) != nil {
										t.Fatal("owned late policy", e)
									}
								} else if fault == "late-presentation" {
									if os.WriteFile(target, append(append([]byte(nil), before...), '\n'), 0600) != nil {
										t.Fatal("owned presentation drift")
									}
								} else {
									writeOther("other-pending")
								}
								return verify()
							}
						case "readback-drift", "readback-record":
							syncDir := tx.syncDirectory
							tx.syncDirectory = func() error {
								if e := syncDir(); e != nil {
									return e
								}
								if fault == "readback-record" {
									return os.WriteFile(target, before, 0600)
								}
								policy.Revision++
								data, e := EncodeOwnershipRecord(policy)
								if e != nil {
									return e
								}
								return os.WriteFile(policyPath, data, 0600)
							}
						}
						return tx
					}, false)
					if (e == nil) != positive || positive && !reflect.DeepEqual(got, next) || !positive && !reflect.DeepEqual(got, PropertyJob{}) {
						t.Fatal("original successor acknowledgement", e)
					}
					uncertain := mode == "lost-applying" || mode == "orphan" || fault == "cleanup" || fault == "directory-sync" || fault == "readback-drift" || fault == "readback-record"
					if uncertain && (!errors.Is(e, ErrStateUncertain) || !s.publicationBlocked) {
						t.Fatal("uncertainty not retained", e)
					}
					if !positive && (original.acknowledgedReceipt != receipt || original.acknowledgedRevision != revision) {
						t.Fatal("failed publication fabricated acknowledgement")
					}
				}
				published := positive || strings.HasSuffix(mode, "directory-sync") || strings.HasSuffix(mode, "readback-drift")
				want := before
				if published {
					want, err = EncodePropertyJob(next)
				}
				if mode == "late-presentation" {
					want = append(append([]byte(nil), before...), '\n')
				}
				after, readErr := os.ReadFile(target)
				if err != nil || readErr != nil || !bytes.Equal(after, want) {
					t.Fatal("metadata preservation", err, readErr)
				}
				lockAfter, e := os.Stat(filepath.Join(path, "writer.lock"))
				if e != nil || !os.SameFile(lockBefore, lockAfter) {
					t.Fatal("stable writer lock", e)
				}
				if positive {
					if s.propertyAdmissions.active != original {
						t.Fatal("publication released original claim")
					}
					if got, e := c.PublishPropertyJobSuccessor(next); e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
						t.Fatal("successor replay acknowledged", e)
					}
					if mode == "observing-success" || mode == "copy" {
						done := advance(next, JobSucceeded)
						got, e := c.PublishPropertyJobSuccessor(done)
						if e != nil || !reflect.DeepEqual(got, done) {
							t.Fatal("original observed success", e)
						}
						got.job.History[0].At++
						read, _, e := s.InspectPropertyJob(done.job.ID)
						if e != nil || !reflect.DeepEqual(read, done) {
							t.Fatal("reply aliases acknowledged metadata", e)
						}
						if got, e := c.PublishPropertyJobSuccessor(done); e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
							t.Fatal("terminal replay acknowledged", e)
						}
					}
				} else {
					if got, e := c.PublishPropertyJobSuccessor(next); e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
						t.Fatal("failed successor retried", e)
					}
				}
				entries, e := os.ReadDir(path)
				if e != nil {
					t.Fatal(e)
				}
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), ".job-") && mode != "orphan" {
						t.Fatal("temporary not removed")
					}
				}
			})
		}
	})
}
