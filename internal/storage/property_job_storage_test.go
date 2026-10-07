// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestPropertyStateCatalog(t *testing.T) {
	p := propertyJobFixture(t)
	data, _ := EncodePropertyJob(p)
	legacy := catalogJob(2, false)
	old, _ := EncodeJobRecord(legacy)
	limits := stateCatalogLimits{1, 2, 1, uint64(len(data) + len(old))}
	b := newStateCatalogBuilder(limits)
	if e := b.add(stateRecordName{jobID: p.Job().ID}, data); e != nil {
		t.Fatal("complete property job not cataloged", e)
	}
	if e := b.add(stateRecordName{jobID: legacy.ID}, old); e != nil {
		t.Fatal("legacy coexistence", e)
	}
	if b.catalog.Bytes != limits.bytes || b.catalog.Nonterminal != 2 || len(b.catalog.Jobs) != 2 || b.catalog.Jobs[0].Bytes != uint64(len(data)) {
		t.Fatal("baseline bytes or class omitted", b.catalog)
	}
	before := b.catalog
	collision := catalogJob(3, false)
	collision.Intent.SubmissionID = p.Job().Intent.SubmissionID
	duplicated, _ := EncodeJobRecord(collision)
	b.limits.bytes = 64 << 20
	if e := b.add(stateRecordName{jobID: collision.ID}, duplicated); !errors.Is(e, ErrDrift) || !reflect.DeepEqual(before, b.catalog) {
		t.Fatal("cross-format duplicate", e)
	}
	fresh := newStateCatalogBuilder(defaultStateCatalogLimits)
	if e := fresh.add(stateRecordName{jobID: legacy.ID}, data); !errors.Is(e, ErrDrift) || fresh.catalog.Bytes != 0 {
		t.Fatal("filename binding", e)
	}
	cancelled, e := TransitionPropertyJob(p, JobCancelled, p.Job().History[0].At, "operator-cancelled", "")
	if e != nil {
		t.Fatal(e)
	}
	terminal, _ := EncodePropertyJob(cancelled)
	fresh = newStateCatalogBuilder(defaultStateCatalogLimits)
	if e := fresh.add(stateRecordName{jobID: p.Job().ID}, terminal); e != nil || fresh.catalog.Terminal != 1 || fresh.catalog.Nonterminal != 0 {
		t.Fatal("terminal classification", e)
	}
	for _, bad := range [][]byte{data[:len(data)-1], append(append([]byte(nil), data...), []byte(" {}")...)} {
		fresh = newStateCatalogBuilder(defaultStateCatalogLimits)
		if e := fresh.add(stateRecordName{jobID: p.Job().ID}, bad); e == nil || fresh.catalog.Bytes != 0 {
			t.Fatal("damaged property metadata admitted", e)
		}
	}
}
