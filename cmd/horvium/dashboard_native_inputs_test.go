// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

func nativeWitnessTSV(s inventory.Snapshot) []byte {
	var b strings.Builder
	for _, d := range s.Datasets.Data[:100] {
		fmt.Fprintf(&b, "%s\t%s\t%d\t%d\t%d\n", d.Name, d.GUID, d.Used, d.Available, d.Referenced)
	}
	return []byte(b.String())
}
func TestDashboardNativeInputsOwnOriginalAndObserveAllWitnesses(t *testing.T) {
	original := adapterNativeInventory()
	prepared, err := dashboardPrepareNativeInputs("native-small", original, nativeWitnessTSV(original))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.fixture.OwnedDatasets != 100 || prepared.count != 103 {
		t.Fatal("scope")
	}
	original.Datasets.Data[0].GUID = "99999"
	if prepared.fixture.InventorySHA256 == "" {
		t.Fatal("no original digest")
	}
	// The previously prepared oracle must still require the original GUID.
	expected := adapterNativeInventory()
	other, err := dashboardPrepareNativeInputs("native-small", expected, nativeWitnessTSV(expected))
	if err != nil || prepared.fixture != other.fixture {
		t.Fatal("mutable original")
	}
}
func TestDashboardNativeInputsRejectMalformedOrChangedWitnesses(t *testing.T) {
	s := adapterNativeInventory()
	good := string(nativeWitnessTSV(s))
	for _, bad := range []string{"", good + strings.Split(good, "\n")[0] + "\n", strings.Replace(good, "\t100\t0\t", "\t999999\t0\t", 1), strings.Replace(good, "\t100\t0\t", "\t100\t1\t", 1), strings.Replace(good, "\t100\t0\t", "\t100\tNaN\t", 1)} {
		if p, err := dashboardPrepareNativeInputs("native-small", s, []byte(bad)); err == nil || p != nil {
			t.Fatal("accepted incomplete, duplicate, changed or malformed independent observation")
		}
	}
	s.Pools.Status = inventory.Unavailable
	if _, err := dashboardPrepareNativeInputs("native-small", s, []byte(good)); err == nil {
		t.Fatal("partial original")
	}
}
func TestDashboardProjectionKeepsNativeFactsAndDeclaresOnlyDatasetProjection(t *testing.T) {
	original := adapterNativeInventory()
	before, _ := inventory.CloneBounded(original, inventory.DefaultLimits())
	prepared, err := dashboardPrepareNativeInputs("projection-10000", original, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := before
	expected.Datasets.Data = adapterProjection().Datasets.Data
	oracle, err := newDashboardPreparedFixture("projection-10000", expected, nil)
	if err != nil || prepared.fixture != oracle.fixture {
		t.Fatal("native facts replaced or wrong projected values", err)
	}
	if !reflect.DeepEqual(original, before) {
		t.Fatal("projection mutated original")
	}
	for _, mode := range []string{"synthetic-100", "unknown"} {
		if _, err := dashboardPrepareNativeInputs(mode, original, nil); err == nil {
			t.Fatal("invalid native mode")
		}
	}
	if _, err := dashboardPrepareNativeInputs("projection-10000", original, []byte("unexpected")); err == nil {
		t.Fatal("owned projection")
	}
}
func TestDashboardWorkloadRequiresThreeCompletedOverlappingReplacements(t *testing.T) {
	running := make(chan struct{})
	finish := make(chan struct{})
	calls := 0
	load := func(phase string) (*dashboardPhaseReceipt, []dashboardSample, error) {
		if phase == "replacement" {
			close(running)
			<-finish
		}
		raw := adapterSamples()
		r, e := dashboardBuildPhaseReceipt(raw, phase, false)
		return r, raw, e
	}
	replace := func() error {
		<-running
		calls++
		if calls == 3 {
			defer func() { go func() { time.Sleep(20 * time.Millisecond); close(finish) }() }()
		}
		return nil
	}
	result, err := dashboardMeasureWorkload(load, replace)
	if err != nil || calls != 3 || result.Replacements != 3 || len(result.InitialRaw) != 1000 || len(result.ReplacementRaw) != 1000 || result.Initial == nil || result.Replacement == nil {
		t.Fatal("missing complete phases", err)
	}
}
func TestDashboardWorkloadRetainsOutcomesOnLateOrFailedReplacement(t *testing.T) {
	for _, failure := range []string{"late", "callback"} {
		t.Run(failure, func(t *testing.T) {
			released := make(chan struct{})
			completed := make(chan struct{})
			load := func(phase string) (*dashboardPhaseReceipt, []dashboardSample, error) {
				if phase == "replacement" {
					<-released
					close(completed)
				}
				raw := adapterSamples()
				r, e := dashboardBuildPhaseReceipt(raw, phase, false)
				return r, raw, e
			}
			result, err := dashboardMeasureWorkload(load, func() error {
				close(released)
				<-completed
				if failure == "callback" {
					return errors.New("replacement failure")
				}
				time.Sleep(20 * time.Millisecond)
				return nil
			})
			if err == nil || len(result.ReplacementRaw) != 1000 || result.Initial == nil {
				t.Fatal("late/failed replacement passed or lost raw outcomes")
			}
		})
	}
}
func TestDashboardWorkloadDoesNotReplaceAfterFailedInitial(t *testing.T) {
	calls := 0
	result, err := dashboardMeasureWorkload(func(phase string) (*dashboardPhaseReceipt, []dashboardSample, error) {
		return nil, adapterSamples(), errors.New("initial failed")
	}, func() error { calls++; return nil })
	if err == nil || calls != 0 || len(result.InitialRaw) != 1000 || result.Replacement != nil {
		t.Fatal("failed initial retried or discarded")
	}
}

func TestDashboardReplacementReturnsControlFailuresWithoutLosingWorkload(t *testing.T) {
	for _, status := range []int{403, 500} {
		client := &http.Client{Transport: dashboardTransport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("denied"))}, nil
		})}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := dashboardReplaceGeneration(ctx, client, "http://127.0.0.1:8080", "fixture", 100, "old")
		cancel()
		if err == nil {
			t.Fatal("control denial accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := dashboardReplaceGeneration(ctx, &http.Client{}, "http://127.0.0.1:8080", "fixture", 100, "old"); err == nil {
		t.Fatal("cancelled control passed")
	}
}
func TestDashboardReplacementWaitsForCompleteNewGeneration(t *testing.T) {
	reads, posts := 0, 0
	client := &http.Client{Transport: dashboardTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method == "POST" {
			posts++
			if r.URL.Path != "/api/refresh" {
				t.Error("unexpected controller route")
			}
			return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		}
		reads++
		body := `{"generation":"new","complete":true,"DatasetCount":100,"datasets":{"stale":false}}`
		if reads == 1 {
			body = `{"generation":"old","complete":true,"DatasetCount":100,"datasets":{"stale":false}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	generation, err := dashboardReplaceGeneration(ctx, client, "http://127.0.0.1:8080", "fixture", 100, "old")
	if err != nil || generation != "new" || posts != 1 || reads != 2 {
		t.Fatal("incomplete replacement", err, posts, reads)
	}
}
