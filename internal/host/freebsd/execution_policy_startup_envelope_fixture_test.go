//go:build freebsd && horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

// Separate adopted fixture profile; old trace fixture remains unchanged.
var policyEnvelopeChildControls = strings.Fields("cpu_affinity child_budget caller_deadline original_call sampler_joined samples_bounded owner_release deadline_unchanged")

func TestMultiplePolicyStartupEnvelopeChild(t *testing.T) {
	if os.Getenv("HORVIUM_POLICY_STARTUP_ENVELOPE_CHILD") != "1" {
		t.Skip("owned diagnostic child only")
	}
	requirePolicyFactsVM(t)
	if os.Getenv("HORVIUM_POLICY_STARTUP_ENVELOPE_VM") != "1" || runtime.GOMAXPROCS(0) != 1 {
		t.Fatal("fixed diagnostic child")
	}
	j, l := os.Getenv("HORVIUM_RESTART_JOURNAL"), os.Getenv("HORVIUM_RESTART_LEDGER")
	_, seed := readActivationFixture(t, j)
	if seed.Owner.State != "closed" || seed.Metadata.Pending || seed.Native.Pending || seed.Disarmed || len(multiplePolicyReviewFiles(t, l)) != 3 {
		t.Fatal("original closed two-policy seed")
	}
	dir := filepath.Join(filepath.Dir(j), "startup-envelope")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal("child-owned snapshot directory", e)
	}
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("startup trace child", name)
		}
		seen = append(seen, name)
	}
	check("cpu_affinity", multiplePolicyLifecycleAffinity(t))
	flagTimeout := flag.Lookup("test.timeout")
	if flagTimeout == nil {
		t.Fatal("missing actual test timeout flag")
	}
	deadline, bounded := t.Deadline()
	budget, e := time.ParseDuration(flagTimeout.Value.String())
	check("child_budget", e == nil && bounded && budget == 210*time.Second && deadline.After(time.Now()) && time.Until(deadline) <= budget)
	start := time.Now()
	c, stop := newPolicyEnvelopeCaller(start)
	defer stop()
	callerDeadline, ok := c.Deadline()
	check("caller_deadline", ok && policyEnvelopeCallerUnchanged(c, start) && c.Err() == nil && time.Until(callerDeadline) > 179*time.Second && time.Until(callerDeadline) <= 180*time.Second)
	sampler, e := startPolicyTraceSampler(start, dir, [3]time.Duration{30 * time.Second, 75 * time.Second, 110 * time.Second}, func(buffer []byte) int { return runtime.Stack(buffer, true) })
	if e != nil {
		t.Fatal(e)
	}
	// One actual original factory call; never substitute or wrap native producers.
	owner, startupError := activateNativeClosedExecutionOwner(c, j, l)
	returned := time.Since(start).Nanoseconds()
	callerError := policyTraceError(c.Err())
	released := false
	defer func() {
		if owner != nil && !released {
			released = true
			if e := owner.Close(); e != nil {
				t.Error("original owner release", e)
			}
		}
	}()
	rows, e := sampler.finish(5 * time.Second)
	check("sampler_joined", e == nil && len(rows) == 3)
	check("original_call", (startupError == nil) == (owner != nil))
	for i := range rows {
		if rows[i].Status == "captured" && rows[i].End > returned {
			rows[i].Status = "overlap"
		}
	}
	check("samples_bounded", len(rows) == 3)
	if owner != nil {
		released = true
		e = owner.Close()
	}
	check("owner_release", e == nil)
	again, bounded := c.Deadline()
	check("deadline_unchanged", bounded && again.Equal(callerDeadline) && policyEnvelopeCallerUnchanged(c, start))
	summary := policyTraceSummary{Return: returned, Deadline: callerDeadline.Sub(start).Nanoseconds(), CallerError: callerError, StartupError: policyTraceError(startupError), Owner: owner != nil, Samples: rows}
	data, e := json.Marshal(summary)
	if e != nil {
		t.Fatal(e)
	}
	fmt.Println("POLICY_STARTUP_ENVELOPE_SUMMARY " + string(data))
	fmt.Println("POLICY_STARTUP_ENVELOPE_CHILD_RESULT " + strings.Join(seen, " "))
}

func TestNativeMultiplePolicyStartupEnvelopeUFS(t *testing.T) {
	if os.Getenv("HORVIUM_POLICY_STARTUP_ENVELOPE_VM") != "1" {
		t.Skip("owned startup trace only")
	}
	requirePolicyFactsVM(t)
	var seen []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("startup trace parent", name)
		}
		seen = append(seen, name+"=pass")
	}
	check("cpu_affinity", multiplePolicyLifecycleAffinity(t))
	budgetOK := nativeMultiplePolicyFactsFixtureWithProfile(t, policyFixtureFactsWork, func(ctx context.Context, policies []storage.Policy, disk inventory.Provider) {
		domains, mounts := nativeMultiplePolicyMetadataFixture(t, ctx, "ufs", disk, func(jp, lp string) {
			j, l := filepath.Join(jp, "trace"), filepath.Join(lp, "trace")
			system, e := NewSystemUseSource()
			if e != nil {
				t.Fatal(e)
			}
			if e = storage.SeedExecutionPolicyFixture(ctx, j, l, readNativeExecutionBacking, system.Collect, readNativeActivationBoot, policies); e != nil {
				t.Fatal("original trace seed", e)
			}
			before, seed := readActivationFixture(t, j)
			ledgerBefore := multiplePolicyReviewFiles(t, l)
			journalBefore := multiplePolicyReviewFiles(t, j)
			check("seed_closed", seed.Owner.State == "closed" && !seed.Metadata.Pending && !seed.Native.Pending && !seed.Disarmed && len(ledgerBefore) == 3 && !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.v", "-test.count=1", "-test.timeout=210s", "-test.run=^TestMultiplePolicyStartupEnvelopeChild$")
			cmd.Env = append(os.Environ(), "GOMAXPROCS=1", "HORVIUM_POLICY_STARTUP_ENVELOPE_CHILD=1", "HORVIUM_RESTART_JOURNAL="+j, "HORVIUM_RESTART_LEDGER="+l)
			data, e := cmd.CombinedOutput()
			if e != nil {
				t.Logf("bounded diagnostic child: %.16384s", data)
			}
			check("child_waited", e == nil && cmd.ProcessState != nil && cmd.ProcessState.Success() && len(data) <= 16<<10)
			var controls, summaryRows []string
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "POLICY_STARTUP_ENVELOPE_CHILD_RESULT ") {
					controls = append(controls, strings.TrimPrefix(line, "POLICY_STARTUP_ENVELOPE_CHILD_RESULT "))
				}
				if strings.HasPrefix(line, "POLICY_STARTUP_ENVELOPE_SUMMARY ") {
					summaryRows = append(summaryRows, strings.TrimPrefix(line, "POLICY_STARTUP_ENVELOPE_SUMMARY "))
				}
			}
			good := len(controls) == 1 && len(summaryRows) == 1
			if good {
				actual := strings.Fields(controls[0])
				want := append([]string(nil), policyEnvelopeChildControls...)
				sort.Strings(actual)
				sort.Strings(want)
				good = reflect.DeepEqual(actual, want)
			}
			check("child_inventory", good)
			for _, name := range policyEnvelopeChildControls {
				check("child_"+name, true)
			}
			var summary policyTraceSummary
			decoder := json.NewDecoder(strings.NewReader(summaryRows[0]))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&summary) != nil || len(summary.Samples) != 3 {
				t.Fatal("exact scalar child record")
			}
			fmt.Println("POLICY_STARTUP_ENVELOPE_SUMMARY " + summaryRows[0])
			dir := filepath.Join(jp, "startup-envelope")
			var directoryStat unix.Stat_t
			if unix.Lstat(dir, &directoryStat) != nil || directoryStat.Uid != 0 || directoryStat.Mode&07777 != 0700 || directoryStat.Mode&unix.S_IFMT != unix.S_IFDIR {
				t.Fatal("original child-owned snapshot directory")
			}
			entries, e := os.ReadDir(dir)
			if e != nil {
				t.Fatal(e)
			}
			files := 0
			for i, row := range summary.Samples {
				if row.ID != i+1 || row.Offset != int64([]time.Duration{30 * time.Second, 75 * time.Second, 110 * time.Second}[i]) {
					t.Fatal("fixed ordered samples")
				}
				status, frames := row.Status, []policyTraceFrame{}
				if row.Status == "captured" || row.Status == "overlap" {
					files++
					raw := readPolicyTraceSnapshot(t, dir, row)
					status, frames = policyTraceFrames(raw)
					if row.Status == "overlap" {
						status = "overlap"
						frames = nil
					}
					if e = os.Remove(filepath.Join(dir, fmt.Sprintf("sample%d.stack", row.ID))); e != nil {
						t.Fatal("private stack removal", e)
					}
				} else if row.Status != "skipped" || row.Bytes != 0 || row.SHA != "" || summary.Return >= row.Offset {
					t.Fatal("honest early-return skip")
				}
				stack := struct {
					ID     int                `json:"id"`
					Status string             `json:"status"`
					Frames []policyTraceFrame `json:"frames"`
				}{row.ID, status, frames}
				if stack.Frames == nil {
					stack.Frames = []policyTraceFrame{}
				}
				encoded, e := json.Marshal(stack)
				if e != nil {
					t.Fatal(e)
				}
				fmt.Println("POLICY_STARTUP_ENVELOPE_STACK " + string(encoded))
			}
			check("snapshots_verified", files == len(entries))
			check("raw_removed", os.Remove(dir) == nil && !activationPathExists(dir))
			after := multiplePolicyReviewFiles(t, j)
			final, exists := after["execution.json"]
			var decoded nativeActivationRecord
			decodable := exists && json.Unmarshal(final, &decoded) == nil && decoded.Generation > 0 && (decoded.Owner.State == "closed" || decoded.Owner.State == "active")
			state := "invalid"
			generation := uint64(0)
			if decodable {
				state = decoded.Owner.State
				generation = decoded.Generation
			}
			journal := struct {
				Before           string `json:"before_sha256"`
				After            string `json:"after_sha256"`
				Unchanged        bool   `json:"unchanged"`
				Decoded          bool   `json:"decoded"`
				State            string `json:"state"`
				GenerationBefore uint64 `json:"generation_before"`
				GenerationAfter  uint64 `json:"generation_after"`
			}{fmt.Sprintf("%x", sha256.Sum256(before)), fmt.Sprintf("%x", sha256.Sum256(final)), bytes.Equal(before, final), decodable, state, seed.Generation, generation}
			encoded, e := json.Marshal(journal)
			if e != nil {
				t.Fatal(e)
			}
			fmt.Println("POLICY_STARTUP_ENVELOPE_JOURNAL " + string(encoded))
			var journalStat unix.Stat_t
			check("final_journal_observed", exists && len(final) > 0 && len(final) <= 64<<10 && unix.Lstat(filepath.Join(j, "execution.json"), &journalStat) == nil && journalStat.Uid == 0 && journalStat.Nlink == 1 && journalStat.Mode&07777 == 0600 && journalStat.Mode&unix.S_IFMT == unix.S_IFREG)
			check("ledger_preserved", reflect.DeepEqual(ledgerBefore, multiplePolicyReviewFiles(t, l)))
			delete(after, "execution.json")
			delete(journalBefore, "execution.json")
			check("nonjournal_preserved", reflect.DeepEqual(after, journalBefore))
			check("child_locks_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
		})
		check("domain_teardown", domains)
		check("mounts_restored", mounts)
		fmt.Println("POLICY_STARTUP_ENVELOPE_DOMAIN_TEARDOWN ufs filesystems=pass directories=pass mounts=pass")
	})
	check("parent_budget", budgetOK)
	t.Log("POLICY_STARTUP_ENVELOPE_UFS_RESULT " + strings.Join(seen, " "))
}
