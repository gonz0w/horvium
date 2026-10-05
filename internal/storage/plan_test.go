// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

var instant = time.Date(2026, 10, 4, 23, 0, 0, 0, time.UTC)

func fixture() (Policy, Observation, Intent) {
	props := Properties{Quota: Value{"none", Default, "", ""}, RefQuota: Value{"none", Default, "", ""}, Compression: Value{"lz4", Local, "101", ""}}
	root := Filesystem{GUID: "101", Name: "tank", Properties: props}
	child := Filesystem{GUID: "102", Name: "tank/data", ParentGUID: "101", Properties: Properties{Quota: Value{"none", Default, "", ""}, RefQuota: Value{"none", Default, "", ""}, Compression: Value{"lz4", Inherited, "101", ""}}}
	obs := Observation{Complete: true, PoolGUID: "100", PoolName: "tank", Eligible: true, Filesystems: []Filesystem{root, child}}
	obs.States = []FilesystemState{stateFixture(root), stateFixture(child)}
	obs.Identities = []DatasetIdentity{{GUID: "101", Name: "tank", Type: FilesystemType}, {GUID: "102", Name: "tank/data", Type: FilesystemType}}
	policy := Policy{Revision: 1, PoolGUID: "100", PoolName: "tank", Scopes: []Scope{{GUID: "101", Name: "tank", Properties: []Property{Quota, RefQuota, Compression}}, {GUID: "102", Name: "tank/data", ParentGUID: "101", Properties: []Property{Quota, Compression}}}}
	return policy, obs, Intent{FilesystemGUID: "101", Property: Quota, Value: "1099511627776"}
}

func TestReviewScopeAndImpact(t *testing.T) {
	policy, obs, intent := fixture()
	plan, err := Build(instant, "session-a", policy, obs, intent)
	if err != nil {
		t.Fatal(err)
	}
	review := plan.Review()
	if review.Intent != intent || len(review.Affected) != 2 || review.Affected[0].GUID != "101" || review.Affected[1].GUID != "102" || len(review.ID) != 64 || !review.ExpiresAt.Equal(instant.Add(5*time.Minute)) {
		t.Fatalf("unexpected review: %+v", review)
	}
	if err := plan.Validate(instant.Add(time.Minute), "session-a", policy, obs); err != nil {
		t.Fatal(err)
	}
	second, err := Build(instant, "session-a", policy, obs, intent)
	if err != nil || second.Review().ID == review.ID {
		t.Fatal("plan identifiers must be independent")
	}
}

func TestObservationNeverEnrollsResources(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Policy, *Observation)
	}{
		{"empty policy", func(p *Policy, _ *Observation) { *p = Policy{} }},
		{"other pool", func(p *Policy, _ *Observation) { p.PoolGUID = "999" }},
		{"unselected target", func(p *Policy, _ *Observation) { p.Scopes = p.Scopes[1:] }},
		{"property outside scope", func(p *Policy, _ *Observation) { p.Scopes[0].Properties = []Property{Compression} }},
		{"foreign child", func(p *Policy, _ *Observation) { p.Scopes = p.Scopes[:1] }},
		{"child property outside scope", func(p *Policy, _ *Observation) { p.Scopes[1].Properties = []Property{Compression} }},
		{"unknown eligibility", func(_ *Policy, o *Observation) { o.Eligible = false }},
		{"paged collection", func(_ *Policy, o *Observation) { o.Complete = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, o, i := fixture()
			tc.change(&p, &o)
			if _, err := Build(instant, "session-a", p, o, i); err == nil {
				t.Fatal("unqualified scope granted a plan")
			}
		})
	}
}

func TestCompressionImpactRespectsOverrides(t *testing.T) {
	p, o, i := fixture()
	i.Property = Compression
	i.Value = "off"
	p.Scopes = p.Scopes[:1]
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrDenied) {
		t.Fatalf("foreign inheritor: %v", err)
	}
	o.Filesystems[1].Properties.Compression = Value{"off", Local, "102", ""}
	grandchild := o.Filesystems[1]
	grandchild.GUID = "103"
	grandchild.Name = "tank/data/archive"
	grandchild.ParentGUID = "102"
	grandchild.Properties.Compression = Value{"off", Inherited, "102", ""}
	o.Filesystems = append(o.Filesystems, grandchild)
	o.States = append(o.States, stateFixture(grandchild))
	o.Identities = append(o.Identities, DatasetIdentity{GUID: grandchild.GUID, Name: grandchild.Name, Type: FilesystemType})
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Review().Affected) != 1 {
		t.Fatal("local override must protect its branch")
	}
	o.Filesystems[1].Properties.Compression = Value{"off", Received, "102", "off"}
	if _, err := Build(instant, "session-a", p, o, i); err != nil {
		t.Fatal(err)
	}
}

func TestReferenceQuotaDoesNotEnrollDescendants(t *testing.T) {
	p, o, i := fixture()
	p.Scopes = p.Scopes[:1]
	i.Property = RefQuota
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Review().Affected) != 1 {
		t.Fatal("reference quota affects only its filesystem")
	}
}

func TestLimitsRemainLiteralAndExplicit(t *testing.T) {
	for _, prop := range []Property{Quota, RefQuota} {
		for _, value := range []string{"none", "1", "18446744073709551615"} {
			p, o, i := fixture()
			i.Property = prop
			i.Value = value
			if _, err := Build(instant, "session-a", p, o, i); err != nil {
				t.Fatalf("%s=%s: %v", prop, value, err)
			}
		}
		for _, value := range []string{"", "0", "00", "01", "-1", "+1", "1G", "1.5", " 1", "18446744073709551616", "NONE"} {
			p, o, i := fixture()
			i.Property = prop
			i.Value = value
			if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrInvalid) {
				t.Fatalf("accepted %s=%q: %v", prop, value, err)
			}
		}
	}
	for _, value := range []string{"off", "lz4", "on", "gzip", "inherit", "-o", "lz4\n"} {
		p, o, i := fixture()
		i.Property = Compression
		i.Value = value
		_, err := Build(instant, "session-a", p, o, i)
		if (value == "off" || value == "lz4") != (err == nil) {
			t.Fatalf("compression %q: %v", value, err)
		}
	}
	p, o, i := fixture()
	i.Property = "mountpoint"
	i.Value = "/tmp"
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}

func TestPlansRequireFreshReviewOnDrift(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Policy, *Observation)
	}{
		{"policy revision", func(p *Policy, _ *Observation) { p.Revision++ }},
		{"scope revocation", func(p *Policy, _ *Observation) { p.Scopes = p.Scopes[:1] }},
		{"renamed pool", func(_ *Policy, o *Observation) {
			for n := range o.Identities {
				o.Identities[n].Name = strings.Replace(o.Identities[n].Name, "tank", "renamed", 1)
			}
			o.PoolName = "renamed"
		}},
		{"reused name", func(_ *Policy, o *Observation) { o.Filesystems[0].GUID = "999" }},
		{"renamed dataset", func(_ *Policy, o *Observation) { o.Filesystems[1].Name = "tank/new" }},
		{"changed value", func(_ *Policy, o *Observation) { o.Filesystems[0].Properties.Quota = Value{"9", Local, "101", ""} }},
		{"changed source", func(_ *Policy, o *Observation) { o.Filesystems[0].Properties.Compression.Source = Received }},
		{"foreign descendant", func(_ *Policy, o *Observation) {
			f := o.Filesystems[1]
			f.GUID = "103"
			f.Name = "tank/foreign"
			o.Filesystems = append(o.Filesystems, f)
			o.States = append(o.States, stateFixture(f))
		}},
		{"pool missing", func(_ *Policy, o *Observation) { o.PoolGUID = "" }},
		{"partial reread", func(_ *Policy, o *Observation) { o.Complete = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, o, i := fixture()
			plan, err := Build(instant, "session-a", p, o, i)
			if err != nil {
				t.Fatal(err)
			}
			tc.change(&p, &o)
			if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
				t.Fatalf("stale plan: %v", err)
			}
		})
	}
}

func TestPlanContextAndExpiry(t *testing.T) {
	p, o, i := fixture()
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Time{instant.Add(-time.Nanosecond), instant.Add(5 * time.Minute), instant.Add(time.Hour)} {
		if err := plan.Validate(at, "session-a", p, o); !errors.Is(err, ErrExpired) {
			t.Fatal(err)
		}
	}
	if err := plan.Validate(instant, "session-b", p, o); !errors.Is(err, ErrDenied) {
		t.Fatal(err)
	}
	if err := (Plan{}).Validate(instant, "session-a", p, o); err == nil {
		t.Fatal("zero plan accepted")
	}
	for _, session := range []string{"", strings.Repeat("a", 65), "a\nb"} {
		if _, err := Build(instant, session, p, o, i); !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}
}

func TestPlanOwnsReviewAndInputCopies(t *testing.T) {
	p, o, i := fixture()
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	original := plan.Review()
	view := plan.Review()
	view.Affected[0].Name = "altered"
	view.Context[0].Properties.Quota.Effective = "5"
	p.Scopes[0].Properties[0] = Compression
	o.Filesystems[0].Name = "altered"
	if !reflect.DeepEqual(original, plan.Review()) {
		t.Fatal("caller changed immutable review")
	}
	p, o, _ = fixture()
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedObservationsAndPolicyFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Policy, *Observation)
	}{
		{"duplicate guid", func(_ *Policy, o *Observation) { o.Filesystems[1].GUID = "101" }},
		{"duplicate name", func(_ *Policy, o *Observation) { o.Filesystems[1].Name = "tank" }},
		{"missing parent", func(_ *Policy, o *Observation) { o.Filesystems[1].ParentGUID = "999" }},
		{"false parent", func(_ *Policy, o *Observation) { o.Filesystems[1].Name = "tank/missing/data" }},
		{"bad guid", func(_ *Policy, o *Observation) { o.PoolGUID = "0100" }},
		{"overflow guid", func(_ *Policy, o *Observation) { o.Filesystems[1].GUID = "18446744073709551616" }},
		{"source outside ancestors", func(_ *Policy, o *Observation) { o.Filesystems[1].Properties.Compression.SourceGUID = "102" }},
		{"inconsistent inheritance", func(_ *Policy, o *Observation) { o.Filesystems[1].Properties.Compression.Effective = "off" }},
		{"unknown source", func(_ *Policy, o *Observation) { o.Filesystems[1].Properties.Compression.Source = "unknown" }},
		{"inherited quota", func(_ *Policy, o *Observation) {
			o.Filesystems[1].Properties.Quota = Value{"none", Inherited, "101", ""}
		}},
		{"duplicate scope", func(p *Policy, _ *Observation) { p.Scopes = append(p.Scopes, p.Scopes[0]) }},
		{"duplicate property", func(p *Policy, _ *Observation) { p.Scopes[0].Properties = append(p.Scopes[0].Properties, Quota) }},
		{"arbitrary property scope", func(p *Policy, _ *Observation) { p.Scopes[0].Properties = append(p.Scopes[0].Properties, "mountpoint") }},
		{"unsafe name", func(_ *Policy, o *Observation) { o.Filesystems[1].Name = "tank/-option" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, o, i := fixture()
			tc.change(&p, &o)
			if _, err := Build(instant, "session-a", p, o, i); err == nil {
				t.Fatal("malformed state accepted")
			}
		})
	}
}

func TestOrderingDoesNotCauseDrift(t *testing.T) {
	p, o, i := fixture()
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	o.Filesystems[0], o.Filesystems[1] = o.Filesystems[1], o.Filesystems[0]
	p.Scopes[0], p.Scopes[1] = p.Scopes[1], p.Scopes[0]
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal(err)
	}
}

func TestObservationAndPlanBudgets(t *testing.T) {
	p, o, i := fixture()
	o.Filesystems = make([]Filesystem, 100001)
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	p, o, i = fixture()
	o.Filesystems[1].Name = strings.Repeat("a", 4097)
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	// A valid large pool must fail review budget rather than omit impact rows.
	p, o, i = fixture()
	i.Property = RefQuota
	for n := 0; n < 300; n++ {
		f := o.Filesystems[1]
		f.GUID = strconv.FormatUint(uint64(1000+n), 10)
		f.Name = "tank/child" + f.GUID
		o.Filesystems = append(o.Filesystems, f)
		o.States = append(o.States, stateFixture(f))
		o.Identities = append(o.Identities, DatasetIdentity{GUID: f.GUID, Name: f.Name, Type: FilesystemType})
	}
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
}

func TestNewBaselinePreservesGUIDAfterRename(t *testing.T) {
	p, o, i := fixture()
	old, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	for n := range o.Filesystems {
		o.Filesystems[n].Name = strings.Replace(o.Filesystems[n].Name, "tank", "renamed", 1)
	}
	for n := range o.Identities {
		o.Identities[n].Name = strings.Replace(o.Identities[n].Name, "tank", "renamed", 1)
	}
	for n := range o.States {
		o.States[n].Name = strings.Replace(o.States[n].Name, "tank", "renamed", 1)
	}
	o.PoolName = "renamed"
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrDenied) {
		t.Fatal("rename automatically enrolled", err)
	}
	p.Revision++
	p.PoolName = "renamed"
	for n := range p.Scopes {
		p.Scopes[n].Name = strings.Replace(p.Scopes[n].Name, "tank", "renamed", 1)
	}
	fresh, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Review().Intent.FilesystemGUID != "101" || fresh.Review().PoolGUID != "100" {
		t.Fatal("rename changed identity")
	}
	if err := old.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
		t.Fatal(err)
	}
}

func TestAncestorConstraintsAndUnrelatedSibling(t *testing.T) {
	p, o, i := fixture()
	i.FilesystemGUID = "102"
	i.Property = RefQuota
	p.Scopes[1].Properties = append(p.Scopes[1].Properties, RefQuota)
	o.Filesystems[0].Properties.Quota = Value{"4096", Local, "101", ""}
	sibling := o.Filesystems[1]
	sibling.GUID = "103"
	sibling.Name = "tank/sibling"
	o.Filesystems = append(o.Filesystems, sibling)
	o.States = append(o.States, stateFixture(sibling))
	o.Identities = append(o.Identities, DatasetIdentity{GUID: sibling.GUID, Name: sibling.Name, Type: FilesystemType})
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	review := plan.Review()
	if len(review.Context) != 2 || review.Context[0].Properties.Quota.Effective != "4096" {
		t.Fatal("ancestor constraints omitted")
	}
	// An unrelated, unenrolled sibling remains observational. No ancestor or target
	// identity/property changes; its quota does not alter this reviewed subtree.
	o.Filesystems[2].Properties.Quota = Value{"8192", Local, "103", ""}
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal(err)
	}
	o.Filesystems[0].Properties.Quota = Value{"8192", Local, "101", ""}
	if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
		t.Fatal(err)
	}
}

func TestDefaultCompressionStillHasInheritanceImpact(t *testing.T) {
	p, o, i := fixture()
	i.Property = Compression
	i.Value = "lz4"
	for n := range o.Filesystems {
		o.Filesystems[n].Properties.Compression = Value{"on", Default, "", ""}
	}
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Review().Affected) != 2 {
		t.Fatal("default compression impact omitted")
	}
	p.Scopes = p.Scopes[:1]
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrDenied) {
		t.Fatal(err)
	}
}

func TestPolicyPropertyOrderingDoesNotInvalidateReview(t *testing.T) {
	p, o, i := fixture()
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	p.Scopes[0].Properties[0], p.Scopes[0].Properties[2] = p.Scopes[0].Properties[2], p.Scopes[0].Properties[0]
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal(err)
	}
}

func FuzzPropertyPlan(f *testing.F) {
	for _, seed := range []struct{ guid, property, value, name, source string }{
		{"101", "quota", "none", "tank/data", "inherited"},
		{"101", "refquota", "18446744073709551615", "tank/data", "inherited"},
		{"101", "compression", "off", "tank/data", "inherited"},
		{"101", "quota", "0", "tank/-option", "unknown"},
	} {
		f.Add(seed.guid, seed.property, seed.value, seed.name, seed.source)
	}
	f.Fuzz(func(t *testing.T, guid, property, value, name, source string) {
		p, o, i := fixture()
		i.FilesystemGUID = guid
		i.Property = Property(property)
		i.Value = value
		o.Filesystems[1].Name = name
		o.Filesystems[1].Properties.Compression.Source = Source(source)
		before := o.Filesystems[0]
		plan, err := Build(instant, "session-a", p, o, i)
		if err != nil {
			return
		}
		if err := plan.Validate(instant, "session-a", p, o); err != nil {
			t.Fatal("fresh plan rejected", err)
		}
		if o.Filesystems[0] != before {
			t.Fatal("planning changed its input")
		}
		if !validIntent(i) {
			t.Fatal("invalid intent planned")
		}
		if err := plan.Validate(instant.Add(5*time.Minute), "session-a", p, o); !errors.Is(err, ErrExpired) {
			t.Fatal(err)
		}
	})
}
