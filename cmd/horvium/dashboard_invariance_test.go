// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

func TestDashboardValueChangesIdentifyOwnedAndOtherFields(t *testing.T) {
	original := adapterNativeInventory()
	original.Pools.Data = append(original.Pools.Data, inventory.Pool{Name: "private-system-pool", GUID: "9999", Size: 1000, Allocated: 200, Free: 700, Health: "ONLINE"})
	original.Datasets.Data[100].Name = "private-system-dataset"
	before, _ := inventory.CloneBounded(original, inventory.DefaultLimits())
	current, _ := inventory.CloneBounded(original, inventory.DefaultLimits())
	current.Pools.Data[0].Allocated++
	current.Pools.Data[1].Free--
	current.Datasets.Data[0].Used++
	current.Datasets.Data[100].Available--
	owned := append([]inventory.Dataset(nil), original.Datasets.Data[:100]...)
	got, err := dashboardValueChanges(original, current, original.Pools.Data[0].GUID, owned)
	want := map[string]map[string]int{"owned-pools": {"allocated": 1}, "other-pools": {"free": 1}, "owned-datasets": {"used": 1}, "other-datasets": {"available": 1}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("lost original ownership/column distinction", got, err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private") || strings.Contains(string(encoded), "9999") {
		t.Fatal("identity leaked")
	}
	if !reflect.DeepEqual(original, before) {
		t.Fatal("independent original mutated")
	}
	// Diagnostic comparison is not permission to rebaseline the HTML oracle.
	plan, err := dashboardPlan(len(original.Datasets.Data))
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := newDashboardHTMLPredicate(original, plan)
	if err != nil {
		t.Fatal(err)
	}
	h, _, cookie := dashboardHTMLFixture(t, current)
	response := dashboardHTMLRequest(h, cookie, "GET", "/", "")
	if err := predicate(plan[0], response.Code, response.Body.Bytes()); err == nil {
		t.Fatal("changed original accepted")
	}
}

func TestDashboardValueChangesRejectUncertainOwnershipAndIdentities(t *testing.T) {
	original := adapterNativeInventory()
	owned := original.Datasets.Data[:100]
	for _, kind := range []string{"unknown-pool", "unknown-owned-dataset", "duplicate-owned", "missing-dataset", "duplicate-current-guid", "unavailable"} {
		t.Run(kind, func(t *testing.T) {
			current, _ := inventory.CloneBounded(original, inventory.DefaultLimits())
			pool := original.Pools.Data[0].GUID
			witness := append([]inventory.Dataset(nil), owned...)
			switch kind {
			case "unknown-pool":
				pool = "999999"
			case "unknown-owned-dataset":
				witness[0].GUID = "999999"
			case "duplicate-owned":
				witness[1] = witness[0]
			case "missing-dataset":
				current.Datasets.Data = current.Datasets.Data[1:]
			case "duplicate-current-guid":
				current.Datasets.Data[1].GUID = current.Datasets.Data[0].GUID
			case "unavailable":
				current.Pools.Status = inventory.Unavailable
			}
			if out, err := dashboardValueChanges(original, current, pool, witness); err == nil || out != nil {
				t.Fatal("uncertain comparison accepted")
			}
		})
	}
}
