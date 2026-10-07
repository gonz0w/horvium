// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func reservationBytes(t testing.TB, value any) []byte {
	t.Helper()
	var data []byte
	var err error
	switch v := value.(type) {
	case PropertyJob:
		data, err = EncodePropertyJob(v)
	case Job:
		data, err = EncodeJobRecord(v)
	default:
		t.Fatal("unexpected reservation fixture")
	}
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func reservationJob(number int, terminal bool) Job {
	j := jobFixture()
	j.ID = fmt.Sprintf("%064x", number*3+1)
	j.Intent.PlanID = fmt.Sprintf("%064x", number*3+2)
	j.Intent.SubmissionID = fmt.Sprintf("%064x", number*3+3)
	j.Intent.PoolGUID = fmt.Sprint(number + 1000)
	if terminal {
		j.History = append(j.History, JobTransition{Revision: 2, At: 1001, State: JobCancelled, Reason: "operator-cancelled"})
	}
	return j
}

func TestReservationViewAndIsolation(t *testing.T) {
	p := propertyJobFixture(t)
	j := reservationJob(1, false)
	j.Intent.PoolGUID = p.Job().Intent.PoolGUID
	data := [][]byte{reservationBytes(t, j), reservationBytes(t, p)}
	view, err := BuildJobReservations(data)
	if err != nil {
		t.Fatal(err)
	}
	pools := view.Pools()
	if len(pools) != 1 || pools[0].PoolGUID != "100" || len(pools[0].Jobs) != 2 || pools[0].Jobs[0].ID != j.ID || pools[0].Jobs[1].ID != p.Job().ID {
		t.Fatal("lost whole-pool holders", pools)
	}
	before := view.Pools()
	pools[0].PoolGUID = "changed"
	pools[0].Jobs[0].State = JobSucceeded
	for _, record := range data {
		record[0] = 'x'
	}
	if !reflect.DeepEqual(before, view.Pools()) {
		t.Fatal("input/output aliases retained view")
	}
	got, duplicate, err := view.CheckPropertySubmission(p)
	if err != nil || !duplicate || !reflect.DeepEqual(got, p) {
		t.Fatal("lost complete duplicate baseline", err)
	}
	got.job.History[0].Reason = "changed"
	got.intent.review.Context[0].Name = "changed"
	again, duplicate, err := view.CheckPropertySubmission(p)
	if err != nil || !duplicate || !reflect.DeepEqual(again, p) {
		t.Fatal("duplicate return aliases view", err)
	}
	other, err := BuildJobReservations([][]byte{reservationBytes(t, p), reservationBytes(t, j)})
	if err != nil || !reflect.DeepEqual(view.Pools(), other.Pools()) {
		t.Fatal("record order changed projection", err)
	}
}

func TestReservationViewDenials(t *testing.T) {
	p := propertyJobFixture(t)
	j := reservationJob(1, false)
	for _, tc := range []struct {
		name string
		data [][]byte
		want error
	}{
		{"malformed", [][]byte{[]byte("{")}, ErrStateRecord},
		{"late malformed", [][]byte{reservationBytes(t, p), []byte("{")}, ErrStateRecord},
		{"oversized", [][]byte{bytes.Repeat([]byte("x"), maxStateRecordBytes+1)}, ErrLimit},
		{"duplicate job", [][]byte{reservationBytes(t, p), reservationBytes(t, p)}, ErrDrift},
		{"ownership kind", [][]byte{[]byte(`{"kind":"ownership"}`)}, ErrStateRecord},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildJobReservations(tc.data)
			if !errors.Is(err, tc.want) || !reflect.DeepEqual(got, JobReservations{}) {
				t.Fatal("invalid census retained partial view", err)
			}
		})
	}
	for _, field := range []string{"id", "submission", "plan"} {
		t.Run("mixed collision "+field, func(t *testing.T) {
			other := j
			switch field {
			case "id":
				other.ID = p.Job().ID
			case "submission":
				other.Intent.SubmissionID = p.Job().Intent.SubmissionID
			case "plan":
				other.Intent.PlanID = p.Job().Intent.PlanID
			}
			got, err := BuildJobReservations([][]byte{reservationBytes(t, p), reservationBytes(t, other)})
			if !errors.Is(err, ErrDrift) || !reflect.DeepEqual(got, JobReservations{}) {
				t.Fatal("cross-format collision accepted", err)
			}
		})
	}
	if result, duplicate, err := (JobReservations{}).CheckPropertySubmission(p); !errors.Is(err, ErrInvalid) || duplicate || !reflect.DeepEqual(result, PropertyJob{}) {
		t.Fatal("zero view treated as complete empty census", err)
	}
}

func TestReservationPoolHolds(t *testing.T) {
	p := propertyJobFixture(t)
	for _, state := range []JobState{JobQueued, JobApplying, JobObserving, JobSucceeded, JobFailed, JobCancelled, JobReconcileRequired} {
		t.Run(string(state), func(t *testing.T) {
			j := reservationJob(1, false)
			j.Intent.PoolGUID = p.Job().Intent.PoolGUID
			if state == JobCancelled {
				j, _ = TransitionJob(j, state, 1001, "operator-cancelled", "")
			} else if state != JobQueued {
				j, _ = TransitionJob(j, JobApplying, 1001, "launch-intent", "")
				if state == JobReconcileRequired {
					j, _ = TransitionJob(j, state, 1002, "executor-unverified", "")
				} else if state != JobApplying {
					j, _ = TransitionJob(j, JobObserving, 1002, "native-observation", strings.Repeat("d", 64))
					if state == JobSucceeded || state == JobFailed {
						reason := "verified"
						if state == JobFailed {
							reason = "verified-failure"
						}
						j, _ = TransitionJob(j, state, 1003, reason, strings.Repeat("d", 64))
					}
				}
			}
			view, err := BuildJobReservations([][]byte{reservationBytes(t, j)})
			if err != nil {
				t.Fatal(err)
			}
			got, duplicate, err := view.CheckPropertySubmission(p)
			if !errors.Is(err, ErrDenied) || duplicate || !reflect.DeepEqual(got, PropertyJob{}) || len(view.Pools()) != 1 {
				t.Fatal("metadata label released whole pool", err)
			}
		})
	}
	j := reservationJob(1, false)
	view, err := BuildJobReservations([][]byte{reservationBytes(t, j)})
	got, duplicate, check := view.CheckPropertySubmission(p)
	if err != nil || check != nil || duplicate || !reflect.DeepEqual(got, PropertyJob{}) {
		t.Fatal("unrelated pool blocked metadata check", err, check)
	}
}

func TestReservationSubmissionChecks(t *testing.T) {
	p := propertyJobFixture(t)
	applying, err := TransitionPropertyJob(p, JobApplying, p.job.History[0].At+1, "launch-intent", "")
	if err != nil {
		t.Fatal(err)
	}
	view, err := BuildJobReservations([][]byte{reservationBytes(t, applying)})
	if err != nil {
		t.Fatal(err)
	}
	// A retry can propose a new job ID/capture time but never a new review,
	// session/policy binding or submission meaning. Return current durable history.
	retry := p
	retry.job = p.Job()
	retry.intent = p.Intent()
	retry.job.ID = strings.Repeat("f", 64)
	retry.intent.capturedAt = retry.intent.capturedAt.Add(time.Nanosecond)
	got, duplicate, err := view.CheckPropertySubmission(retry)
	if err != nil || !duplicate || !reflect.DeepEqual(got, applying) {
		t.Fatal("retry did not return existing current record", err)
	}
	for _, tc := range []struct {
		name   string
		change func(*PropertyJob)
		want   error
	}{
		{"new submission same plan", func(p *PropertyJob) { p.job.Intent.SubmissionID = strings.Repeat("e", 64) }, ErrDenied},
		{"session", func(p *PropertyJob) { p.intent.sessionDigest[0] ^= 1 }, ErrDenied},
		{"policy", func(p *PropertyJob) { p.intent.policyDigest[0] ^= 1 }, ErrDenied},
		{"policy revision", func(p *PropertyJob) { p.job.Intent.PolicyRevision++; p.intent.review.PolicyRevision++ }, ErrDenied},
		{"another plan", func(p *PropertyJob) {
			p.job.Intent.PlanID = strings.Repeat("e", 64)
			p.intent.review.ID = p.job.Intent.PlanID
		}, ErrDenied},
		{"changed selection", func(p *PropertyJob) { p.job.Intent.Change.Value = "none"; p.intent.review.Intent.Value = "none" }, ErrDenied},
		{"oversized baseline", func(p *PropertyJob) { p.intent.review.Context = make([]Filesystem, maxRecords+1) }, ErrLimit},
		{"oversized history", func(p *PropertyJob) { p.job.History = make([]JobTransition, maxJobTransitions+1) }, ErrLimit},
		{"capture after admission", func(p *PropertyJob) { p.intent.capturedAt = p.intent.capturedAt.Add(time.Hour) }, ErrStateRecord},
		{"not initial queued", func(p *PropertyJob) { *p = applying }, ErrStateRecord},
		{"zero", func(p *PropertyJob) { *p = PropertyJob{} }, ErrStateRecord},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := retry
			candidate.job = retry.Job()
			candidate.intent = retry.Intent()
			tc.change(&candidate)
			got, duplicate, err := view.CheckPropertySubmission(candidate)
			if !errors.Is(err, tc.want) || duplicate || !reflect.DeepEqual(got, PropertyJob{}) {
				t.Fatal("contradictory submission survived", err)
			}
		})
	}
	_, plan, policy, observation := propertyIntentFixture(t)
	observation.Filesystems[0].Properties.Compression.Effective = "off"
	observation.Filesystems[1].Properties.Compression.Effective = "off"
	// Compare a separately valid changed review under the same plan/submission.
	changedPlan, err := Build(plan.review.CreatedAt, "fixture-session-secret", policy, observation, p.job.Intent.Change)
	if err != nil {
		t.Fatal(err)
	}
	changedPlan.review.ID = p.job.Intent.PlanID
	changedIntent, err := changedPlan.CapturePropertyIntent(p.intent.capturedAt, "fixture-session-secret", policy, observation)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := BindPropertyJob(retry.Job(), changedIntent)
	if err != nil {
		t.Fatal(err)
	}
	if got, duplicate, err := view.CheckPropertySubmission(changed); !errors.Is(err, ErrDenied) || duplicate || !reflect.DeepEqual(got, PropertyJob{}) {
		t.Fatal("changed valid review reused submission", err)
	}
	legacy, err := BuildJobReservations([][]byte{reservationBytes(t, p.Job())})
	if err != nil {
		t.Fatal(err)
	}
	if got, duplicate, err := legacy.CheckPropertySubmission(p); !errors.Is(err, ErrDenied) || duplicate || !reflect.DeepEqual(got, PropertyJob{}) {
		t.Fatal("legacy record fabricated retry baseline", err)
	}
	other := reservationJob(1, false)
	other.ID = retry.job.ID
	mixed, err := BuildJobReservations([][]byte{reservationBytes(t, applying), reservationBytes(t, other)})
	if err != nil {
		t.Fatal(err)
	}
	if _, duplicate, err := mixed.CheckPropertySubmission(retry); !errors.Is(err, ErrDenied) || duplicate {
		t.Fatal("retry's job ID collided with another submission", err)
	}
	for _, state := range []JobState{JobCancelled, JobReconcileRequired} {
		t.Run("retry "+string(state), func(t *testing.T) {
			reason := "operator-cancelled"
			if state == JobReconcileRequired {
				reason = "review-expired"
			}
			current, err := TransitionPropertyJob(p, state, p.job.History[0].At+1, reason, "")
			if err != nil {
				t.Fatal(err)
			}
			view, err := BuildJobReservations([][]byte{reservationBytes(t, current)})
			if err != nil {
				t.Fatal(err)
			}
			got, duplicate, err := view.CheckPropertySubmission(retry)
			if err != nil || !duplicate || !reflect.DeepEqual(got, current) {
				t.Fatal("retry reset existing outcome", err)
			}
		})
	}
}

func TestReservationBounds(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		name, limit := "nonterminal", 64
		if terminal {
			name, limit = "terminal", 1000
		}
		t.Run(name, func(t *testing.T) {
			records := make([][]byte, limit+1)
			for i := range records {
				records[i] = reservationBytes(t, reservationJob(i, terminal))
			}
			if _, err := BuildJobReservations(records[:limit]); err != nil {
				t.Fatal("exact ceiling", err)
			}
			if got, err := BuildJobReservations(records); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, JobReservations{}) {
				t.Fatal("class overflow", err)
			}
		})
	}
	if _, err := BuildJobReservations(make([][]byte, 1065)); !errors.Is(err, ErrLimit) {
		t.Fatal("unbounded record count", err)
	}
	large := bytes.Repeat([]byte(" "), maxStateRecordBytes)
	records := make([][]byte, 1025)
	for i := range records {
		records[i] = large
	}
	if _, err := BuildJobReservations(records); !errors.Is(err, ErrLimit) {
		t.Fatal("aggregate preflight before decoding", err)
	}
	p := propertyJobFixture(t)
	data := reservationBytes(t, p)
	limits := stateCatalogLimits{1, 1, 1, uint64(len(data))}
	view, err := buildJobReservations([][]byte{data}, limits)
	if err != nil {
		t.Fatal("exact reduced byte ceiling", err)
	}
	if got, duplicate, err := view.CheckPropertySubmission(p); err != nil || !duplicate || !reflect.DeepEqual(got, p) {
		t.Fatal("full view lost duplicate", err)
	}
	if got, err := buildJobReservations([][]byte{append(data, ' ')}, limits); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, JobReservations{}) {
		t.Fatal("presentation bytes omitted", err)
	}
	other := reservationBytes(t, reservationJob(1, false))
	limits.bytes = uint64(len(other))
	view, err = buildJobReservations([][]byte{other}, limits)
	if err != nil {
		t.Fatal(err)
	}
	if got, duplicate, err := view.CheckPropertySubmission(p); !errors.Is(err, ErrLimit) || duplicate || !reflect.DeepEqual(got, PropertyJob{}) {
		t.Fatal("new work exceeded capacity", err)
	}
	// Byte exhaustion independently denies work when a nonterminal slot remains.
	limits.nonterminal = 2
	view, err = buildJobReservations([][]byte{other}, limits)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := view.CheckPropertySubmission(p); !errors.Is(err, ErrLimit) {
		t.Fatal("bytes overflow with free slot", err)
	}
	if got, err := buildJobReservations(nil, stateCatalogLimits{}); !errors.Is(err, ErrLimit) || !reflect.DeepEqual(got, JobReservations{}) {
		t.Fatal("invalid limits", err)
	}
}

func TestReservationChecksAreReadOnly(t *testing.T) {
	view, err := BuildJobReservations(nil)
	if err != nil || len(view.Pools()) != 0 {
		t.Fatal("complete empty view", err)
	}
	p := propertyJobFixture(t)
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			got, duplicate, err := view.CheckPropertySubmission(p)
			if err != nil || duplicate || !reflect.DeepEqual(got, PropertyJob{}) {
				t.Error("read-only check", err)
			}
		})
	}
	workers.Wait()
	if len(view.Pools()) != 0 {
		t.Fatal("check claimed a plan or acquired a reservation")
	}
}

func FuzzJobReservations(f *testing.F) {
	p := propertyJobFixture(f)
	f.Add(reservationBytes(f, p))
	f.Add(reservationBytes(f, reservationJob(1, false)))
	f.Add([]byte("{}"))
	f.Fuzz(func(t *testing.T, data []byte) {
		view, err := BuildJobReservations([][]byte{data})
		if err != nil {
			if !reflect.DeepEqual(view, JobReservations{}) {
				t.Fatal("partial invalid view")
			}
			return
		}
		if pools := view.Pools(); len(pools) != 1 || len(pools[0].Jobs) != 1 {
			t.Fatal("valid record lost holder")
		}
		got, duplicate, err := view.CheckPropertySubmission(p)
		if err != nil && (duplicate || !reflect.DeepEqual(got, PropertyJob{})) {
			t.Fatal("partial denial")
		}
		if duplicate && (err != nil || got.job.Intent != p.job.Intent || !reflect.DeepEqual(got.intent.review, p.intent.review)) {
			t.Fatal("false duplicate")
		}
	})
}

func TestReservationStreamingBuilder(t *testing.T) {
	p := propertyJobFixture(t)
	j := reservationJob(1, false)
	data := [][]byte{reservationBytes(t, p), reservationBytes(t, j)}
	b, err := newReservationBuilder(defaultStateCatalogLimits)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range data {
		if err := b.add(record); err != nil {
			t.Fatal(err)
		}
	}
	before := b.view.bytes
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{"invalid", []byte("{"), ErrStateRecord},
		{"duplicate", data[0], ErrDrift},
		{"oversized", bytes.Repeat([]byte("x"), maxStateRecordBytes+1), ErrLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := b.add(tc.data); !errors.Is(err, tc.want) || b.view.bytes != before || len(b.view.records) != 2 {
				t.Fatal("failed addition altered builder", err)
			}
		})
	}
	got := b.finish()
	want, err := BuildJobReservations(data)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("streaming lost complete view", err)
	}
	if err := b.add(data[0]); !errors.Is(err, ErrInvalid) {
		t.Fatal("finished builder changed published view", err)
	}
	if !reflect.DeepEqual(b.finish(), JobReservations{}) || !reflect.DeepEqual(got, want) {
		t.Fatal("finish aliased a mutable builder")
	}
	if _, err := newReservationBuilder(stateCatalogLimits{}); !errors.Is(err, ErrLimit) {
		t.Fatal("invalid builder limits", err)
	}
	limits := stateCatalogLimits{1, 1, 1, uint64(len(data[0]))}
	b, err = newReservationBuilder(limits)
	if err != nil || b.add(data[0]) != nil {
		t.Fatal("reduced exact builder bound", err)
	}
	if err := b.add(data[1]); !errors.Is(err, ErrLimit) || b.view.bytes != uint64(len(data[0])) || len(b.view.records) != 1 {
		t.Fatal("overflow altered builder", err)
	}
}
