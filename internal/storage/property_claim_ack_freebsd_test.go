// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func nativePropertyClaimAcknowledgement(t *testing.T, anchor string) {
	t.Run("claim-acknowledgement", func(t *testing.T) {
		for _, mode := range []string{"exact", "copy", "queued", "raw-applying", "lost-applying", "foreign", "copied-handle", "closed", "reopened", "changed-expected", "changed-durable", "missing-policy", "other-pending", "same-terminal", "orphan", "observing", "succeeded", "reconcile", "failed-successor"} {
			t.Run(mode, func(t *testing.T) {
				path := filepath.Join(anchor, "ack-"+mode)
				if os.Mkdir(path, 0700) != nil {
					t.Fatal("owned directory")
				}
				s, err := OpenStateDirectory(path)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if s.Close() != nil || os.RemoveAll(path) != nil {
						t.Error("owned acknowledgement teardown")
					}
					if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
						t.Error("owned directory remains")
					}
				}()
				policyPath := filepath.Join(path, "ownership-100.json")
				policy, err := EncodeOwnershipRecord(enrolledPolicy())
				if err != nil || os.WriteFile(policyPath, policy, 0600) != nil {
					t.Fatal(err)
				}
				queued := nativePropertyJobFixture(t)
				if _, dup, e := s.PublishPropertySubmission(queued); e != nil || dup {
					t.Fatal(e)
				}
				c, err := s.ClaimPropertyState(queued.job.ID)
				if err != nil {
					t.Fatal(err)
				}
				applying, err := TransitionPropertyJob(queued, JobApplying, queued.job.History[0].At+1, "launch-intent", "")
				if err != nil {
					t.Fatal(err)
				}
				if mode == "raw-applying" {
					if s.PublishPropertyJob(applying, 1) != nil {
						t.Fatal("owned raw applying")
					}
				} else if mode == "lost-applying" {
					ack, err := c.publishApplyingPropertyJobBounded(applying, defaultStateCatalogLimits, func(pub ownershipPublication) ownershipPublication {
						pub.syncDirectory = func() error { return errors.New("owned acknowledgement cut") }
						return pub
					})
					if err == nil || ack.job.ID != "" {
						t.Fatal("lost acknowledgement fixture")
					}
				} else if mode != "queued" {
					if _, err := c.PublishApplyingPropertyJob(applying); err != nil {
						t.Fatal(err)
					}
				}
				expected := clonePropertyJob(applying)
				switch mode {
				case "copy":
					copied := *c
					c = &copied
				case "queued":
					expected = queued
				case "foreign":
					copied := *c.identity
					c = &PropertyStateClaim{state: s, identity: &copied}
				case "copied-handle":
					c = &PropertyStateClaim{state: &StateDirectory{directory: s.directory, writer: s.writer, propertyOrigin: s.propertyOrigin, propertyAdmissions: s.propertyAdmissions}, identity: c.identity}
				case "closed":
					if s.Close() != nil {
						t.Fatal("close")
					}
				case "reopened":
					if s.Close() != nil {
						t.Fatal("close")
					}
					s, err = OpenStateDirectory(path)
					if err != nil {
						t.Fatal(err)
					}
				case "changed-expected":
					expected.job.History[1].At++
				case "changed-durable":
					changed := clonePropertyJob(applying)
					changed.job.History[1].At++
					data, e := EncodePropertyJob(changed)
					if e != nil || os.WriteFile(filepath.Join(path, "job-"+queued.job.ID+".json"), data, 0600) != nil {
						t.Fatal(e)
					}
				case "missing-policy":
					if os.Remove(policyPath) != nil {
						t.Fatal("remove owned policy")
					}
				case "other-pending", "same-terminal":
					pool := "999"
					if mode == "same-terminal" {
						pool = "100"
					}
					other := freshPropertySubmission(t, 601, pool)
					if mode == "same-terminal" {
						other, err = TransitionPropertyJob(other, JobCancelled, other.job.History[0].At+1, "operator-cancelled", "")
					}
					data, e := EncodePropertyJob(other)
					if err != nil || e != nil || os.WriteFile(filepath.Join(path, "job-"+other.job.ID+".json"), data, 0600) != nil {
						t.Fatal(err, e)
					}
				case "orphan":
					if os.WriteFile(filepath.Join(path, ".job-"+strings.Repeat("e", 32)+".tmp"), []byte("owned orphan"), 0600) != nil {
						t.Fatal("owned orphan")
					}
				case "observing", "succeeded", "reconcile", "failed-successor":
					state, reason, digest := JobObserving, "native-observation", strings.Repeat("d", 64)
					if mode == "reconcile" {
						state, reason, digest = JobReconcileRequired, "executor-unverified", ""
					}
					next, e := TransitionPropertyJob(applying, state, applying.job.History[1].At+1, reason, digest)
					if e != nil {
						t.Fatal(e)
					}
					if mode == "failed-successor" {
						if _, e = c.publishClaimPropertyJobBounded(next, defaultStateCatalogLimits, func(pub ownershipPublication) ownershipPublication {
							pub.rename = func() error { return errors.New("owned successor cut") }
							return pub
						}, false); e == nil {
							t.Fatal("owned failure")
						}
					} else {
						if _, e = c.PublishPropertyJobSuccessor(next); e != nil {
							t.Fatal(e)
						}
						if mode == "succeeded" {
							next, e = TransitionPropertyJob(next, JobSucceeded, next.job.History[2].At+1, "verified", digest)
							if e != nil {
								t.Fatal(e)
							}
							if _, e = c.PublishPropertyJobSuccessor(next); e != nil {
								t.Fatal(e)
							}
						}
					}
				}
				jobPath := filepath.Join(path, "job-"+queued.job.ID+".json")
				before, err := os.ReadFile(jobPath)
				if err != nil {
					t.Fatal(err)
				}
				identity := *c.identity
				lock, err := os.Stat(filepath.Join(path, "writer.lock"))
				if err != nil {
					t.Fatal(err)
				}
				err = c.VerifyPropertyApplyingAcknowledgement(expected)
				positive := mode == "exact" || mode == "copy"
				if (err == nil) != positive {
					t.Fatal("exact original applying acknowledgement", err)
				}
				if positive && c.VerifyPropertyApplyingAcknowledgement(expected) != nil {
					t.Fatal("read check consumed lineage")
				}
				after, err := os.ReadFile(jobPath)
				if err != nil || !bytes.Equal(before, after) || *c.identity != identity {
					t.Fatal("read check mutated lineage/bytes", err)
				}
				final, err := os.Stat(filepath.Join(path, "writer.lock"))
				if err != nil || !os.SameFile(lock, final) {
					t.Fatal("writer lock changed", err)
				}
				if mode == "orphan" && !c.state.publicationBlocked {
					t.Fatal("orphan uncertainty not latched")
				}
				if positive && c.state.propertyAdmissions.active != c.identity {
					t.Fatal("claim released")
				}
			})
		}
	})
}
