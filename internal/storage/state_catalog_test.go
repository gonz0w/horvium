// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func catalogJob(n int, terminal bool) Job {
	j := jobFixture()
	j.ID = fmt.Sprintf("%064x", n)
	j.Intent.SubmissionID = fmt.Sprintf("%064x", n+2000)
	j.Intent.PlanID = fmt.Sprintf("%064x", n+4000)
	if terminal {
		j, _ = TransitionJob(j, JobCancelled, 1001, "operator-cancelled", "")
	}
	return j
}

func TestStateCatalogNames(t *testing.T) {
	id := strings.Repeat("a", 64)
	names := []string{"ownership-200.json", "job-" + id + ".json", "writer.lock", "ownership-100.json"}
	before := append([]string(nil), names...)
	got, err := stateCatalogNames(names, defaultStateCatalogLimits)
	if err != nil || len(got) != 3 || got[0].jobID != id || got[1].poolGUID != "100" || got[2].poolGUID != "200" || !reflect.DeepEqual(names, before) {
		t.Fatal("complete independent sorted selectors", got, err)
	}
	got[0].name = "changed"
	again, err := stateCatalogNames(names, defaultStateCatalogLimits)
	if err != nil || again[0].name != "job-"+id+".json" {
		t.Fatal("result aliased input")
	}
	for _, name := range []string{"job-1.json", "job-" + strings.Repeat("A", 64) + ".json", "job-" + id + ".JSON", "job-" + id + ".json/child", "ownership-0100.json", "ownership-0.json", "ownership-18446744073709551616.json", "../writer.lock", "unknown", ".ownership-" + strings.Repeat("a", 31) + ".tmp"} {
		if out, e := stateCatalogNames([]string{"writer.lock", name}, defaultStateCatalogLimits); !errors.Is(e, ErrStateRecord) || out != nil {
			t.Fatal("invalid selector", name, e)
		}
	}
	if out, e := stateCatalogNames([]string{"writer.lock", ".ownership-" + strings.Repeat("a", 32) + ".tmp"}, defaultStateCatalogLimits); !errors.Is(e, ErrStateUncertain) || out != nil {
		t.Fatal("orphan accepted", e)
	}
	for _, input := range [][]string{nil, {"ownership-100.json"}, {"writer.lock", "writer.lock"}, {"writer.lock", "job-" + id + ".json", "job-" + id + ".json"}} {
		if out, e := stateCatalogNames(input, defaultStateCatalogLimits); e == nil || out != nil {
			t.Fatal("missing/duplicate name accepted")
		}
	}
	limits := defaultStateCatalogLimits
	limits.ownership = 1
	limits.nonterminal = 1
	limits.terminal = 1
	if out, e := stateCatalogNames([]string{"writer.lock", "ownership-100.json", "ownership-200.json"}, limits); !errors.Is(e, ErrLimit) || out != nil {
		t.Fatal("ownership count", e)
	}
	if out, e := stateCatalogNames(append(names, "ownership-300.json"), limits); !errors.Is(e, ErrLimit) || out != nil {
		t.Fatal("complete name count", e)
	}
}

func TestStateCatalogRecords(t *testing.T) {
	b := newStateCatalogBuilder(defaultStateCatalogLimits)
	owner := ownershipGolden()
	if err := b.add(stateRecordName{name: "ownership-100.json", poolGUID: "100"}, owner); err != nil {
		t.Fatal(err)
	}
	total := uint64(len(owner))
	for i, state := range []JobState{JobQueued, JobApplying, JobObserving, JobSucceeded, JobFailed, JobCancelled, JobReconcileRequired} {
		j := catalogJob(i+1, false)
		if state == JobApplying || state == JobObserving || state == JobSucceeded || state == JobFailed {
			j, _ = TransitionJob(j, JobApplying, 1001, "launch-intent", "")
		}
		if state == JobObserving || state == JobSucceeded || state == JobFailed {
			j, _ = TransitionJob(j, JobObserving, 1002, "native-observation", strings.Repeat("d", 64))
		}
		switch state {
		case JobSucceeded:
			j, _ = TransitionJob(j, state, 1003, "verified", strings.Repeat("d", 64))
		case JobFailed:
			j, _ = TransitionJob(j, state, 1003, "verified-failure", strings.Repeat("d", 64))
		case JobCancelled:
			j, _ = TransitionJob(j, state, 1001, "operator-cancelled", "")
		case JobReconcileRequired:
			j, _ = TransitionJob(j, state, 1001, "review-expired", "")
		}
		data, err := EncodeJobRecord(j)
		if err != nil {
			t.Fatal(err)
		}
		if err = b.add(stateRecordName{name: "job-" + j.ID + ".json", jobID: j.ID}, data); err != nil {
			t.Fatal(err)
		}
		last := j.History[len(j.History)-1]
		if got := b.catalog.Jobs[i]; got.ID != j.ID || got.SubmissionID != j.Intent.SubmissionID || got.Revision != last.Revision || got.State != state || got.Bytes != uint64(len(data)) {
			t.Fatal("exact job summary", got)
		}
		total += uint64(len(data))
	}
	if len(b.catalog.Ownership) != 1 || b.catalog.Nonterminal != 4 || b.catalog.Terminal != 3 || b.catalog.Bytes != total {
		t.Fatal("mixed census", b.catalog)
	}
	j := catalogJob(20, false)
	j.Intent.SubmissionID = b.catalog.Jobs[0].SubmissionID
	data, _ := EncodeJobRecord(j)
	if err := b.add(stateRecordName{jobID: j.ID}, data); !errors.Is(err, ErrDrift) {
		t.Fatal("duplicate submission accepted", err)
	}
	for _, tc := range []struct {
		entry stateRecordName
		data  []byte
		want  error
	}{
		{stateRecordName{poolGUID: "200"}, owner, ErrDrift},
		{stateRecordName{jobID: strings.Repeat("f", 64)}, data, ErrDrift},
		{stateRecordName{jobID: j.ID}, owner, ErrStateRecord},
		{stateRecordName{poolGUID: "100"}, data, ErrStateRecord},
		{stateRecordName{jobID: j.ID}, []byte("damaged"), ErrStateRecord},
	} {
		builder := newStateCatalogBuilder(defaultStateCatalogLimits)
		if err := builder.add(tc.entry, tc.data); !errors.Is(err, tc.want) {
			t.Fatal("record binding/codec", err)
		}
	}
}

func TestStateCatalogLimits(t *testing.T) {
	for _, limits := range []stateCatalogLimits{{}, {maxRecords + 1, 64, 1000, 64 << 20}, {1, 65, 1000, 64 << 20}, {1, 64, 1001, 64 << 20}, {1, 64, 1000, (64 << 20) + 1}} {
		if out, e := stateCatalogNames([]string{"writer.lock"}, limits); !errors.Is(e, ErrLimit) || out != nil {
			t.Fatal("invalid limits", limits, e)
		}
	}
	if defaultStateCatalogLimits != (stateCatalogLimits{maxRecords, 64, 1000, 64 << 20}) {
		t.Fatal("approved ceilings changed")
	}
	for _, terminal := range []bool{false, true} {
		b := newStateCatalogBuilder(defaultStateCatalogLimits)
		limit := 64
		if terminal {
			limit = 1000
		}
		for i := 1; i <= limit+1; i++ {
			j := catalogJob(i, terminal)
			data, _ := EncodeJobRecord(j)
			err := b.add(stateRecordName{jobID: j.ID}, data)
			if i <= limit && err != nil || i > limit && !errors.Is(err, ErrLimit) {
				t.Fatal("job class boundary", terminal, i, err)
			}
		}
		if len(b.catalog.Jobs) != limit {
			t.Fatal("failed add exposed a record")
		}
	}
	j := catalogJob(1, false)
	data, _ := EncodeJobRecord(j)
	limits := defaultStateCatalogLimits
	limits.bytes = uint64(len(data))
	b := newStateCatalogBuilder(limits)
	if err := b.add(stateRecordName{jobID: j.ID}, data); err != nil || b.catalog.Bytes != limits.bytes {
		t.Fatal("exact byte bound", err)
	}
	limits.bytes--
	b = newStateCatalogBuilder(limits)
	if err := b.add(stateRecordName{jobID: j.ID}, data); !errors.Is(err, ErrLimit) || b.catalog.Bytes != 0 || len(b.catalog.Jobs) != 0 {
		t.Fatal("byte exhaustion changed builder", err)
	}
}

func FuzzStateCatalogNames(f *testing.F) {
	for _, s := range []string{"writer.lock", "writer.lock\x00ownership-100.json", "writer.lock\x00job-" + strings.Repeat("a", 64) + ".json", "writer.lock\x00../bad"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data string) {
		if len(data) > 4096 {
			return
		}
		names := strings.Split(data, "\x00")
		got, err := stateCatalogNames(names, defaultStateCatalogLimits)
		if strings.Join(names, "\x00") != data {
			t.Fatal("input changed")
		}
		if err != nil {
			if got != nil {
				t.Fatal("partial result")
			}
			return
		}
		if len(got) != len(names)-1 {
			t.Fatal("incomplete census")
		}
		for i, e := range got {
			if i > 0 && got[i-1].name >= e.name || e.poolGUID != "" && !validGUID(e.poolGUID) || e.jobID != "" && !validJobDigest(e.jobID) || (e.poolGUID == "") == (e.jobID == "") {
				t.Fatal("unsafe selector")
			}
		}
	})
}
