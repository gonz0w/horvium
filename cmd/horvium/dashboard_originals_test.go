// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDashboardLaunchCopyRequiresOriginalArtifactBytes(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "source"), filepath.Join(dir, "target")
	data := []byte("original executable fixture")
	if os.WriteFile(source, data, 0700) != nil {
		t.Fatal("fixture executable")
	}
	expected := fmt.Sprintf("%x", sha256.Sum256(data))
	if err := dashboardCopyOriginalArtifact(source, target, expected); err != nil {
		t.Fatal(err)
	}
	copy, _ := os.ReadFile(target)
	if string(copy) != string(data) {
		t.Fatal("launch copy changed original bytes")
	}
	if err := dashboardCopyOriginalArtifact(source, target, expected); err == nil {
		t.Fatal("existing launch target overwritten")
	}
	if os.WriteFile(source, []byte("changed"), 0700) != nil {
		t.Fatal("mutation")
	}
	if err := dashboardCopyOriginalArtifact(source, filepath.Join(dir, "changed-target"), expected); err == nil {
		t.Fatal("changed artifact launched")
	}
}

func TestDashboardOriginalIsExclusiveAndRetainsOriginalBytes(t *testing.T) {
	dir := t.TempDir()
	r := dashboardOriginalFixture{Schema: 1, Mode: "synthetic-100", Fixture: dashboardFixtureReceipt{Datasets: 100, InventorySHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Origin: "generated"}}
	file, err := dashboardSaveOriginal(dir, "dashboard-original-synthetic-100.json", r)
	if err != nil || file.check() != nil {
		t.Fatal("original observation unavailable", err)
	}
	if _, err := dashboardSaveOriginal(dir, "dashboard-original-synthetic-100.json", r); err == nil {
		t.Fatal("duplicate declaration overwrote original")
	}
	if file.check() != nil {
		t.Fatal("duplicate attempt changed original")
	}
	if err := os.WriteFile(filepath.Join(dir, file.name), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if file.check() == nil {
		t.Fatal("changed original accepted")
	}
}

func TestDashboardOriginalRejectsLinksAndUnboundedInputs(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"../escape.json", "unknown.json", "dashboard-original-unknown.json"} {
		if _, err := dashboardSaveOriginal(dir, name, map[string]int{"schema": 1}); err == nil {
			t.Fatal("unknown original path accepted")
		}
	}
	name := "dashboard-original-native-small.json"
	out := filepath.Join(t.TempDir(), "outside")
	if os.WriteFile(out, []byte("outside"), 0600) != nil || os.Symlink(out, filepath.Join(dir, name)) != nil {
		t.Fatal("link fixture")
	}
	if _, err := dashboardSaveOriginal(dir, name, map[string]int{"schema": 1}); err == nil {
		t.Fatal("linked observation accepted")
	}
	data, _ := os.ReadFile(out)
	if string(data) != "outside" {
		t.Fatal("outside bytes changed")
	}
	large := make([]byte, 65537)
	if _, err := dashboardSaveOriginal(dir, "dashboard-original-projection-10000.json", large); err == nil {
		t.Fatal("unbounded original accepted")
	}
}

func TestDashboardReaperCensusRequiresOriginalOwnerAndNoDescendants(t *testing.T) {
	for _, row := range []struct {
		flags, children, descendants uint32
		owner, actual                int
		valid                        bool
	}{{1, 0, 0, 42, 42, true}, {0, 0, 0, 42, 42, false}, {1, 1, 0, 42, 42, false}, {1, 0, 1, 42, 42, false}, {1, 0, 0, 42, 43, false}, {1, 0, 0, 0, 0, false}} {
		if dashboardEmptyReaper(row.flags, row.children, row.descendants, row.owner, row.actual) != row.valid {
			t.Fatal("invalid original reaper census classification", row)
		}
	}
}
