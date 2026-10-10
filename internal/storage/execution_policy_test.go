// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"reflect"
	"runtime"
	"testing"
)

func TestExecutionStartupPolicyFacts(t *testing.T) {
	p, o, d, u := enrollmentFixture(t)
	before, _ := EncodeOwnershipRecord(p)
	if e := checkExecutionStartupPolicyFacts(p, o, d, u); e != nil {
		t.Fatal("valid policy facts", e)
	}
	after, _ := EncodeOwnershipRecord(p)
	if !reflect.DeepEqual(before, after) || o.Eligible {
		t.Fatal("facts changed authority or policy")
	}
	for _, mode := range []string{"legacy", "scope", "pool", "member", "provider", "missing-system", "system-overlap", "incomplete", "eligible"} {
		t.Run(mode, func(t *testing.T) {
			p, o, d, u := enrollmentFixture(t)
			switch mode {
			case "legacy":
				p.Members = nil
			case "scope":
				p.Scopes[0].GUID = "999"
			case "pool":
				o.PoolGUID = "999"
			case "member":
				p.Members[0].Serial = "OTHER"
			case "provider":
				d = d[:1]
			case "missing-system":
				u = PoolSystemUse{}
			case "system-overlap":
				u = propertySystemFixture(t, "tank/data / zfs rw 0 0\n", emptySwap)
			case "incomplete":
				o.Complete = false
			case "eligible":
				o.Eligible = true
			}
			if checkExecutionStartupPolicyFacts(p, o, d, u) == nil {
				t.Fatal("unsafe policy facts")
			}
		})
	}
}
func TestExecutionStartupPolicyCatalog(t *testing.T) {
	for _, c := range []StateCatalog{{Ownership: []OwnershipSummary{}}, {Ownership: []OwnershipSummary{{PoolGUID: "100", Revision: 1, Bytes: 64}}, Bytes: 64}} {
		if e := checkExecutionStartupPolicyCatalog(c); e != nil {
			t.Fatal("job-free catalog", e)
		}
	}
	for _, state := range []JobState{JobQueued, JobApplying, JobObserving, JobReconcileRequired, JobSucceeded, JobFailed, JobCancelled} {
		t.Run(string(state), func(t *testing.T) {
			c := StateCatalog{Jobs: []JobSummary{{ID: "retained", State: state}}}
			if checkExecutionStartupPolicyCatalog(c) == nil {
				t.Fatal("retained job accepted")
			}
		})
	}
	for _, c := range []StateCatalog{{Nonterminal: 1}, {Terminal: 1}} {
		if checkExecutionStartupPolicyCatalog(c) == nil {
			t.Fatal("counts bypassed job check")
		}
	}
}
func TestExecutionPolicyStartupUnavailable(t *testing.T) {
	if runtime.GOOS == "freebsd" {
		t.Skip("portable unsupported-host assertion")
	}
	r, e := ReviewExecutionPolicyStartup(context.Background(), "unused-journal", "unused-ledger", nil, nil, nil, nil)
	if e == nil || r != nil {
		t.Fatal("unsupported startup did filesystem work or granted review")
	}
}
