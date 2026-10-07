// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
)

func restartJobFixture(t testing.TB, j Job, state JobState, from JobState, reason string) Job {
	t.Helper()
	if state == JobQueued {
		return j
	}
	step := func(state JobState, reason, digest string) {
		var err error
		j, err = TransitionJob(j, state, j.History[len(j.History)-1].At+1, reason, digest)
		if err != nil {
			t.Fatal(err)
		}
	}
	if state == JobCancelled {
		step(state, "operator-cancelled", "")
		return j
	}
	if state == JobReconcileRequired && from == JobQueued {
		step(state, reason, "")
		return j
	}
	step(JobApplying, "launch-intent", "")
	if state == JobApplying {
		return j
	}
	if state == JobReconcileRequired && from == JobApplying {
		step(state, reason, "")
		return j
	}
	digest := strings.Repeat("d", 64)
	step(JobObserving, "native-observation", digest)
	if state != JobObserving {
		step(state, reason, digest)
	}
	return j
}

func TestRestartPageStates(t *testing.T) {
	cases := []struct {
		name                    string
		state, from, projected  JobState
		reason, projectedReason string
	}{
		{"queued", JobQueued, "", JobReconcileRequired, "admitted", "review-expired"},
		{"applying", JobApplying, "", JobReconcileRequired, "launch-intent", "executor-unverified"},
		{"observing", JobObserving, "", JobReconcileRequired, "native-observation", "executor-unverified"},
		{"succeeded", JobSucceeded, "", JobSucceeded, "verified", "verified"},
		{"failed", JobFailed, "", JobFailed, "verified-failure", "verified-failure"},
		{"cancelled", JobCancelled, "", JobCancelled, "operator-cancelled", "operator-cancelled"},
		{"expired", JobReconcileRequired, JobQueued, JobReconcileRequired, "review-expired", "review-expired"},
	}
	for _, from := range []JobState{JobApplying, JobObserving} {
		for _, reason := range []string{"ambiguous-effect", "storage-unavailable", "executor-unverified", "state-io-failure"} {
			cases = append(cases, struct {
				name                    string
				state, from, projected  JobState
				reason, projectedReason string
			}{string(from) + "/" + reason, JobReconcileRequired, from, JobReconcileRequired, reason, reason})
		}
	}
	for _, property := range []bool{false, true} {
		format := "legacy"
		if property {
			format = "property"
		}
		for _, tc := range cases {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				j := jobFixture()
				p := propertyJobFixture(t)
				if property {
					j = p.Job()
				}
				j = restartJobFixture(t, j, tc.state, tc.from, tc.reason)
				var value any = j
				if property {
					bound, err := BindPropertyJob(j, p.Intent())
					if err != nil {
						t.Fatal(err)
					}
					value = bound
				}
				data := reservationBytes(t, value)
				before := slices.Clone(data)
				view, err := BuildJobReservations([][]byte{data})
				if err != nil {
					t.Fatal(err)
				}
				holds := view.Pools()
				page, err := view.RestartPage("", 1)
				want := []JobRestartSummary{{ID: j.ID, SubmissionID: j.Intent.SubmissionID, PoolGUID: j.Intent.PoolGUID, Last: j.History[len(j.History)-1], State: tc.projected, Reason: tc.projectedReason}}
				if err != nil || !reflect.DeepEqual(page, want) {
					t.Fatal("incorrect restart state or lost durable observation", page, err)
				}
				if !slices.Equal(data, before) || !reflect.DeepEqual(holds, view.Pools()) {
					t.Fatal("projection rewrote history or released holds")
				}
			})
		}
	}
}

func TestRestartPageMixedOrderAndIsolation(t *testing.T) {
	p := propertyJobFixture(t)
	j := reservationJob(1, false)
	j.Intent.PoolGUID = p.job.Intent.PoolGUID
	data := [][]byte{reservationBytes(t, p), reservationBytes(t, j)}
	view, err := BuildJobReservations(data)
	if err != nil {
		t.Fatal(err)
	}
	before := view.Pools()
	page, err := view.RestartPage("", 64)
	if err != nil || len(page) != 2 || page[0].ID != j.ID || page[1].ID != p.job.ID {
		t.Fatal("mixed records were omitted or not sorted", page, err)
	}
	reversed, err := BuildJobReservations([][]byte{data[1], data[0]})
	if err != nil {
		t.Fatal(err)
	}
	again, err := reversed.RestartPage("", 64)
	if err != nil || !reflect.DeepEqual(page, again) {
		t.Fatal("input order changed projection", err)
	}
	page[0].ID = "changed"
	page[0].Last.State = JobSucceeded
	page[1].Reason = "changed"
	for _, record := range data {
		record[0] = 'x'
	}
	got, err := view.RestartPage("", 64)
	if err != nil || !reflect.DeepEqual(got, again) || !reflect.DeepEqual(before, view.Pools()) {
		t.Fatal("projection or input aliases view", err)
	}
	current, duplicate, err := view.CheckPropertySubmission(p)
	if err != nil || !duplicate || !reflect.DeepEqual(current, p) {
		t.Fatal("restart projection changed retry history", err)
	}
}

func TestRestartPageCeilingAndTraversal(t *testing.T) {
	data := make([][]byte, 1064)
	for i := range data {
		data[i] = reservationBytes(t, reservationJob(i, i >= 64))
	}
	slices.Reverse(data)
	view, err := BuildJobReservations(data)
	if err != nil {
		t.Fatal(err)
	}
	first, err := view.RestartPage("", 64)
	if err != nil || len(first) != 64 {
		t.Fatal("exact page ceiling", err)
	}
	var got []JobRestartSummary
	cursor := ""
	for {
		page, err := view.RestartPage(cursor, 37)
		if err != nil || page == nil || len(page) > 37 {
			t.Fatal("invalid bounded page", err)
		}
		if len(page) == 0 {
			break
		}
		got = append(got, page...)
		if len(got) > 1064 {
			t.Fatal("cursor failed to progress")
		}
		cursor = page[len(page)-1].ID
	}
	if len(got) != 1064 {
		t.Fatal("retained jobs dropped", len(got))
	}
	for i, row := range got {
		j := reservationJob(i, i >= 64)
		want := JobReconcileRequired
		if i >= 64 {
			want = JobCancelled
		}
		if row.ID != j.ID || row.Last.State != j.History[len(j.History)-1].State || row.State != want {
			t.Fatal("cursor skipped/repeated/replayed job", i, row)
		}
	}
	for _, i := range []int{0, 500, 1062, 1063} {
		page, err := view.RestartPage(got[i].ID, 1)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1063 {
			if page == nil || len(page) != 0 {
				t.Fatal("end cursor")
			}
		} else if len(page) != 1 || page[0] != got[i+1] {
			t.Fatal("exclusive cursor")
		}
	}
	if len(view.Pools()) != 1064 {
		t.Fatal("enumeration released retained holders")
	}
}

func TestRestartPageDenials(t *testing.T) {
	view, err := BuildJobReservations([][]byte{reservationBytes(t, jobFixture())})
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{-1, 0, 65, int(^uint(0) >> 1)} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			page, err := view.RestartPage("", limit)
			if !errors.Is(err, ErrLimit) || page != nil {
				t.Fatal("unbounded page accepted", err)
			}
		})
	}
	for _, cursor := range []string{"x", strings.Repeat("A", 64), strings.Repeat("a", 65), "../job", strings.Repeat("0", 64)} {
		t.Run(cursor, func(t *testing.T) {
			page, err := view.RestartPage(cursor, 1)
			if !errors.Is(err, ErrInvalid) || page != nil {
				t.Fatal("unknown/noncanonical cursor accepted", err)
			}
		})
	}
	if page, err := (JobReservations{}).RestartPage("", 1); !errors.Is(err, ErrInvalid) || page != nil {
		t.Fatal("unavailable view became empty success", err)
	}
	empty, err := BuildJobReservations(nil)
	if err != nil {
		t.Fatal(err)
	}
	if page, err := empty.RestartPage("", 64); err != nil || page == nil || len(page) != 0 {
		t.Fatal("complete empty view unavailable", err)
	}
	// A damaged later record cannot yield a prefix that looks like recovery.
	bad, err := BuildJobReservations([][]byte{reservationBytes(t, jobFixture()), []byte("{")})
	if err == nil {
		t.Fatal("damaged complete view accepted")
	}
	if page, err := bad.RestartPage("", 1); !errors.Is(err, ErrInvalid) || page != nil {
		t.Fatal("partial damaged view escaped", err)
	}
}

func TestRestartPageConcurrentReaders(t *testing.T) {
	view, err := BuildJobReservations([][]byte{reservationBytes(t, jobFixture()), reservationBytes(t, reservationJob(1, true))})
	if err != nil {
		t.Fatal(err)
	}
	want, err := view.RestartPage("", 64)
	if err != nil {
		t.Fatal(err)
	}
	holds := view.Pools()
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			for range 8 {
				page, err := view.RestartPage("", 64)
				if err != nil || !reflect.DeepEqual(page, want) {
					t.Error("concurrent projection", err)
				}
				if len(page) > 0 {
					page[0].Last.Reason = "changed"
				}
			}
		})
	}
	workers.Wait()
	if !reflect.DeepEqual(holds, view.Pools()) {
		t.Fatal("concurrent inspection released holds")
	}
}

func FuzzJobRestartPage(f *testing.F) {
	p := propertyJobFixture(f)
	f.Add(reservationBytes(f, p), "", 64)
	f.Add(reservationBytes(f, restartJobFixture(f, jobFixture(), JobObserving, "", "")), "", 1)
	f.Add(reservationBytes(f, reservationJob(1, true)), reservationJob(1, true).ID, 1)
	f.Add(reservationBytes(f, jobFixture()), strings.Repeat("0", 64), 1)
	f.Add([]byte("{}"), "", 65)
	f.Fuzz(func(t *testing.T, data []byte, cursor string, limit int) {
		view, decodeErr := BuildJobReservations([][]byte{data})
		holds := view.Pools()
		page, err := view.RestartPage(cursor, limit)
		if decodeErr != nil || limit < 1 || limit > 64 || cursor != "" && (len(holds) == 0 || cursor != holds[0].Jobs[0].ID) {
			if err == nil || page != nil {
				t.Fatal("invalid request exposed partial recovery", err)
			}
			return
		}
		if err != nil || page == nil || len(page) > 1 {
			t.Fatal("valid one-record view unavailable", err)
		}
		if cursor != "" && len(page) != 0 || cursor == "" && len(page) != 1 {
			t.Fatal("cursor repeated or dropped record")
		}
		for _, row := range page {
			if row.ID != holds[0].Jobs[0].ID || row.PoolGUID != holds[0].PoolGUID || row.Last.State != holds[0].Jobs[0].State || row.Last.Revision != holds[0].Jobs[0].Revision {
				t.Fatal("lost durable identity/history")
			}
			if row.State == JobQueued || row.State == JobApplying || row.State == JobObserving {
				t.Fatal("restart presentation enabled replay")
			}
		}
		again, err := view.RestartPage(cursor, limit)
		if err != nil || !reflect.DeepEqual(page, again) || !reflect.DeepEqual(holds, view.Pools()) {
			t.Fatal("projection changed retained view", err)
		}
	})
}
