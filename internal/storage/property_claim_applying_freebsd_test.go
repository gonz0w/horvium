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

// Actual metadata operations use the existing supported disposable anchor.
// This group performs no ZFS property effect and supplies no native eligibility.
func nativePropertyClaimApplying(t *testing.T, anchor string) {
	t.Run("claim-applying", func(t *testing.T) {
		for _, mode := range []string{"exact", "copy", "concurrent", "invalid", "foreign-candidate", "intent-drift", "cancelled", "closed", "copied-handle", "foreign-claim", "missing-policy", "other-pending", "same-terminal", "orphan", "late-policy", "late-presentation", "late-pending", "capacity", "file-sync", "rename", "cleanup", "directory-sync", "readback-drift"} {
			t.Run(mode, func(t *testing.T) {
				path := filepath.Join(anchor, "applying-"+mode)
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
				s, err := OpenStateDirectory(path)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if s.Close() != nil || os.RemoveAll(path) != nil {
						t.Error("owned applying teardown")
					}
					if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
						t.Error("owned applying path remains")
					}
				}()
				policy := enrolledPolicy()
				policyPath := filepath.Join(path, "ownership-100.json")
				policyData, err := EncodeOwnershipRecord(policy)
				if err != nil || os.WriteFile(policyPath, policyData, 0600) != nil {
					t.Fatal("owned policy", err)
				}
				p := nativePropertyJobFixture(t)
				if _, dup, e := s.PublishPropertySubmission(p); e != nil || dup {
					t.Fatal("owned admission", e)
				}
				c, err := s.ClaimPropertyState(p.job.ID)
				if err != nil {
					t.Fatal(err)
				}
				next, err := TransitionPropertyJob(p, JobApplying, p.job.History[0].At+1, "launch-intent", "")
				if err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(path, "job-"+p.job.ID+".json")
				before, err := os.ReadFile(target)
				if err != nil {
					t.Fatal(err)
				}
				lockBefore, err := os.Stat(filepath.Join(path, "writer.lock"))
				if err != nil {
					t.Fatal(err)
				}
				writeOther := func(terminal bool) {
					t.Helper()
					pool := "999"
					if terminal {
						pool = "100"
					}
					other := freshPropertySubmission(t, 181, pool)
					if terminal {
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
				case "invalid":
					next = PropertyJob{}
				case "foreign-candidate":
					next = freshPropertySubmission(t, 182, "999")
				case "intent-drift":
					next.intent.sessionDigest[0] ^= 1
				case "cancelled":
					next, err = TransitionPropertyJob(p, JobCancelled, p.job.History[0].At+1, "operator-cancelled", "")
				case "closed":
					if err = s.Close(); err != nil {
						t.Fatal(err)
					}
				case "copied-handle":
					c = &PropertyStateClaim{state: &StateDirectory{directory: s.directory, writer: s.writer, propertyOrigin: s.propertyOrigin, propertyAdmissions: s.propertyAdmissions}, identity: c.identity}
				case "foreign-claim":
					c = &PropertyStateClaim{state: s, identity: &propertyStateClaimIdentity{id: p.job.ID}}
				case "missing-policy":
					if err = os.Remove(policyPath); err != nil {
						t.Fatal(err)
					}
				case "other-pending":
					writeOther(false)
				case "same-terminal":
					writeOther(true)
				case "orphan":
					if err = os.WriteFile(filepath.Join(path, ".job-"+strings.Repeat("c", 32)+".tmp"), []byte("owned orphan"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if mode == "concurrent" {
					var wg sync.WaitGroup
					successes := make(chan bool, 16)
					for range 16 {
						wg.Add(1)
						go func() {
							defer wg.Done()
							got, e := c.PublishApplyingPropertyJob(next)
							successes <- e == nil && reflect.DeepEqual(got, next)
						}()
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
						t.Fatal("duplicate applying grants", count)
					}
				} else {
					limits := defaultStateCatalogLimits
					if mode == "capacity" {
						limits.bytes = uint64(len(before) + len(policyData))
					}
					wrap := func(tx ownershipPublication) ownershipPublication {
						switch mode {
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
								if mode == "late-policy" {
									policy.Revision++
									data, e := EncodeOwnershipRecord(policy)
									if e != nil || os.WriteFile(policyPath, data, 0600) != nil {
										t.Fatal("owned late policy", e)
									}
								}
								if mode == "late-presentation" && os.WriteFile(target, append(append([]byte(nil), before...), '\n'), 0600) != nil {
									t.Fatal("owned late presentation")
								}
								if mode == "late-pending" {
									writeOther(false)
								}
								return verify()
							}
						case "readback-drift":
							syncDir := tx.syncDirectory
							tx.syncDirectory = func() error {
								if e := syncDir(); e != nil {
									return e
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
					}
					got, e := c.publishApplyingPropertyJobBounded(next, limits, wrap)
					positive := mode == "exact" || mode == "copy"
					if (e == nil) != positive || positive && !reflect.DeepEqual(got, next) || !positive && !reflect.DeepEqual(got, PropertyJob{}) {
						t.Fatal("original applying acknowledgement", e)
					}
					if positive {
						got.job.History[0].At++
						read, digest, readErr := c.InspectPropertyJob(next.job.ID)
						if readErr != nil || digest == ([32]byte{}) || !reflect.DeepEqual(read, next) {
							t.Fatal("acknowledgement aliases original metadata", readErr)
						}
					}
					uncertain := mode == "cleanup" || mode == "directory-sync" || mode == "readback-drift" || mode == "orphan"
					if uncertain && (!errors.Is(e, ErrStateUncertain) || !s.publicationBlocked) {
						t.Fatal("uncertainty not retained", e)
					}
				}
				after, e := os.ReadFile(target)
				if e != nil {
					t.Fatal(e)
				}
				published := mode == "exact" || mode == "copy" || mode == "concurrent" || mode == "directory-sync" || mode == "readback-drift"
				want := before
				if published {
					want, err = EncodePropertyJob(next)
				}
				if mode == "late-presentation" {
					want = append(append([]byte(nil), before...), '\n')
				}
				if err != nil || !bytes.Equal(after, want) {
					t.Fatal("metadata preservation", err)
				}
				lockAfter, e := os.Stat(filepath.Join(path, "writer.lock"))
				if e != nil || !os.SameFile(lockBefore, lockAfter) {
					t.Fatal("stable writer lock", e)
				}
				if published {
					got, e := c.PublishApplyingPropertyJob(next)
					if e == nil || !reflect.DeepEqual(got, PropertyJob{}) {
						t.Fatal("applying replay acknowledged", e)
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
