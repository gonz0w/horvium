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

// Nested in the existing independent claim parent, retaining its actual backing,
// strict source/metadata/provider/sentinel preservation and 135-second deadline.
// The live callback is explicitly test-only; no complete backend is qualified.
func nativePropertyClaimControlBinding(t *testing.T, ctx context.Context, rawClaim *storage.PropertyStateClaim, rawApplying storage.PropertyJob, anchor string, policy storage.Policy) {
	t.Run("control-binding", func(t *testing.T) {
		command := func(c context.Context, args ...string) (string, error) {
			out, err := execute(c, "/sbin/zfs", args, 4096)
			return string(out.data), err
		}
		original, err := command(ctx, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
		if err != nil || strings.TrimSpace(original) != "0\tdefault" {
			t.Fatal("declared default quota fixture", original, err)
		}
		changed := false
		restore := func(c context.Context) error {
			guid, err := command(c, "get", "-H", "-p", "-o", "value", "guid", policy.PoolName)
			if err != nil || strings.TrimSpace(guid) != policy.Scopes[0].GUID {
				return errors.New("owned quota restoration GUID mismatch")
			}
			// Quota is non-inheritable; -S restores its default when no received
			// value exists. The declared fresh fixture must regain exact source.
			if _, err = command(c, "inherit", "-S", "quota", policy.PoolName); err != nil {
				return err
			}
			restored, err := command(c, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
			if err != nil || restored != original {
				return errors.New("owned quota source restoration failed")
			}
			return nil
		}
		defer func() {
			if changed {
				clean, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if e := restore(clean); e != nil {
					t.Error("owned quota restoration", e)
				}
			}
		}()
		t.Run("raw-ack-denied", func(t *testing.T) {
			if c, err := newPropertyClaimControl(rawClaim, rawApplying, func(context.Context) error { return nil }); c != nil || err == nil {
				t.Fatal("raw metadata supplied control lineage", err)
			}
		})
		path := filepath.Join(anchor, "control-state")
		if os.Mkdir(path, 0700) != nil {
			t.Fatal("owned control metadata directory")
		}
		state, err := storage.OpenStateDirectory(path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if state.Close() != nil || os.RemoveAll(path) != nil {
				t.Error("owned control metadata teardown")
			}
			if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
				t.Error("owned control metadata remains")
			}
		}()
		policyBytes, err := storage.EncodeOwnershipRecord(policy)
		if err != nil || os.WriteFile(filepath.Join(path, "ownership-"+policy.PoolGUID+".json"), policyBytes, 0600) != nil {
			t.Fatal(err)
		}
		safety, err := NewPropertySafetySource(state)
		if err != nil {
			t.Fatal(err)
		}
		p, o, err := safety.Collect(ctx, policy.PoolGUID)
		if err != nil || o.Eligible {
			t.Fatal("native read-only baseline", err)
		}
		capture := o
		capture.Eligible = true // Test-only intent capture; no source promotion.
		at := time.Now().UTC().Truncate(time.Millisecond)
		plan, err := storage.Build(at, "native-control-binding-fixture", p, capture, storage.Intent{FilesystemGUID: policy.Scopes[0].GUID, Property: storage.Quota, Value: "2097152"})
		if err != nil {
			t.Fatal(err)
		}
		intent, err := plan.CapturePropertyIntent(at, "native-control-binding-fixture", p, capture)
		if err != nil {
			t.Fatal(err)
		}
		r := intent.Review()
		queued, err := storage.BindPropertyJob(storage.Job{ID: strings.Repeat("b", 64), Intent: storage.JobIntent{PlanID: r.ID, SubmissionID: strings.Repeat("f", 64), PoolGUID: r.PoolGUID, PolicyRevision: r.PolicyRevision, Change: r.Intent}, History: []storage.JobTransition{{Revision: 1, At: uint64(at.UnixMilli()), State: storage.JobQueued, Reason: "admitted"}}}, intent)
		if err != nil {
			t.Fatal(err)
		}
		if ack, dup, e := state.PublishPropertySubmission(queued); e != nil || dup || !reflect.DeepEqual(ack, queued) {
			t.Fatal("exact fresh native submission", e)
		}
		claim, err := state.ClaimPropertyState(queued.Job().ID)
		if err != nil {
			t.Fatal(err)
		}
		applying, err := storage.TransitionPropertyJob(queued, storage.JobApplying, uint64(time.Now().UnixMilli()), "launch-intent", "")
		if err != nil {
			t.Fatal(err)
		}
		applying, err = claim.PublishApplyingPropertyJob(applying)
		if err != nil {
			t.Fatal(err)
		}
		jobPath := filepath.Join(path, "job-"+applying.Job().ID+".json")
		jobBytes, err := os.ReadFile(jobPath)
		if err != nil {
			t.Fatal(err)
		}
		lock, err := os.Stat(filepath.Join(path, "writer.lock"))
		if err != nil {
			t.Fatal(err)
		}
		// Only the disposable fixture supplies this clearance. The complete backend
		// still needs original authenticated session/clock/recovery/serialization.
		live := func(c context.Context) error {
			if err := c.Err(); err != nil {
				return err
			}
			if !time.Now().Before(r.ExpiresAt) {
				return storage.ErrExpired
			}
			return nil
		}
		t.Run("clearance-denied", func(t *testing.T) {
			c, err := newPropertyClaimControl(claim, applying, func(context.Context) error { return storage.ErrDenied })
			if err != nil {
				t.Fatal(err)
			}
			if quiet, err := c.apply(ctx, intent); quiet || !errors.Is(err, storage.ErrDenied) {
				t.Fatal("missing clearance launched", err)
			}
			actual, err := command(ctx, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
			if err != nil || actual != original {
				t.Fatal("denied control changed native quota", err)
			}
		})
		t.Run("exact-control", func(t *testing.T) {
			c, err := newPropertyClaimControl(claim, applying, live)
			if err != nil {
				t.Fatal(err)
			}
			changed = true // Arm restoration before the actual effect.
			if quiet, err := c.apply(ctx, intent); !quiet || err != nil {
				t.Fatal("original-claim fixed native control", err)
			}
			actual, err := command(ctx, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
			if err != nil || strings.TrimSpace(actual) != "2097152\tlocal" {
				t.Fatal("actual quota effect", actual, err)
			}
			if quiet, err := c.apply(ctx, intent); quiet || !errors.Is(err, storage.ErrDenied) {
				t.Fatal("single control repeated", err)
			}
		})
		t.Run("independent-postcondition", func(t *testing.T) {
			source, err := NewPropertyClaimSource(claim)
			if err != nil {
				t.Fatal(err)
			}
			j, p, o, err := source.Collect(ctx, applying.Job().ID)
			if err != nil || !reflect.DeepEqual(j, applying) || o.Eligible || intent.ComparePostcondition(p, o) != nil {
				t.Fatal("independent native effect facts", err)
			}
		})
		t.Run("preserved-applying", func(t *testing.T) {
			actual, err := os.ReadFile(jobPath)
			if err != nil || !bytes.Equal(actual, jobBytes) {
				t.Fatal("control altered durable applying metadata", err)
			}
			actual, err = os.ReadFile(filepath.Join(path, "ownership-"+policy.PoolGUID+".json"))
			if err != nil || !bytes.Equal(actual, policyBytes) {
				t.Fatal("control altered policy", err)
			}
			final, err := os.Stat(filepath.Join(path, "writer.lock"))
			if err != nil || !os.SameFile(lock, final) {
				t.Fatal("control replaced lock", err)
			}
			if err := claim.VerifyPropertyApplyingAcknowledgement(applying); err != nil {
				t.Fatal("control lost applying lineage", err)
			}
		})
		if err := restore(ctx); err != nil {
			t.Fatal("owned quota restoration", err)
		}
		changed = false
		t.Run("ledger-drift-denied", func(t *testing.T) {
			c, err := newPropertyClaimControl(claim, applying, live)
			if err != nil {
				t.Fatal(err)
			}
			job := applying.Job()
			job.History[1].At++
			other, err := storage.BindPropertyJob(job, applying.Intent())
			if err != nil {
				t.Fatal(err)
			}
			data, err := storage.EncodePropertyJob(other)
			if err != nil || os.WriteFile(jobPath, data, 0600) != nil {
				t.Fatal(err)
			}
			if quiet, err := c.apply(ctx, intent); quiet || err == nil {
				t.Fatal("drifted applying record launched", err)
			}
			if os.WriteFile(jobPath, jobBytes, 0600) != nil {
				t.Fatal("owned metadata restoration")
			}
			actual, err := command(ctx, "get", "-H", "-p", "-o", "value,source", "quota", policy.PoolName)
			if err != nil || actual != original {
				t.Fatal("denied control changed native quota", err)
			}
		})
	})
}
