// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gonz0w/horvium/internal/observation"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// Original native observations and separate owned zfs TSV are controller inputs,
// never management HTML/API responses. All preparation precedes timed requests.
func dashboardPrepareNativeInputs(mode string, original inventory.Snapshot, ownedTSV []byte) (*dashboardPreparedFixture, error) {
	s, err := inventory.CloneBounded(original, inventory.DefaultLimits())
	if err != nil || !s.Complete() {
		return nil, errors.New("incomplete original native observation")
	}
	var owned []inventory.Dataset
	switch mode {
	case "native-small":
		owned, err = inventory.ParseDatasets(ownedTSV, inventory.DefaultLimits())
		if err != nil || len(owned) != 100 {
			return nil, errors.New("incomplete independent owned dataset observation")
		}
		originals := make(map[string]inventory.Dataset, len(s.Datasets.Data))
		for _, d := range s.Datasets.Data {
			originals[d.Name] = d
		}
		for _, d := range owned {
			if originals[d.Name] != d {
				return nil, errors.New("independent owned dataset values changed")
			}
		}
	case "projection-10000":
		if len(ownedTSV) != 0 {
			return nil, errors.New("projection cannot own native datasets")
		}
		s.Datasets.Data = make([]inventory.Dataset, 10000)
		for i := range s.Datasets.Data {
			s.Datasets.Data[i] = inventory.Dataset{Name: fmt.Sprintf("fixture/dataset-%06d", i), GUID: fmt.Sprint(i + 100), Used: uint64(i), Available: 1 << 40, Referenced: uint64(i)}
		}
	default:
		return nil, errors.New("invalid native resource mode")
	}
	return newDashboardPreparedFixture(mode, s, owned)
}

type dashboardMeasuredWorkload struct {
	Initial        *dashboardPhaseReceipt `json:"initial"`
	Replacement    *dashboardPhaseReceipt `json:"replacement"`
	InitialRaw     []dashboardSample      `json:"-"`
	ReplacementRaw []dashboardSample      `json:"-"`
	Replacements   int                    `json:"replacements"`
}
type dashboardPhaseLoader func(string) (*dashboardPhaseReceipt, []dashboardSample, error)

// The native caller supplies the mandatory prepared predicate via its loader.
// No workload pacing/retries are added to make a slow replacement appear active.
func dashboardMeasureWorkload(load dashboardPhaseLoader, replace func() error) (dashboardMeasuredWorkload, error) {
	var result dashboardMeasuredWorkload
	if load == nil || replace == nil {
		return result, errors.New("missing workload or replacement controller")
	}
	var err error
	result.Initial, result.InitialRaw, err = load("initial")
	if err != nil {
		return result, err
	}
	type finished struct {
		receipt *dashboardPhaseReceipt
		raw     []dashboardSample
		err     error
	}
	done := make(chan finished, 1)
	started := make(chan struct{})
	var active atomic.Bool
	go func() {
		active.Store(true)
		close(started)
		receipt, raw, e := load("replacement")
		active.Store(false)
		done <- finished{receipt, raw, e}
	}()
	<-started
	for i := 0; i < 3; i++ {
		if !active.Load() {
			err = errors.New("replacement workload finished before all replacements")
			break
		}
		if e := replace(); e != nil {
			err = e
			break
		}
		result.Replacements++
		if !active.Load() {
			err = errors.New("replacement completed after workload ended")
			break
		}
	}
	observed := <-done
	result.Replacement, result.ReplacementRaw = observed.receipt, observed.raw
	return result, errors.Join(err, observed.err)
}

// Partial fixture output, deliberately distinct from full DASHBOARD_RESOURCE.
// Failed observations are sanitized and retained; this is never qualification.
func dashboardRawWire(samples []dashboardSample) []dashboardWireSample {
	out := make([]dashboardWireSample, len(samples))
	for i, s := range samples {
		out[i] = dashboardWireSample{s.Index, s.Case, s.Outcome, s.Status, int64(s.Duration), s.BodyBytes, s.Predicate}
	}
	return out
}

// Bounded replacement control is outside the 1,000 measured HTML requests.
// Return failures to the coordinator so completed raw outcomes can be published.
func dashboardReplaceGeneration(ctx context.Context, client *http.Client, base, csrf string, count int, previous string) (string, error) {
	wait := func() error {
		timer := time.NewTimer(20 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	request := func(method, path, body string) (int, []byte, error) {
		req, err := http.NewRequestWithContext(ctx, method, base+path, strings.NewReader(body))
		if err != nil {
			return 0, nil, err
		}
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		response, err := client.Do(req)
		if err != nil {
			return 0, nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, 65537))
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil || len(data) > 65536 {
			return 0, nil, errors.New("incomplete replacement control body")
		}
		return response.StatusCode, data, nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		status, _, err := request(http.MethodPost, "/api/refresh", url.Values{"csrf": {csrf}}.Encode())
		if err != nil {
			return "", err
		}
		if status == http.StatusAccepted {
			break
		}
		if status != http.StatusConflict {
			return "", errors.New("replacement refresh denied")
		}
		if err := wait(); err != nil {
			return "", err
		}
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		status, data, err := request(http.MethodGet, "/api/overview", "")
		if err != nil {
			return "", err
		}
		if status != http.StatusServiceUnavailable {
			var o observation.Overview
			if status != http.StatusOK || json.Unmarshal(data, &o) != nil {
				return "", errors.New("replacement overview schema")
			}
			if o.Generation != "" && o.Generation != previous && o.Complete && o.DatasetCount == count && !o.Datasets.Stale {
				return o.Generation, nil
			}
		}
		if err := wait(); err != nil {
			return "", err
		}
	}
}
