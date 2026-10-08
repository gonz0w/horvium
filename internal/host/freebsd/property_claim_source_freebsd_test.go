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

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

// Both declared native parents run serially in the same disposable guest.
// The new fixture has its own bounded context rather than consuming the old
// collector's unchanged five-minute deadline. No runtime behavior changes.
func TestNativePropertyClaimCollection(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PROPERTY_VM") != "1" {
		t.Skip("requires disposable property guest opt-ins")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires disposable guest root")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 135*time.Second)
	defer cancel()
	teardown := false
	nativePropertySystemFixtureWithClaim(t, ctx, func(name string, ok bool) {
		if name != "claim_fixture_teardown" || !ok || teardown {
			t.Fatal("owned claim fixture restoration", name)
		}
		teardown = true
	}, func(p storage.Policy, ordinary string, disk inventory.Provider) {
		nativePropertyClaimFixture(t, ctx, p, ordinary, disk)
	})
	if !teardown {
		t.Fatal("owned claim fixture teardown missing")
	}
	t.Log("CLAIM_PROPERTY_COLLECTION_RESULT fixture_teardown=pass")
}

// The existing system fixture supplies its enrolled data mirror and one of its
// already declared nonboot snapshot disks. Metadata uses a separate unenrolled
// ZFS pool under /root, with actual native physical backing on both guests.
// No root filesystem flags or product provisioning behavior change.
func nativePropertyClaimFixture(t *testing.T, ctx context.Context, policy storage.Policy, ordinary string, metadataDisk inventory.Provider) {
	t.Helper()
	t.Run("native-claim-source", func(t *testing.T) {
		command := func(c context.Context, path string, args ...string) (string, error) {
			out, err := execute(c, path, args, 1<<20)
			return string(out.data), err
		}
		must := func(path string, args ...string) string {
			t.Helper()
			out, err := command(ctx, path, args...)
			if err != nil {
				t.Fatal("owned claim fixture command", path, err)
			}
			return out
		}
		raw := func() string {
			return must("/sbin/zfs", "get", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,property,value,received,source", "guid,quota,refquota,compression,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus", policy.PoolName)
		}
		original := raw()
		sentinelPath := filepath.Join(ordinary, "sentinel")
		sentinel, err := os.ReadFile(sentinelPath)
		if err != nil {
			t.Fatal(err)
		}
		mountsBefore := must("/sbin/mount", "-p")
		anchor, err := os.MkdirTemp("/root", "horvium-claim-backing-")
		if err != nil {
			t.Fatal(err)
		}
		pool := "horvium_claim_" + strings.TrimPrefix(filepath.Base(anchor), "horvium-claim-backing-")
		guid := ""
		created := false
		var state *storage.StateDirectory
		path := filepath.Join(anchor, "state")
		cleanup := func() bool {
			clean, stop := context.WithTimeout(context.Background(), 30*time.Second)
			defer stop()
			if state != nil {
				if state.Close() != nil {
					return false
				}
				state = nil
			}
			if created {
				if err := os.RemoveAll(path); err != nil {
					return false
				}
				actual, err := command(clean, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool)
				if err != nil || guid == "" || strings.TrimSpace(actual) != guid {
					return false
				}
				if _, err = command(clean, "/sbin/zpool", "destroy", pool); err != nil {
					return false
				}
				created = false
				if _, err = command(clean, "/sbin/zpool", "list", "-H", pool); err == nil {
					return false
				}
			}
			if err := os.Remove(anchor); err != nil {
				return false
			}
			if _, err := os.Lstat(anchor); !errors.Is(err, os.ErrNotExist) {
				return false
			}
			return true
		}
		cleaned := false
		defer func() {
			if !cleaned && !cleanup() {
				t.Error("owned claim backing teardown failed")
			}
		}()
		if metadataDisk.Class != "DISK" || metadataDisk.Kind != inventory.DiskCandidate || metadataDisk.MediaSize != 64<<20 || !strings.HasPrefix(metadataDisk.Name, "vtbd") || strings.ContainsAny(metadataDisk.Name, "/ \\\t\r\n") {
			t.Fatal("undeclared metadata fixture disk")
		}
		if _, e := command(ctx, "/sbin/zpool", "list", "-H", pool); e == nil {
			t.Fatal("fixture pool exists")
		}
		must("/sbin/zpool", "create", "-m", anchor, "-o", "cachefile=none", "-o", "ashift=12", pool, "/dev/"+metadataDisk.Name)
		created = true
		guid = strings.TrimSpace(must("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool))
		if os.Chmod(anchor, 0700) != nil || os.Mkdir(path, 0700) != nil {
			t.Fatal("private metadata directory")
		}
		state, err = storage.OpenStateDirectory(path)
		if err != nil {
			t.Fatal(err)
		}
		policyBytes, err := storage.EncodeOwnershipRecord(policy)
		if err != nil || os.WriteFile(filepath.Join(path, "ownership-"+policy.PoolGUID+".json"), policyBytes, 0600) != nil {
			t.Fatal("owned enrolled policy", err)
		}
		t.Run("independent-backing", func(t *testing.T) {
			system, err := NewSystemUseSource()
			if err != nil {
				t.Fatal(err)
			}
			u, err := system.Collect(ctx)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, m := range u.System().Mounts {
				if m.Target == anchor && m.Source == pool && m.Type == "zfs" {
					found = len(m.Disks) == 1 && m.Disks[0].Name == metadataDisk.Name && m.PoolGUID == guid
				}
			}
			if !found {
				t.Fatal("actual metadata physical backing")
			}
			var fs unix.Statfs_t
			if unix.Statfs(path, &fs) != nil || unix.ByteSliceToString(fs.Fstypename[:]) != "zfs" || unix.ByteSliceToString(fs.Mntfromname[:]) != pool || unix.ByteSliceToString(fs.Mntonname[:]) != anchor || fs.Flags&unix.MNT_RDONLY != 0 {
				t.Fatal("actual kernel metadata backing")
			}
			if err := state.CheckPropertyState(policy, u); err != nil {
				t.Fatal("real empty-store prerequisite", err)
			}
		})
		safety, err := NewPropertySafetySource(state)
		if err != nil {
			t.Fatal(err)
		}
		p, o, err := safety.Collect(ctx, policy.PoolGUID)
		if err != nil || !o.Complete || o.Eligible {
			t.Fatal("real safety baseline", err)
		}
		at := time.Now().UTC().Truncate(time.Millisecond)
		capture := o
		capture.Eligible = true // Only test capture; sources never return this.
		plan, err := storage.Build(at, "native-claim-fixture", p, capture, storage.Intent{FilesystemGUID: policy.Scopes[0].GUID, Property: storage.Quota, Value: "2097152"})
		if err != nil {
			t.Fatal(err)
		}
		intent, err := plan.CapturePropertyIntent(at, "native-claim-fixture", p, capture)
		if err != nil {
			t.Fatal(err)
		}
		review := intent.Review()
		job, err := storage.BindPropertyJob(storage.Job{ID: strings.Repeat("a", 64), Intent: storage.JobIntent{PlanID: review.ID, SubmissionID: strings.Repeat("c", 64), PoolGUID: review.PoolGUID, PolicyRevision: review.PolicyRevision, Change: review.Intent}, History: []storage.JobTransition{{Revision: 1, At: uint64(at.Add(time.Millisecond).UnixMilli()), State: storage.JobQueued, Reason: "admitted"}}}, intent)
		if err != nil {
			t.Fatal(err)
		}
		if ack, duplicate, err := state.PublishPropertySubmission(job); err != nil || duplicate || !reflect.DeepEqual(ack, job) {
			t.Fatal("real new submission", err)
		}
		claim, err := state.ClaimPropertyState(job.Job().ID)
		if err != nil {
			t.Fatal(err)
		}
		source, err := NewPropertyClaimSource(claim)
		if err != nil {
			t.Fatal(err)
		}
		collect := func(t *testing.T, expected storage.PropertyJob) {
			t.Helper()
			got, p, o, err := source.Collect(ctx, job.Job().ID)
			if err != nil || !reflect.DeepEqual(got, expected) || !o.Complete || o.Eligible {
				t.Fatal("real claimed collection", err)
			}
			a, err := got.AssessObservation(p, o)
			if err != nil || !a.BaselineMatches || a.PostconditionMatches {
				t.Fatal("real baseline", err)
			}
			if _, err := storage.Build(time.Now(), "native-no-authority", p, o, review.Intent); !errors.Is(err, storage.ErrUnavailable) {
				t.Fatal("native eligibility promoted", err)
			}
		}
		t.Run("queued", func(t *testing.T) { collect(t, job) })
		applying, err := storage.TransitionPropertyJob(job, storage.JobApplying, job.Job().History[0].At+1, "launch-intent", "")
		if err != nil || state.PublishPropertyJob(applying, 1) != nil {
			t.Fatal("owned applying publication", err)
		}
		t.Run("applying", func(t *testing.T) { collect(t, applying) })
		t.Run("foreign-selection", func(t *testing.T) {
			j, p, o, err := source.Collect(ctx, strings.Repeat("d", 64))
			if err == nil || !reflect.DeepEqual(j, storage.PropertyJob{}) || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("foreign selection retained projections", err)
			}
		})
		applyingBytes, err := storage.EncodePropertyJob(applying)
		if err != nil {
			t.Fatal(err)
		}
		t.Run("closed", func(t *testing.T) {
			if state.Close() != nil {
				t.Fatal("lease close")
			}
			j, p, o, err := source.Collect(ctx, job.Job().ID)
			if err == nil || !reflect.DeepEqual(j, storage.PropertyJob{}) || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("closed lease returned projections", err)
			}
		})
		t.Run("reopened", func(t *testing.T) {
			state, err = storage.OpenStateDirectory(path)
			if err != nil {
				t.Fatal(err)
			}
			if c, err := state.ClaimPropertyState(job.Job().ID); err == nil || c != nil {
				t.Fatal("reopened job restored claim", err)
			}
			if ack, duplicate, err := state.PublishPropertySubmission(job); err != nil || !duplicate || !reflect.DeepEqual(ack, applying) {
				t.Fatal("retained metadata retry", err)
			}
			if c, err := state.ClaimPropertyState(job.Job().ID); err == nil || c != nil {
				t.Fatal("retry restored claim", err)
			}
		})
		t.Run("preserved", func(t *testing.T) {
			actual, err := os.ReadFile(filepath.Join(path, "job-"+job.Job().ID+".json"))
			if err != nil || !bytes.Equal(actual, applyingBytes) {
				t.Fatal("job changed", err)
			}
			actual, err = os.ReadFile(filepath.Join(path, "ownership-"+policy.PoolGUID+".json"))
			if err != nil || !bytes.Equal(actual, policyBytes) {
				t.Fatal("policy changed", err)
			}
			actual, err = os.ReadFile(sentinelPath)
			if err != nil || !bytes.Equal(actual, sentinel) || raw() != original {
				t.Fatal("data fixture changed", err)
			}
		})
		if !cleanup() {
			t.Fatal("owned backing cleanup")
		}
		cleaned = true
		t.Run("teardown", func(t *testing.T) {
			if must("/sbin/mount", "-p") != mountsBefore {
				t.Fatal("owned metadata mount remains")
			}
		})
	})
}
