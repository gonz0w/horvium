// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

// Native storage/census/publication are real; these ownership, boot and
// physical-use transfers remain synthetic. Fixed-root policy facts are separate.
func nativePolicyHistoryCases(t *testing.T, f restoreFixture, check func(string, bool)) {
	t.Helper()
	states := []struct {
		state  JobState
		reason string
	}{{JobQueued, "admitted"}, {JobApplying, "launch-intent"}, {JobObserving, "native-observation"}, {JobSucceeded, "verified"}, {JobFailed, "verified-failure"}, {JobCancelled, "operator-cancelled"}, {JobReconcileRequired, "review-expired"}}
	for _, relation := range []string{"same-pool", "other-pool", "mixed"} {
		p, o, disks, use := policyStorageInputs(t, f)
		other, _, _, _ := enrollmentFixture(t)
		other.PoolGUID, other.PoolName = "200", "vault"
		other.Scopes[0].GUID, other.Scopes[0].Name, other.Scopes[0].ParentGUID = "202", "vault/data", "201"
		for i := range other.Members {
			other.Members[i].Serial = "OTHER-" + other.Members[i].Serial
		}
		policies := []Policy{p}
		if relation == "other-pool" {
			policies = []Policy{other}
		} else if relation == "mixed" {
			policies = append(policies, other)
		}
		// Resolve the other owned pool only in this synthetic system-use transfer;
		// no physical/native policy eligibility is established by a denial seed.
		if relation != "same-pool" {
			use.pools = append(use.pools, ImportedPoolUse{Name: "vault", GUID: "200", Health: "ONLINE", Disks: []inventory.Provider{{ID: "history-data", Name: "history-data"}}})
		}
		want := newStateCatalogBuilder(defaultStateCatalogLimits)
		for _, policy := range policies {
			data, err := EncodeOwnershipRecord(policy)
			if err != nil {
				t.Fatal(err)
			}
			if err = want.add(stateRecordName{poolGUID: policy.PoolGUID}, data); err != nil {
				t.Fatal(err)
			}
		}
		var records [][]byte
		for formatIndex, format := range []string{"legacy", "property"} {
			for stateIndex, tc := range states {
				job, data := policyAdmissionJob(t, format, tc.state, tc.reason, formatIndex*len(states)+stateIndex+1, relation == "mixed")
				if err := want.add(stateRecordName{jobID: job.ID}, data); err != nil {
					t.Fatal(err)
				}
				records = append(records, data)
			}
		}
		wantedHolds, err := BuildJobReservations(records)
		if err != nil {
			t.Fatal("valid complete expected holders", err)
		}
		j, l := policyHistoryStorageSeed(t, f, "history-"+relation, policies, records)
		ctx := context.Background()
		readback := func() (StateCatalog, JobReservations, error) {
			pair, err := OpenExecutionDomainLeases(ctx, j, l)
			if err != nil {
				return StateCatalog{}, JobReservations{}, err
			}
			catalog, ce := pair.native.ledger.InspectState()
			holds, he := pair.native.ledger.InspectJobReservations()
			return catalog, holds, errors.Join(ce, he, pair.Close())
		}
		catalog, originalHolds, err := readback()
		mode := "history-" + relation
		// A typed denial alone could pass for an unrelated backing/journal failure.
		// Require the ordinary metadata path to accept the same complete domains,
		// boot and system inputs before testing the separate all-job refusal gate.
		metadata, _, metadataErr := reviewExecutionStartup(ctx, j, l, f.read, func(context.Context) (PoolSystemUse, error) { return use, nil }, activationBoot)
		metadataValid := metadataErr == nil && metadata != nil
		if metadata != nil {
			metadataValid = metadata.Close() == nil && metadataValid
		}
		check(mode+"_catalog", err == nil && metadataValid && reflect.DeepEqual(catalog, want.catalog) && len(catalog.Jobs) == 14 && catalog.Nonterminal == 8 && catalog.Terminal == 6)
		if !reflect.DeepEqual(originalHolds.Pools(), wantedHolds.Pools()) {
			t.Fatal("seed omitted a retained holder")
		}
		beforeJ, beforeL := freshNamespaceSnapshot(t, j), freshNamespaceSnapshot(t, l)
		calls, bootCalls, leasesHeld := 0, 0, true
		source := func(context.Context, Policy) (Observation, []inventory.DiskObservation, PoolSystemUse, error) {
			calls++
			return o, disks, use, nil
		}
		boot := func(c context.Context) ([16]byte, error) {
			bootCalls++
			leasesHeld = restartLocksHeld(t, j, l) && leasesHeld
			return activationBoot(c)
		}
		attempt := func(c context.Context) bool {
			r, n, err := reviewExecutionStartupPolicy(c, j, l, f.read, func(context.Context) (PoolSystemUse, error) { return use, nil }, boot, source)
			if r != nil {
				defer r.Close()
			}
			return errors.Is(err, ErrDenied) && r == nil && n == nil && calls == 0
		}
		denied := attempt(ctx)
		check(mode+"_leases-held", bootCalls > 0 && leasesHeld)
		check(mode+"_denied", denied)
		check(mode+"_preserved", reflect.DeepEqual(beforeJ, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(beforeL, freshNamespaceSnapshot(t, l)))
		check(mode+"_released", restartLocksReleased(t, j, l))
		again, holds, err := readback()
		check(mode+"_holds", err == nil && reflect.DeepEqual(originalHolds.Pools(), holds.Pools()) && reflect.DeepEqual(wantedHolds.Pools(), holds.Pools()))
		check(mode+"_reopened", err == nil && reflect.DeepEqual(catalog, again) && restartLocksReleased(t, j, l))
		previousBootCalls := bootCalls
		retryCtx, cancel := context.WithCancel(ctx)
		retry := attempt(retryCtx)
		cancel()
		check(mode+"_retry", retry && bootCalls > previousBootCalls && leasesHeld && restartLocksReleased(t, j, l) && reflect.DeepEqual(beforeJ, freshNamespaceSnapshot(t, j)) && reflect.DeepEqual(beforeL, freshNamespaceSnapshot(t, l)))
	}
}
