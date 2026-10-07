// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestPropertyEnrollmentStateExplicitPredecessor(t *testing.T) {
	p := enrolledPolicy()
	previous, err := enrollmentPredecessor(p, Policy{})
	if err != nil || !reflect.DeepEqual(previous, Policy{}) {
		t.Fatal("explicit first enrollment", err)
	}
	for _, mode := range []string{"not-first", "partial-zero", "different-pool", "same-revision", "skipped-revision", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			p, previous := enrolledPolicy(), enrolledPolicy()
			p.Revision++
			switch mode {
			case "not-first":
				previous = Policy{}
			case "partial-zero":
				previous = Policy{PoolGUID: p.PoolGUID}
			case "different-pool":
				previous.PoolGUID = "999"
			case "same-revision":
				p.Revision = previous.Revision
			case "skipped-revision":
				p.Revision++
			case "overflow":
				previous.Revision = ^uint64(0)
				p.Revision = 1
			}
			if _, err := enrollmentPredecessor(p, previous); err == nil {
				t.Fatal("inferred absence or invalid successor")
			}
		})
	}
	previous = enrolledPolicy()
	previous.Members = nil
	p.Revision = 2
	canonical, err := enrollmentPredecessor(p, previous)
	if err != nil {
		t.Fatal("explicit legacy predecessor", err)
	}
	before, _ := EncodeOwnershipRecord(canonical)
	previous.Scopes[0].Properties[0] = Compression
	after, _ := EncodeOwnershipRecord(canonical)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("predecessor aliases input")
	}
}

func TestPropertyEnrollmentStateCatalogProjectsCapacityAndBacking(t *testing.T) {
	p, _, _, u := enrollmentFixture(t)
	p.Revision = 1
	data, err := EncodeOwnershipRecord(p)
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := BuildJobReservations(nil)
	if err != nil {
		t.Fatal(err)
	}
	b := propertyStateBacking{kind: "ufs", source: "/dev/gpt/rootfs", target: "/"}
	if err = checkEnrollmentStateCatalog(StateCatalog{}, p, Policy{}, uint64(len(data)), defaultStateCatalogLimits, jobs, b, u); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"records", "temporary-bytes", "existing-first", "missing-predecessor", "new-owner-overlap", "queued-job", "terminal-job", "unsupported"} {
		t.Run(mode, func(t *testing.T) {
			p, _, _, u := enrollmentFixture(t)
			p.Revision = 1
			c, previous, limits, jobs, backing := StateCatalog{}, Policy{}, defaultStateCatalogLimits, jobs, b
			switch mode {
			case "records":
				limits.ownership = 1
				c.Ownership = []OwnershipSummary{{PoolGUID: "999", Revision: 1, Bytes: 1}}
				c.Bytes = 1
			case "temporary-bytes":
				limits.bytes = uint64(len(data))
				c.Bytes = 1
			case "existing-first":
				c.Ownership = []OwnershipSummary{{PoolGUID: p.PoolGUID, Revision: 1, Bytes: 1}}
				c.Bytes = 1
			case "missing-predecessor":
				previous = p
				p.Revision++
			case "new-owner-overlap":
				u.pools[0].Disks[0].ID = u.system.Mounts[0].Disks[0].ID
			case "queued-job", "terminal-job":
				j := jobFixture()
				if mode == "terminal-job" {
					j = restartJobFixture(t, j, JobCancelled, "", "")
				}
				encoded, e := EncodeJobRecord(j)
				if e != nil {
					t.Fatal(e)
				}
				jobs, e = BuildJobReservations([][]byte{encoded})
				if e != nil {
					t.Fatal(e)
				}
			case "unsupported":
				backing.softupdates = true
			}
			if err := checkEnrollmentStateCatalog(c, p, previous, uint64(len(data)), limits, jobs, backing, u); err == nil {
				t.Fatal("unsafe enrollment store accepted")
			}
		})
	}
}

func TestPropertyEnrollmentStateZeroNeverValidates(t *testing.T) {
	_, o, d, u := enrollmentFixture(t)
	zero := PropertyEnrollmentState{}
	if err := zero.Validate(instant, "root", o, d, u); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(zero.Review(), PropertyEnrollmentReview{}) || !reflect.DeepEqual(zero.Predecessor(), Policy{}) {
		t.Fatal("zero state leaked review")
	}
}
