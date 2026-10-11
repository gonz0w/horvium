// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

// Remap only synthetic fixture transfers, not encoded ownership/state files.
// Independent GUIDs, names, serials and provider paths must remain consistent
// throughout every supplied observation, including descendants/topology.
func secondPolicyFixture(t *testing.T) (Policy, Observation, []inventory.DiskObservation) {
	t.Helper()
	p, o, d, _ := enrollmentFixture(t)
	fixture := struct {
		Policy      Policy
		Observation Observation
		Disks       []inventory.DiskObservation
	}{p, o, d}
	data, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	remap := strings.NewReplacer(`"100"`, `"200"`, `"101"`, `"201"`, `"102"`, `"202"`, `"900"`, `"1900"`, `"901"`, `"1901"`, `"902"`, `"1902"`, "tank", "vault", "SERIAL-A", "OTHER-A", "SERIAL-B", "OTHER-B", "ada0", "ada2", "ada1", "ada3")
	if err := json.Unmarshal([]byte(remap.Replace(string(data))), &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture.Policy, fixture.Observation, fixture.Disks
}

func TestMultiplePolicySuppliedFactsRemainDistinctAndValid(t *testing.T) {
	p, o, d, use := enrollmentFixture(t)
	q, other, disks := secondPolicyFixture(t)
	use.pools = append(use.pools, ImportedPoolUse{Name: q.PoolName, GUID: q.PoolGUID, Health: "ONLINE", Disks: []inventory.Provider{disks[0].Provider, disks[1].Provider}})
	use.topologies = append(use.topologies, other.Topology)
	if q.PoolGUID != "200" || q.Scopes[0].GUID != "202" || q.Scopes[0].ParentGUID != "201" || q.Members[0].Serial == p.Members[0].Serial || disks[0].Provider.ID == d[0].Provider.ID {
		t.Fatal("aliased second policy")
	}
	if err := checkExecutionStartupPolicyFacts(p, o, d, use); err != nil {
		t.Fatal("first facts", err)
	}
	if err := checkExecutionStartupPolicyFacts(q, other, disks, use); err != nil {
		t.Fatal("second facts", err)
	}
	other.Filesystems[1].GUID = "999"
	if checkExecutionStartupPolicyFacts(q, other, disks, use) == nil {
		t.Fatal("last scope drift accepted")
	}
}
