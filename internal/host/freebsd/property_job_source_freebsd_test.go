// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func TestNativeAssessmentUIDChild(t *testing.T) {
	if os.Getenv("HORVIUM_ASSESSMENT_UID_CHILD") != "1" {
		t.Skip("owned child only")
	}
	if os.Geteuid() == 0 {
		t.Fatal("child retained root")
	}
	p, digest, err := (&storage.StateDirectory{}).InspectPropertyJob(strings.Repeat("a", 64))
	if !errors.Is(err, storage.ErrStateDirectory) || !reflect.DeepEqual(p, storage.PropertyJob{}) || digest != ([32]byte{}) {
		t.Fatal("wrong UID returned metadata", err)
	}
}

// The parent owns the pool/children and deadline. This fixture provisions only
// test metadata in its private /root directory; no native publication, runtime
// admission or executor is inferred from those bytes.
func nativePropertyJobSource(t *testing.T, ctx context.Context, poolSource *PoolPropertySource, policy storage.Policy, i storage.PropertyIntent, must func(testing.TB, string, ...string) string) {
	t.Helper()
	path, err := os.MkdirTemp("/root", "horvium-assessment-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if os.RemoveAll(path) != nil {
			t.Error("owned assessment directory cleanup")
		}
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Error("owned assessment directory remains")
		}
	}()
	state, err := storage.OpenStateDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
	r := i.Review()
	at := uint64(r.CreatedAt.Add(time.Millisecond).UnixMilli())
	j := storage.Job{ID: strings.Repeat("a", 64), Intent: storage.JobIntent{PlanID: r.ID, SubmissionID: strings.Repeat("c", 64), PolicyRevision: r.PolicyRevision, PoolGUID: r.PoolGUID, Change: r.Intent}, History: []storage.JobTransition{{Revision: 1, At: at, State: storage.JobQueued, Reason: "admitted"}}}
	job, err := storage.BindPropertyJob(j, i)
	if err != nil {
		t.Fatal(err)
	}
	job, err = storage.TransitionPropertyJob(job, storage.JobApplying, at+1, "launch-intent", "")
	if err != nil {
		t.Fatal(err)
	}
	jobBytes, err := storage.EncodePropertyJob(job)
	if err != nil {
		t.Fatal(err)
	}
	policyBytes, err := storage.EncodeOwnershipRecord(policy)
	if err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(path, "job-"+j.ID+".json")
	policyPath := filepath.Join(path, "ownership-"+policy.PoolGUID+".json")
	write := func(t testing.TB, name string, data []byte) {
		t.Helper()
		if os.WriteFile(name, data, 0600) != nil {
			t.Fatal("owned metadata setup")
		}
	}
	write(t, jobPath, jobBytes)
	write(t, policyPath, policyBytes)
	source, err := NewPropertyJobSource(state)
	if err != nil {
		t.Fatal(err)
	}
	lockBefore, err := os.Stat(filepath.Join(path, "writer.lock"))
	if err != nil {
		t.Fatal(err)
	}
	assess := func(t testing.TB, baseline, postcondition bool) {
		t.Helper()
		got, err := source.Assess(ctx, j.ID)
		if err != nil || got.BaselineMatches != baseline || got.PostconditionMatches != postcondition || got.Restart.Last.State != storage.JobApplying || got.Restart.State != storage.JobReconcileRequired || got.Restart.Reason != "executor-unverified" {
			t.Fatal("actual native root assessment", got, err)
		}
		view, err := state.InspectJobReservations()
		if err != nil || len(view.Pools()) != 1 || len(view.Pools()[0].Jobs) != 1 {
			t.Fatal("retained native pool hold", err)
		}
	}
	target := ""
	for _, f := range r.Context {
		if f.GUID == r.Intent.FilesystemGUID {
			target = f.Name
		}
	}
	if target == "" {
		t.Fatal("owned target missing")
	}
	t.Run("baseline", func(t *testing.T) { assess(t, true, false) })
	t.Run("expected", func(t *testing.T) {
		must(t, "/sbin/zfs", "set", "quota=2097152", target)
		assess(t, false, true)
		must(t, "/sbin/zfs", "set", "quota=1048576", target)
	})
	t.Run("neither", func(t *testing.T) {
		must(t, "/sbin/zfs", "set", "refquota=1048576", target)
		assess(t, false, false)
		must(t, "/sbin/zfs", "inherit", "-S", "refquota", target)
	})
	denied := func(t testing.TB, expected error) {
		t.Helper()
		got, err := source.Assess(ctx, j.ID)
		if err == nil || got != (storage.PropertyJobAssessment{}) || expected != nil && !errors.Is(err, expected) {
			t.Fatal("native denial exposed partial assessment", got, err)
		}
	}
	for _, kind := range []string{"unknown", "orphan", "malformed", "unsafe-mode", "missing-job", "missing-policy", "legacy", "oversized", "duplicate-plan"} {
		t.Run(kind, func(t *testing.T) {
			var extra string
			switch kind {
			case "unknown":
				extra = filepath.Join(path, "foreign")
				write(t, extra, []byte("untouched fixture"))
			case "orphan":
				extra = filepath.Join(path, ".job-"+strings.Repeat("b", 32)+".tmp")
				write(t, extra, []byte("unpublished fixture"))
			case "malformed":
				write(t, jobPath, []byte("{}"))
			case "unsafe-mode":
				if os.Chmod(jobPath, 0644) != nil {
					t.Fatal("fixture mode")
				}
			case "missing-job":
				if os.Remove(jobPath) != nil {
					t.Fatal("fixture removal")
				}
			case "missing-policy":
				if os.Remove(policyPath) != nil {
					t.Fatal("fixture removal")
				}
			case "legacy":
				data, err := storage.EncodeJobRecord(job.Job())
				if err != nil {
					t.Fatal(err)
				}
				write(t, jobPath, data)
			case "oversized":
				write(t, jobPath, bytes.Repeat([]byte(" "), (64<<10)+1))
			case "duplicate-plan":
				other := job.Job()
				other.ID, other.Intent.SubmissionID = strings.Repeat("d", 64), strings.Repeat("e", 64)
				data, err := storage.EncodeJobRecord(other)
				if err != nil {
					t.Fatal(err)
				}
				extra = filepath.Join(path, "job-"+other.ID+".json")
				write(t, extra, data)
			}
			denied(t, nil)
			if extra != "" && os.Remove(extra) != nil {
				t.Fatal("owned extra removal")
			}
			if os.Chmod(jobPath, 0600) != nil && kind != "missing-job" {
				t.Fatal("owned mode restoration")
			}
			write(t, jobPath, jobBytes)
			write(t, policyPath, policyBytes)
		})
	}
	t.Run("whole-ledger-drift", func(t *testing.T) {
		other := policy
		other.PoolGUID, other.PoolName, other.Scopes = "999", "fixture_other", nil
		data, err := storage.EncodeOwnershipRecord(other)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(path, "ownership-999.json")
		write(t, name, data)
		real := source.policy.pool.read
		cycles := 0
		source.policy.pool.read = func(ctx context.Context, probe poolProbe, name string, budget int) (readOutput, error) {
			out, err := real(ctx, probe, name, budget)
			if probe == poolTopologyPath {
				cycles++
				if cycles == 2 {
					other.Revision++
					data, e := storage.EncodeOwnershipRecord(other)
					if e != nil {
						t.Fatal(e)
					}
					write(t, filepath.Join(path, "ownership-999.json"), data)
				}
			}
			return out, err
		}
		denied(t, storage.ErrDrift)
		source.policy.pool.read = real
		if cycles != 2 || os.Remove(name) != nil {
			t.Fatal("actual native drift window missing")
		}
	})
	t.Run("presentation-drift", func(t *testing.T) {
		_, before, err := state.InspectPropertyJob(j.ID)
		if err != nil {
			t.Fatal(err)
		}
		write(t, jobPath, append(append([]byte(nil), jobBytes...), '\n'))
		_, after, err := state.InspectPropertyJob(j.ID)
		if err != nil || before == after {
			t.Fatal("actual presentation drift not bound", err)
		}
		write(t, jobPath, jobBytes)
	})
	t.Run("wrong-uid", func(t *testing.T) {
		// The unprivileged child needs a traversable directory containing only
		// this public test executable, never the private metadata or credentials.
		dir, err := os.MkdirTemp("/tmp", "horvium-assessment-uid-")
		if err != nil || os.Chmod(dir, 0755) != nil {
			t.Fatal("owned UID fixture directory", err)
		}
		defer func() {
			if os.RemoveAll(dir) != nil {
				t.Error("owned UID fixture teardown")
			}
			if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
				t.Error("owned UID fixture remains")
			}
		}()
		data, err := os.ReadFile(os.Args[0])
		binary := filepath.Join(dir, "assessment.test")
		if err != nil || os.WriteFile(binary, data, 0755) != nil {
			t.Fatal("owned executable copy", err)
		}
		cmd := exec.CommandContext(ctx, binary, "-test.v", "-test.run=^TestNativeAssessmentUIDChild$")
		cmd.Dir = dir
		cmd.Env = []string{"PATH=/sbin:/bin:/usr/sbin:/usr/bin", "LC_ALL=C", "HORVIUM_ASSESSMENT_UID_CHILD=1"}
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534, Groups: []uint32{}}}
		out, err := cmd.CombinedOutput()
		if err != nil || !bytes.Contains(out, []byte("--- PASS: TestNativeAssessmentUIDChild")) {
			t.Fatal("owned wrong-UID child failed", err)
		}
	})
	t.Run("preserved", func(t *testing.T) {
		got, _, err := state.InspectPropertyJob(j.ID)
		data, e := storage.EncodePropertyJob(got)
		lockAfter, f := os.Stat(filepath.Join(path, "writer.lock"))
		if err != nil || e != nil || f != nil || !bytes.Equal(data, jobBytes) || !os.SameFile(lockBefore, lockAfter) {
			t.Fatal("native metadata/lock preservation", err, e, f)
		}
		actual, err := poolSource.Collect(ctx, policy.PoolGUID, policy.PoolName)
		if err != nil || actual.Eligible {
			t.Fatal("restored ineligible native graph", err)
		}
	})
	t.Run("closed", func(t *testing.T) {
		if state.Close() != nil {
			t.Fatal("owned root lease close")
		}
		denied(t, storage.ErrStateDirectory)
	})
}
