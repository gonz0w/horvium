// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestPropertyLiveStateUIDChild(t *testing.T) {
	if os.Getenv("HORVIUM_LIVE_STATE_UID_CHILD") != "1" {
		t.Skip("owned credential child only")
	}
	if os.Geteuid() == 0 {
		t.Fatal("actual nonroot credential required")
	}
	s := &StateDirectory{}
	c := &PropertyStateClaim{state: s, identity: &propertyStateClaimIdentity{}}
	if j, d, err := c.InspectPropertyJob(strings.Repeat("a", 64)); j.job.ID != "" || d != ([32]byte{}) || !errors.Is(err, ErrStateDirectory) {
		t.Fatal("inspector UID", err)
	}
	if p, err := c.ReadOwnership("100"); !reflect.DeepEqual(p, Policy{}) || !errors.Is(err, ErrStateDirectory) {
		t.Fatal("policy reader UID", err)
	}
	if c, err := s.ClaimPropertyState(strings.Repeat("a", 64)); c != nil || !errors.Is(err, ErrStateDirectory) {
		t.Fatal("claim UID", err)
	}
	c = &PropertyStateClaim{state: s, identity: &propertyStateClaimIdentity{}}
	if err := c.CheckPropertyState(Policy{}, PoolSystemUse{}, PropertyJob{}); !errors.Is(err, ErrStateDirectory) {
		t.Fatal("readiness UID", err)
	}
}

// Called inside the existing submission parent's supported disposable metadata
// backing. Kernel lease/census/publication/UID facts are actual; policy and
// physical system-use facts below are synthetic, never native eligibility.
func nativePropertyLiveState(t *testing.T, anchor string) {
	t.Run("live-state", func(t *testing.T) {
		for _, mode := range []string{"queued-applying", "copy", "copied-handle", "reopen", "raw-publication", "duplicate-reopen", "concurrent", "late-claim", "current-drift", "receipt-drift", "uncertain-sync", "other-pending", "same-terminal", "unsafe-mode", "orphan", "policy-drift", "missing-system", "closed", "wrong-uid"} {
			t.Run(mode, func(t *testing.T) {
				path := filepath.Join(anchor, "live-"+mode)
				if os.Mkdir(path, 0700) != nil {
					t.Fatal("owned directory")
				}
				s, err := OpenStateDirectory(path)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if s.Close() != nil || os.RemoveAll(path) != nil {
						t.Error("owned lease teardown")
					}
					if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
						t.Error("owned directory remains")
					}
				}()
				p := nativePropertyJobFixture(t)
				policy := enrolledPolicy()
				policyBytes, err := EncodeOwnershipRecord(policy)
				if err != nil || os.WriteFile(filepath.Join(path, "ownership-100.json"), policyBytes, 0600) != nil {
					t.Fatal("synthetic policy fixture", err)
				}
				target := filepath.Join(path, "job-"+p.job.ID+".json")
				if mode == "raw-publication" {
					if err := s.PublishPropertyJob(p, 0); err != nil {
						t.Fatal(err)
					}
				} else if mode == "uncertain-sync" {
					got, duplicate, err := s.publishPropertySubmissionBounded(p, defaultStateCatalogLimits, func(tx ownershipPublication) ownershipPublication {
						tx.syncDirectory = func() error { return errors.New("owned barrier cut") }
						return tx
					})
					if err == nil || duplicate || got.job.ID != "" || len(s.propertyAdmissions.receipts) != 0 {
						t.Fatal("uncertain reply created receipt", err)
					}
					if c, err := s.ClaimPropertyState(p.job.ID); c != nil || !errors.Is(err, ErrStateUncertain) {
						t.Fatal("uncertainty bypass", err)
					}
					return
				} else {
					got, duplicate, err := s.PublishPropertySubmission(p)
					if err != nil || duplicate || got.job.ID != p.job.ID {
						t.Fatal("new admission", err)
					}
				}
				if mode == "reopen" || mode == "duplicate-reopen" {
					if s.Close() != nil {
						t.Fatal("close")
					}
					s, err = OpenStateDirectory(path)
					if err != nil {
						t.Fatal(err)
					}
					if mode == "duplicate-reopen" {
						got, duplicate, err := s.PublishPropertySubmission(p)
						if err != nil || !duplicate || got.job.ID != p.job.ID {
							t.Fatal("metadata retry", err)
						}
					}
				}
				if mode == "other-pending" || mode == "same-terminal" {
					pool := "999"
					if mode == "same-terminal" {
						pool = "100"
					}
					other := freshPropertySubmission(t, 151, pool)
					if mode == "same-terminal" {
						other, err = TransitionPropertyJob(other, JobCancelled, other.job.History[0].At+1, "operator-cancelled", "")
						if err != nil {
							t.Fatal(err)
						}
					}
					b, err := EncodePropertyJob(other)
					if err != nil || os.WriteFile(filepath.Join(path, "job-"+other.job.ID+".json"), b, 0600) != nil {
						t.Fatal("owned retained hold", err)
					}
				}
				if mode == "late-claim" {
					p, err = TransitionPropertyJob(p, JobApplying, p.job.History[0].At+1, "launch-intent", "")
					if err != nil || s.PublishPropertyJob(p, 1) != nil {
						t.Fatal("owned applying", err)
					}
				}
				if mode == "concurrent" {
					var wg sync.WaitGroup
					results := make(chan *PropertyStateClaim, 32)
					for range 32 {
						wg.Add(1)
						go func() { defer wg.Done(); c, _ := s.ClaimPropertyState(p.job.ID); results <- c }()
					}
					wg.Wait()
					close(results)
					grants := 0
					for c := range results {
						if c != nil {
							grants++
						}
					}
					if grants != 1 {
						t.Fatal("concurrent grants", grants)
					}
					return
				}
				if mode == "copied-handle" {
					copy := &StateDirectory{directory: s.directory, writer: s.writer, propertyAdmissions: s.propertyAdmissions, propertyOrigin: s.propertyOrigin}
					if c, err := copy.ClaimPropertyState(p.job.ID); c != nil || err == nil {
						t.Fatal("copied handle created second lease identity")
					}
				}
				c, err := s.ClaimPropertyState(p.job.ID)
				if mode == "reopen" || mode == "raw-publication" || mode == "duplicate-reopen" || mode == "late-claim" || mode == "other-pending" || mode == "same-terminal" {
					if err == nil || c != nil {
						t.Fatal("stored or obstructed metadata created claim")
					}
					return
				}
				if err != nil || c == nil {
					t.Fatal("fresh claim", err)
				}
				if again, err := s.ClaimPropertyState(p.job.ID); again != nil || err == nil {
					t.Fatal("second claim")
				}
				read, digest, e := c.InspectPropertyJob(p.job.ID)
				if e != nil || digest == ([32]byte{}) || !reflect.DeepEqual(read, p) {
					t.Fatal("claim-bound job reader", e)
				}
				ownedPolicy, e := c.ReadOwnership(policy.PoolGUID)
				if e != nil {
					t.Fatal("claim-bound policy reader", e)
				}
				ownedBytes, e := EncodeOwnershipRecord(ownedPolicy)
				if e != nil || !bytes.Equal(ownedBytes, policyBytes) {
					t.Fatal("claim-bound complete canonical policy", e)
				}
				if j, d, e := c.InspectPropertyJob(strings.Repeat("d", 64)); e == nil || j.job.ID != "" || d != ([32]byte{}) {
					t.Fatal("foreign job selection", e)
				}
				if p, e := c.ReadOwnership("999"); e == nil || !reflect.DeepEqual(p, Policy{}) {
					t.Fatal("foreign policy selection", e)
				}
				if mode == "wrong-uid" {
					nativePropertyLiveStateUID(t)
					return
				}
				// Match actual statfs source/target/capability, while explicitly
				// retaining model-only physical backing (not collected GEOM).
				backing, err := propertyStateFilesystem(int(s.directory.Fd()))
				if err != nil {
					t.Fatal(err)
				}
				u := propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\ntank/data /srv/data zfs rw 0 0\n", emptySwap)
				u.system.Mounts[0].Source = backing.source
				u.system.Mounts[0].Target = backing.target
				u.system.Mounts[0].Type = backing.kind
				if err := c.CheckPropertyState(policy, u, p); err != nil {
					t.Fatal("synthetic-input readiness", err)
				}
				if err := s.CheckPropertyState(policy, u); err == nil {
					t.Fatal("ordinary pending gate relaxed")
				}
				if mode == "copy" {
					copy := *c
					c = &copy
				}
				if mode == "queued-applying" || mode == "copy" || mode == "copied-handle" {
					p, err = TransitionPropertyJob(p, JobApplying, p.job.History[0].At+1, "launch-intent", "")
					if err != nil || s.PublishPropertyJob(p, 1) != nil {
						t.Fatal("owned applying", err)
					}
				}
				if mode == "receipt-drift" {
					p.job.History[0].At++
					b, err := EncodePropertyJob(p)
					if err != nil || os.WriteFile(target, b, 0600) != nil {
						t.Fatal("owned receipt drift", err)
					}
				}
				if mode == "current-drift" {
					p.job.History[0].At++
				}
				if mode == "unsafe-mode" && os.Chmod(target, 0644) != nil {
					t.Fatal("owned mode")
				}
				if mode == "orphan" && os.WriteFile(filepath.Join(path, ".job-"+strings.Repeat("c", 32)+".tmp"), []byte("owned orphan"), 0600) != nil {
					t.Fatal("owned orphan")
				}
				if mode == "policy-drift" {
					policy.Revision++
				}
				if mode == "missing-system" {
					u = PoolSystemUse{}
				}
				if mode == "closed" && s.Close() != nil {
					t.Fatal("close")
				}
				before, err := os.ReadFile(target)
				if err != nil {
					t.Fatal(err)
				}
				err = c.CheckPropertyState(policy, u, p)
				positive := mode == "queued-applying" || mode == "copy" || mode == "copied-handle"
				if (err == nil) != positive {
					t.Fatal("readiness gate", err)
				}
				after, e := os.ReadFile(target)
				if e != nil || !bytes.Equal(before, after) {
					t.Fatal("readiness changed records", e)
				}
				if mode == "receipt-drift" || mode == "unsafe-mode" || mode == "orphan" || mode == "closed" {
					if j, d, e := c.InspectPropertyJob(p.job.ID); e == nil || j.job.ID != "" || d != ([32]byte{}) {
						t.Fatal("unsafe claim reader retained output", e)
					}
					if p, e := c.ReadOwnership("100"); e == nil || !reflect.DeepEqual(p, Policy{}) {
						t.Fatal("unsafe ownership reader retained output", e)
					}
				}
				if mode == "orphan" && !s.publicationBlocked {
					t.Fatal("orphan not latched")
				}
			})
		}
	})
}

func nativePropertyLiveStateUID(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "horvium-live-state-uid-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if os.RemoveAll(dir) != nil {
			t.Error("owned child removal")
		}
		if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Error("owned child remains")
		}
	}()
	if os.Chmod(dir, 0755) != nil {
		t.Fatal("owned child mode")
	}
	src, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.OpenFile(filepath.Join(dir, "owned.test"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		_ = src.Close()
		t.Fatal(err)
	}
	_, copyErr := io.Copy(dst, src)
	srcErr, dstErr := src.Close(), dst.Close()
	if copyErr != nil || srcErr != nil || dstErr != nil {
		t.Fatal("owned executable copy")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(dir, "owned.test"), "-test.run=^TestPropertyLiveStateUIDChild$")
	cmd.Dir = "/"
	cmd.Env = append(os.Environ(), "HORVIUM_LIVE_STATE_UID_CHILD=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal("actual UID denial", err, string(out))
	}
}
