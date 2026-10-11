// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

// These are portable checks of supplied facts and complete decoded catalogs.
// They do not exercise original leases, native producers or startup publication.
func TestExecutionStartupPolicyFactsDrift(t *testing.T) {
	for _, mode := range []string{
		"pool-name", "scope-name", "scope-parent", "foreign-scope", "empty-scope",
		"missing-member", "foreign-vdev", "serial-source", "media-size", "sector-size",
		"stripe-size", "stripe-offset", "duplicate-provider", "provider-serial",
		"provider-geometry", "missing-imported-pool", "imported-name", "system-topology",
		"system-provider", "system-mount", "system-swap", "other-pool-overlap",
	} {
		t.Run(mode, func(t *testing.T) {
			p, o, disks, use := enrollmentFixture(t)
			if err := checkExecutionStartupPolicyFacts(p, o, disks, use); err != nil {
				t.Fatal("positive control", err)
			}
			switch mode {
			case "pool-name":
				p.PoolName = "renamed"
			case "scope-name":
				p.Scopes[0].Name = "tank/renamed"
			case "scope-parent":
				p.Scopes[0].ParentGUID = "999"
			case "foreign-scope":
				p.Scopes[0].GUID = "999"
			case "empty-scope":
				p.Scopes = nil
			case "missing-member":
				p.Members = p.Members[:1]
			case "foreign-vdev":
				p.Members[0].VdevGUID = "999"
			case "serial-source":
				p.Members[0].SerialSource = "provider.config.ident"
			case "media-size":
				p.Members[0].MediaSize *= 2
			case "sector-size":
				p.Members[0].SectorSize *= 2
			case "stripe-size":
				p.Members[0].StripeSize += 512
			case "stripe-offset":
				p.Members[0].StripeOffset += 512
			case "duplicate-provider":
				disks[1] = disks[0]
			case "provider-serial":
				disks[0].Provider.Identifiers = nil
			case "provider-geometry":
				disks[0].SectorCount++
			case "missing-imported-pool":
				use.pools = nil
			case "imported-name":
				use.pools[0].Name = "renamed"
			case "system-topology":
				use.topologies = nil
			case "system-provider":
				use.pools[0].Disks[0].MediaSize *= 2
			case "system-mount":
				use.system.Mounts = append(use.system.Mounts, MountUse{Source: "tank/data", Target: "/var/db", Type: "zfs", PoolGUID: "100", Disks: use.pools[0].Disks})
			case "system-swap":
				use.system.Swap = append(use.system.Swap, SwapUse{Source: "/dev/ada0", Disks: use.pools[0].Disks})
			case "other-pool-overlap":
				use.pools = append(use.pools, ImportedPoolUse{Name: "other", GUID: "200", Health: "ONLINE", Disks: use.pools[0].Disks})
			}
			before := policyAdmissionFactsSnapshot(t, p, o, disks, use)
			if err := checkExecutionStartupPolicyFacts(p, o, disks, use); err == nil {
				t.Fatal("unsafe facts accepted")
			}
			if !bytes.Equal(before, policyAdmissionFactsSnapshot(t, p, o, disks, use)) {
				t.Fatal("denial repaired facts or changed policy/eligibility")
			}
		})
	}
}

func TestExecutionStartupPolicyFactsEquivalentCaptures(t *testing.T) {
	p, o, disks, use := enrollmentFixture(t)
	// Record ordering and graph-local IDs are not persistent identity. Ordinary
	// data mounts are permitted; accepting them must never set Eligible.
	p.Members[0], p.Members[1] = p.Members[1], p.Members[0]
	disks[0], disks[1] = disks[1], disks[0]
	for i := range disks {
		disks[i].Provider.ID = fmt.Sprintf("fresh-%d", i)
		disks[i].Provider.GeomID = "fresh-geom"
	}
	use.system.Mounts = append(use.system.Mounts, MountUse{Source: "tank/data", Target: "/srv/data", Type: "zfs", PoolGUID: "100", Disks: use.pools[0].Disks})
	before := policyAdmissionFactsSnapshot(t, p, o, disks, use)
	if err := checkExecutionStartupPolicyFacts(p, o, disks, use); err != nil {
		t.Fatal("equivalent complete facts refused", err)
	}
	if o.Eligible || !bytes.Equal(before, policyAdmissionFactsSnapshot(t, p, o, disks, use)) {
		t.Fatal("acceptance changed facts or granted eligibility")
	}
}

func policyAdmissionFactsSnapshot(t *testing.T, p Policy, o Observation, disks []inventory.DiskObservation, use PoolSystemUse) []byte {
	t.Helper()
	data, err := json.Marshal(struct {
		Policy      Policy
		Observation Observation
		Disks       []inventory.DiskObservation
		Initialized bool
		System      SystemDeviceUse
		Pools       []ImportedPoolUse
		Topologies  []PoolTopology
	}{p, o, disks, use.initialized, use.system, use.pools, use.topologies})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func policyAdmissionJob(t *testing.T, format string, state JobState, reason string, n int, otherPool bool) (Job, []byte) {
	t.Helper()
	j := catalogJob(n, false)
	property := propertyJobFixture(t)
	intent := property.Intent()
	if format == "property" {
		// Distinct synthetic review metadata, not captured admission authority.
		intent.review.ID = fmt.Sprintf("%064x", n+4000)
		j = property.Job()
		j.ID = fmt.Sprintf("%064x", n)
		j.Intent.SubmissionID = fmt.Sprintf("%064x", n+2000)
		j.Intent.PlanID = intent.review.ID
	} else if otherPool {
		j.Intent.PoolGUID = "200"
	}
	j = restartJobFixture(t, j, state, JobQueued, reason)
	var data []byte
	var err error
	if format == "property" {
		bound, bindErr := BindPropertyJob(j, intent)
		if bindErr != nil {
			t.Fatal(bindErr)
		}
		data, err = EncodePropertyJob(bound)
	} else {
		data, err = EncodeJobRecord(j)
	}
	if err != nil {
		t.Fatal(err)
	}
	return j, data
}

func TestExecutionStartupPolicyCatalogEncodedHistory(t *testing.T) {
	states := []struct {
		state  JobState
		reason string
	}{{JobQueued, "admitted"}, {JobApplying, "launch-intent"}, {JobObserving, "native-observation"}, {JobSucceeded, "verified"}, {JobFailed, "verified-failure"}, {JobCancelled, "operator-cancelled"}, {JobReconcileRequired, "review-expired"}}
	for _, relation := range []string{"same-pool", "other-pool", "mixed"} {
		t.Run(relation, func(t *testing.T) {
			builder := newStateCatalogBuilder(defaultStateCatalogLimits)
			p, _, _, _ := enrollmentFixture(t)
			if relation == "other-pool" {
				p.PoolGUID = "200"
			}
			owner, err := EncodeOwnershipRecord(p)
			if err != nil {
				t.Fatal(err)
			}
			if err = builder.add(stateRecordName{poolGUID: p.PoolGUID}, owner); err != nil {
				t.Fatal(err)
			}
			if relation == "mixed" {
				p.PoolGUID = "200"
				second, err := EncodeOwnershipRecord(p)
				if err != nil {
					t.Fatal(err)
				}
				if err = builder.add(stateRecordName{poolGUID: p.PoolGUID}, second); err != nil {
					t.Fatal(err)
				}
			}
			if err = checkExecutionStartupPolicyCatalog(builder.catalog); err != nil {
				t.Fatal("valid job-free codec catalog", err)
			}
			var records [][]byte
			for f, format := range []string{"legacy", "property"} {
				for s, tc := range states {
					t.Run(format+"/"+string(tc.state), func(t *testing.T) {
						j, data := policyAdmissionJob(t, format, tc.state, tc.reason, f*len(states)+s+1, relation == "mixed")
						current := newStateCatalogBuilder(defaultStateCatalogLimits)
						if relation == "mixed" {
							current = builder
						} else if err := current.add(stateRecordName{poolGUID: p.PoolGUID}, owner); err != nil {
							t.Fatal(err)
						}
						if err := current.add(stateRecordName{jobID: j.ID}, data); err != nil {
							t.Fatal("valid encoded history", err)
						}
						records = append(records, data)
						if relation != "mixed" {
							records = [][]byte{data}
						}
						holds, err := BuildJobReservations(records)
						if err != nil {
							t.Fatal("valid complete retained holds", err)
						}
						before, err := json.Marshal(current.catalog)
						if err != nil {
							t.Fatal(err)
						}
						encoded, err := json.Marshal(records)
						if err != nil {
							t.Fatal(err)
						}
						if !errors.Is(checkExecutionStartupPolicyCatalog(current.catalog), ErrDenied) {
							t.Fatal("retained history allowed startup")
						}
						after, err := json.Marshal(current.catalog)
						if err != nil || !bytes.Equal(before, after) {
							t.Fatal("refusal changed catalog or encoded history", err)
						}
						retained, err := json.Marshal(records)
						if err != nil || !bytes.Equal(encoded, retained) {
							t.Fatal("refusal changed earlier encoded history", err)
						}
						again, err := BuildJobReservations(records)
						if err != nil || !reflect.DeepEqual(holds.Pools(), again.Pools()) {
							t.Fatal("refusal released retained holds", err)
						}
						count := 0
						for _, pool := range again.Pools() {
							count += len(pool.Jobs)
						}
						if count != len(records) {
							t.Fatal("refusal omitted holders", count)
						}
						last := current.catalog.Jobs[len(current.catalog.Jobs)-1]
						if last.ID != j.ID || last.SubmissionID != j.Intent.SubmissionID || last.State != tc.state || last.Revision != uint64(len(j.History)) || last.Bytes != uint64(len(data)) {
							t.Fatal("history omitted or projected incorrectly", last)
						}
					})
				}
			}
			if relation == "mixed" && (len(builder.catalog.Ownership) != 2 || len(builder.catalog.Jobs) != 14 || builder.catalog.Nonterminal != 8 || builder.catalog.Terminal != 6) {
				t.Fatal("mixed catalog omitted a policy, format or state", builder.catalog)
			}
			if relation == "mixed" {
				holds, err := BuildJobReservations(records)
				if err != nil || len(holds.Pools()) != 2 || len(holds.Pools()[0].Jobs) != 7 || len(holds.Pools()[1].Jobs) != 7 {
					t.Fatal("mixed pool histories incomplete", err)
				}
			}
		})
	}
}
