// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"testing"
)

func TestPropertyPolicyBindingNeverGrantsEligibility(t *testing.T) {
	p, o, i := fixture()
	o.Eligible = false
	if err := ValidatePropertyPolicyBinding(p, o); err != nil {
		t.Fatal(err)
	}
	if o.Eligible {
		t.Fatal("binding granted eligibility")
	}
	if _, err := Build(instant, "session", p, o, i); !errors.Is(err, ErrUnavailable) {
		t.Fatal("binding bypassed planner", err)
	}
}

func TestPropertyPolicyBindingChecksCompleteGraphAndEveryScope(t *testing.T) {
	for _, change := range []string{"partial", "pool", "scope", "parent", "unselected-source", "identity", "state", "feature", "topology", "quota", "empty"} {
		t.Run(change, func(t *testing.T) {
			p, o, _ := fixture()
			o.Eligible = false
			switch change {
			case "partial":
				o.Complete = false
			case "pool":
				p.PoolGUID = "200"
			case "scope":
				p.Scopes[1].GUID = "999"
			case "parent":
				p.Scopes[1].ParentGUID = "999"
			case "unselected-source":
				p.Scopes = p.Scopes[:1]
				o.Filesystems[1].Properties.Compression.SourceGUID = "999"
			case "identity":
				o.Identities = o.Identities[:1]
			case "state":
				o.States[1].GUID = "999"
			case "feature":
				o.Features.Values = o.Features.Values[:1]
			case "topology":
				o.Topology.Nodes = nil
			case "quota":
				o.Filesystems[1].Properties.Quota.Effective = "arbitrary"
			case "empty":
				p.Scopes = nil
			}
			if err := ValidatePropertyPolicyBinding(p, o); err == nil {
				t.Fatal("bad binding accepted")
			}
		})
	}
}
