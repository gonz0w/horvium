// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

func enrollmentFixture(t *testing.T) (Policy, Observation, []inventory.DiskObservation, PoolSystemUse) {
	t.Helper()
	p, disks := enrolledPolicy(), memberDisks()
	_, o, _ := fixture()
	o.Eligible = false
	// Select only the child, without implicitly selecting its ancestor or siblings.
	p.Scopes = p.Scopes[1:]
	u := propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\n", emptySwap)
	u.pools[0].Disks = []inventory.Provider{disks[0].Provider, disks[1].Provider}
	return p, o, disks, u
}

func TestPropertyEnrollmentExplicitImmutableReview(t *testing.T) {
	p, o, d, u := enrollmentFixture(t)
	plan, err := BuildPropertyEnrollment(instant, "root-review", p, o, d, u)
	if err != nil {
		t.Fatal(err)
	}
	r := plan.Review()
	if len(r.ID) != 64 || !r.ExpiresAt.Equal(instant.Add(5*time.Minute)) || len(r.Policy.Scopes) != 1 || r.Policy.Scopes[0].GUID != "102" || len(r.Observation.Filesystems) != 2 || r.Observation.Eligible {
		t.Fatal("implicit selection, incomplete context or authority", r)
	}
	if err = plan.Validate(instant.Add(time.Minute), "root-review", p, o, d, u); err != nil {
		t.Fatal(err)
	}
	second, err := BuildPropertyEnrollment(instant, "root-review", p, o, d, u)
	if err != nil || second.Review().ID == r.ID {
		t.Fatal("identifier reuse", err)
	}
	p.Scopes[0].Properties[0] = Compression
	p.Members[0].Serial = "changed"
	o.Filesystems[0].Name = "changed"
	o.Identities[0].GUID = "999"
	o.States[0].Name = "changed"
	o.Features.Values[0].State = "changed"
	o.Topology.Nodes[0].GUID = "999"
	u.pools[0].Disks[0].Identifiers[0].Value = "changed"
	r.Policy.Scopes[0].Properties[0] = Compression
	r.Policy.Members[0].Serial = "changed"
	r.Observation.Filesystems[0].Name = "changed"
	r.Observation.Identities[0].GUID = "999"
	r.Observation.States[0].Name = "changed"
	r.Observation.Features.Values[0].State = "changed"
	r.Observation.Topology.Nodes[0].GUID = "999"
	r.System.Mounts[0].Disks[0].Name = "changed"
	r.Pools[0].Disks[0].Identifiers[0].Value = "changed"
	freshP, freshO, freshD, freshU := enrollmentFixture(t)
	if err = plan.Validate(instant, "root-review", freshP, freshO, freshD, freshU); err != nil {
		t.Fatal("aliased inputs/review", err)
	}
	if plan.Review().Policy.Members[0].Serial == "changed" {
		t.Fatal("member alias")
	}
	_, _, intent := fixture()
	if _, err = Build(instant, "session", freshP, freshO, intent); !errors.Is(err, ErrUnavailable) {
		t.Fatal("review grants mutation plan", err)
	}
}

func TestPropertyEnrollmentDenialsReturnZero(t *testing.T) {
	for _, mode := range []string{"time", "context", "policy", "members", "scope", "unknown-scope", "partial", "eligible", "native", "system", "system-provider", "system-mount", "budget", "rows", "identifiers"} {
		t.Run(mode, func(t *testing.T) {
			p, o, d, u := enrollmentFixture(t)
			at, context := instant, "root-review"
			switch mode {
			case "time":
				at = time.Time{}
			case "context":
				context = "untrusted context"
			case "policy":
				p.Revision = 0
			case "members":
				p.Members = nil
			case "scope":
				p.Scopes = nil
			case "unknown-scope":
				p.Scopes[0].GUID = "999"
			case "partial":
				o.Complete = false
			case "eligible":
				o.Eligible = true
			case "native":
				d[0].Provider.Identifiers[0].Value = "replacement"
			case "system":
				u = PoolSystemUse{}
			case "system-provider":
				u.pools[0].Disks[0].SectorSize = 4096
			case "system-mount":
				u.system.Mounts = append(u.system.Mounts, MountUse{Source: "tank/data", Target: "/var/db", Type: "zfs", PoolGUID: "100", Disks: u.pools[0].Disks})
			case "budget":
				o.Topology.Diagnostics = strings.Repeat("x", 1<<20)
			case "rows":
				o.Identities = make([]DatasetIdentity, 1<<16)
			case "identifiers":
				d[0].Provider.Identifiers = make([]inventory.Identifier, 1<<16)
			}
			plan, err := BuildPropertyEnrollment(at, context, p, o, d, u)
			if err == nil || !reflect.DeepEqual(plan, PropertyEnrollmentPlan{}) {
				t.Fatal("partial or authoritative review", err)
			}
			if mode == "budget" || mode == "rows" || mode == "identifiers" {
				if !errors.Is(err, ErrLimit) {
					t.Fatal("budget refusal", err)
				}
			}
		})
	}
}

func TestPropertyEnrollmentExactReviewCeiling(t *testing.T) {
	p, o, d, u := enrollmentFixture(t)
	// Individually bounded valid mount fields can expand under JSON escaping.
	// Refuse the whole context even when the preliminary working budget fits.
	for i := 0; i < 4; i++ {
		u.system.Mounts = append(u.system.Mounts, MountUse{Source: "tmpfs", Target: "/" + strings.Repeat("<", 3000) + string(rune('a'+i)), Type: "tmpfs"})
	}
	remaining := 6*maxPlanBytes - 4096
	for _, input := range []any{p, o, d, u.system, u.pools, u.topologies} {
		if err := enrollmentBudget(reflect.ValueOf(input), &remaining); err != nil {
			t.Fatal("fixture did not reach exact review check", err)
		}
	}
	plan, err := BuildPropertyEnrollment(instant, "root-review", p, o, d, u)
	if !errors.Is(err, ErrLimit) || !reflect.DeepEqual(plan, PropertyEnrollmentPlan{}) {
		t.Fatal("oversized complete review", err)
	}
}

func TestPropertyEnrollmentFreshValidationAndExpiry(t *testing.T) {
	p, o, d, u := enrollmentFixture(t)
	plan, err := BuildPropertyEnrollment(instant, "root-review", p, o, d, u)
	if err != nil {
		t.Fatal(err)
	}
	if err = (PropertyEnrollmentPlan{}).Validate(instant, "root-review", p, o, d, u); !errors.Is(err, ErrInvalid) {
		t.Fatal("zero plan", err)
	}
	for _, at := range []time.Time{instant.Add(-time.Nanosecond), instant.Add(5 * time.Minute)} {
		if err = plan.Validate(at, "root-review", p, o, d, u); !errors.Is(err, ErrExpired) {
			t.Fatal("expiry", err)
		}
	}
	if err = plan.Validate(instant, "other-root-review", p, o, d, u); !errors.Is(err, ErrDenied) {
		t.Fatal("context", err)
	}
	for _, mode := range []string{"revision", "selection", "property-scope", "serial", "native", "unselected-property", "new-child", "feature", "incarnation", "mount", "system-provider", "other-topology", "version"} {
		t.Run(mode, func(t *testing.T) {
			p, o, d, u := enrollmentFixture(t)
			switch mode {
			case "revision":
				p.Revision++
			case "selection":
				p.Scopes = nil
			case "property-scope":
				p.Scopes[0].Properties = []Property{RefQuota}
			case "serial":
				p.Members[0].Serial = "changed"
			case "native":
				d[0].StripeOffset = 512
			case "unselected-property":
				o.Filesystems[0].Properties.Quota = Value{"1", Local, "101", ""}
			case "new-child":
				f := o.Filesystems[1]
				f.GUID = "103"
				f.Name = "tank/other"
				o.Filesystems = append(o.Filesystems, f)
				o.States = append(o.States, stateFixture(f))
				o.Identities = append(o.Identities, DatasetIdentity{GUID: f.GUID, Name: f.Name, Type: FilesystemType})
			case "feature":
				o.Features.Values[0].State = "disabled"
			case "incarnation":
				o.Pool.LoadGUID = "999"
			case "mount":
				u.system.Mounts[0].Target = "/boot"
			case "system-provider":
				u.system.Mounts[0].Disks[0].MediaSize *= 2
			case "other-topology":
				other := normalizedTopology(u.topologies[0])
				other.PoolGUID = "999"
				other.PoolName = "other"
				u.topologies = append(u.topologies, other)
			case "version":
				o.Identities = append(o.Identities, DatasetIdentity{GUID: "888", Name: "tank@retained", Type: SnapshotType, OwnerGUID: "101"})
			}
			if err := plan.Validate(instant, "root-review", p, o, d, u); !errors.Is(err, ErrDrift) {
				t.Fatal("drift did not require new review", err)
			}
		})
	}
}

func TestPropertyEnrollmentCanonicalOrderAndGraphIDs(t *testing.T) {
	p, o, d, u := enrollmentFixture(t)
	plan, err := BuildPropertyEnrollment(instant, "root-review", p, o, d, u)
	if err != nil {
		t.Fatal(err)
	}
	p.Members[0], p.Members[1] = p.Members[1], p.Members[0]
	p.Scopes[0].Properties[0], p.Scopes[0].Properties[1] = p.Scopes[0].Properties[1], p.Scopes[0].Properties[0]
	o.Filesystems[0], o.Filesystems[1] = o.Filesystems[1], o.Filesystems[0]
	o.States[0], o.States[1] = o.States[1], o.States[0]
	o.Identities[0], o.Identities[1] = o.Identities[1], o.Identities[0]
	d[0], d[1] = d[1], d[0]
	u.pools[0].Disks[0], u.pools[0].Disks[1] = u.pools[0].Disks[1], u.pools[0].Disks[0]
	for i := range d {
		d[i].Provider.ID = "fresh"
		d[i].Provider.GeomID = "fresh"
	}
	for i := range u.pools[0].Disks {
		u.pools[0].Disks[i].ID = "fresh"
		u.pools[0].Disks[i].GeomID = "fresh"
	}
	if err = plan.Validate(instant, "root-review", p, o, d, u); err != nil {
		t.Fatal("ordering/graph-local ID treated as identity", err)
	}
}
