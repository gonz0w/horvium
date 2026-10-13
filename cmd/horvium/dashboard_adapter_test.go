// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

func adapterNativeInventory() inventory.Snapshot {
	s := performanceSnapshot(103, 0)
	s.Host.Data.RunningKernel = "original-fixture"
	return s
}
func adapterWitness(s inventory.Snapshot) []inventory.Dataset {
	return slices.Clone(s.Datasets.Data[:100])
}
func adapterProjection() inventory.Snapshot {
	s := performanceSnapshot(10000, 0)
	for i := range s.Datasets.Data {
		s.Datasets.Data[i].Name = fmt.Sprintf("fixture/dataset-%06d", i)
	}
	return s
}
func adapterSamples() []dashboardSample {
	samples := dashboardGoodSamples()
	for i := range samples {
		samples[i].BodyBytes = 4096
		samples[i].Predicate = "original-inventory"
	}
	return samples
}
func TestDashboardAdapterPreparesFourModesAndOwnsOriginals(t *testing.T) {
	for _, mode := range []string{"synthetic-100", "synthetic-10000", "native-small", "projection-10000"} {
		t.Run(mode, func(t *testing.T) {
			s := performanceSnapshot(100, 0)
			var owned []inventory.Dataset
			switch mode {
			case "synthetic-10000":
				s = performanceSnapshot(10000, 0)
			case "native-small":
				s = adapterNativeInventory()
				owned = adapterWitness(s)
			case "projection-10000":
				s = adapterProjection()
			}
			before, _ := json.Marshal(s)
			prepared, err := newDashboardPreparedFixture(mode, s, owned)
			if err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(s)
			if string(before) != string(after) {
				t.Fatal("preparation changed original")
			}
			reordered, _ := inventory.CloneBounded(s, inventory.DefaultLimits())
			slices.Reverse(reordered.Pools.Data)
			slices.Reverse(reordered.Datasets.Data)
			slices.Reverse(reordered.GEOM.Data.Providers)
			reordered.FinishedAt = reordered.FinishedAt.Add(time.Hour)
			reordered.Datasets.StartedAt = reordered.Datasets.StartedAt.Add(time.Minute)
			reordered.Datasets.FinishedAt = reordered.Datasets.FinishedAt.Add(time.Minute)
			other, err := newDashboardPreparedFixture(mode, reordered, owned)
			if err != nil || prepared.fixture != other.fixture {
				t.Fatal("order/time changed fixture digest", err)
			}
			oldDigest := prepared.fixture.InventorySHA256
			if len(oldDigest) != 64 || prepared.fixture.Datasets != len(s.Datasets.Data) || len(prepared.plan) != 1000 {
				t.Fatal("incomplete preparation")
			}
			// Caller mutation cannot change expectations, plan or digest.
			s.Datasets.Data[0].Used++
			*s.Host.Data.PhysicalMemory = 1
			if len(owned) > 0 {
				owned[0].GUID = "999999"
			}
			if prepared.fixture.InventorySHA256 != oldDigest {
				t.Fatal("aliased digest")
			}
		})
	}
}
func TestDashboardAdapterRejectsWrongModeOriginalAndWitness(t *testing.T) {
	native := adapterNativeInventory()
	tests := []struct {
		name, mode string
		s          inventory.Snapshot
		owned      []inventory.Dataset
	}{
		{"unknown", "future", performanceSnapshot(100, 0), nil},
		{"wrong-small", "synthetic-100", performanceSnapshot(101, 0), nil},
		{"wrong-large", "synthetic-10000", performanceSnapshot(100, 0), nil},
		{"owned-synthetic", "synthetic-100", performanceSnapshot(100, 0), adapterWitness(native)},
		{"missing-native-witness", "native-small", native, nil},
		{"short-native-witness", "native-small", native, adapterWitness(native)[:99]},
		{"wrong-projection", "projection-10000", performanceSnapshot(10000, 0), nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newDashboardPreparedFixture(test.mode, test.s, test.owned); err == nil {
				t.Fatal("invalid mode/original accepted")
			}
		})
	}
	for _, change := range []func(*inventory.Snapshot, []inventory.Dataset){
		func(s *inventory.Snapshot, w []inventory.Dataset) { s.Pools.Status = inventory.Unavailable },
		func(s *inventory.Snapshot, w []inventory.Dataset) { s.Datasets.Data[1] = s.Datasets.Data[0] },
		func(s *inventory.Snapshot, w []inventory.Dataset) { w[1] = w[0] },
		func(s *inventory.Snapshot, w []inventory.Dataset) { w[0].GUID = "999999" },
		func(s *inventory.Snapshot, w []inventory.Dataset) { w[0].Name = "wrong-original" },
	} {
		s := adapterNativeInventory()
		w := adapterWitness(s)
		change(&s, w)
		if _, err := newDashboardPreparedFixture("native-small", s, w); err == nil {
			t.Fatal("invalid original/witness accepted")
		}
	}
	s := performanceSnapshot(100, 0)
	s.Datasets.Data[0].Used++
	if _, err := newDashboardPreparedFixture("synthetic-100", s, nil); err == nil {
		t.Fatal("changed declared synthetic values accepted")
	}
}
func TestDashboardAdapterReceiptPreservesSamplesAndSeparatesExpiry(t *testing.T) {
	samples := adapterSamples()
	before := slices.Clone(samples)
	phase, err := dashboardBuildPhaseReceipt(samples, "initial", false)
	if err != nil || !slices.Equal(samples, before) || phase.Summary["all"].Success != 1000 || phase.Summary["all"].ExpiryLatency != nil {
		t.Fatal("initial receipt", err)
	}
	if phase.Samples[0].DurationNS != int64(samples[0].Duration) || phase.Samples[0].BodyBytes != 4096 {
		t.Fatal("raw sample changed")
	}
	samples[0].Duration = time.Nanosecond
	samples[0].BodyBytes = 1
	if phase.Samples[0].DurationNS != int64(before[0].Duration) || phase.Samples[0].BodyBytes != 4096 {
		t.Fatal("aliased raw receipt")
	}
	samples = adapterSamples()
	samples[0].Outcome = "expiry"
	samples[0].Status = 409
	samples[0].Predicate = "safe-expiry"
	samples[0].Duration = time.Nanosecond
	phase, err = dashboardBuildPhaseReceipt(samples, "replacement", false)
	if err != nil || phase.Summary["overview"].Expiry != 1 || phase.Summary["overview"].Success != 199 || phase.Summary["overview"].ExpiryLatency.P95NS != 1 {
		t.Fatal("expiry receipt", err)
	}
}
func TestDashboardAdapterReceiptRejectsUnsafeAndIncompleteSamples(t *testing.T) {
	for _, mutate := range []func([]dashboardSample){
		func(s []dashboardSample) { s[0].BodyBytes = 0 }, func(s []dashboardSample) { s[0].BodyBytes = dashboardBodyLimit + 1 },
		func(s []dashboardSample) { s[0].Predicate = "" }, func(s []dashboardSample) { s[0].Predicate = "safe-expiry" },
		func(s []dashboardSample) { s[0].Outcome = "failure" }, func(s []dashboardSample) { s[0].Status = 403 },
		func(s []dashboardSample) { s[0].Index = 1 }, func(s []dashboardSample) { s[0].Duration = 0 },
	} {
		s := adapterSamples()
		mutate(s)
		if r, err := dashboardBuildPhaseReceipt(s, "initial", false); err == nil || r != nil {
			t.Fatal("invalid samples emitted passing receipt")
		}
	}
	s := adapterSamples()
	for i := range s {
		if s[i].Case == "datasets-maximum" {
			s[i].Duration = 250 * time.Millisecond
		}
	}
	if r, err := dashboardBuildPhaseReceipt(s, "initial", true); err == nil || r != nil {
		t.Fatal("per-case budget violation hidden")
	}
	s = adapterSamples()
	s[0].Outcome = "expiry"
	s[0].Status = 409
	s[0].Predicate = "original-inventory"
	if _, err := dashboardBuildPhaseReceipt(s, "replacement", false); err == nil {
		t.Fatal("unsafe expiry emitted")
	}
}
func TestDashboardAdapterLoadsActualConsoleBothPhasesWithoutAPI(t *testing.T) {
	s := adapterNativeInventory()
	prepared, err := newDashboardPreparedFixture("native-small", s, adapterWitness(s))
	if err != nil {
		t.Fatal(err)
	}
	h, _, cookie := dashboardHTMLFixture(t, s)
	// Verify immutable original expectations, not only the digest field.
	s.Datasets.Data[0].Used++
	*s.Host.Data.PhysicalMemory = 1
	var requests, api atomic.Int32
	client := &http.Client{Transport: dashboardTransport(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		if strings.HasPrefix(request.URL.Path, "/api/") {
			api.Add(1)
		}
		r := httptest.NewRequest(request.Method, request.URL.RequestURI(), nil).WithContext(request.Context())
		r.Header = request.Header.Clone()
		r.Host = "127.0.0.1:8088"
		r.RemoteAddr = "127.0.0.1:50000"
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Result(), nil
	})}
	for _, phase := range []string{"initial", "replacement"} {
		receipt, samples, err := prepared.load(client, "http://127.0.0.1:8088", phase)
		if err != nil || len(samples) != 1000 || receipt.Summary["all"].Success != 1000 {
			t.Fatal("actual workload", err)
		}
		for _, sample := range samples {
			if sample.BodyBytes <= 0 || sample.Predicate != "original-inventory" {
				t.Fatal("missing observed body/predicate")
			}
		}
		for _, name := range dashboardCases {
			if receipt.Summary[name].Success != 200 {
				t.Fatal("missing case")
			}
		}
	}
	if requests.Load() != 2000 || api.Load() != 0 {
		t.Fatal("hidden requests", requests.Load(), api.Load())
	}
}
func TestDashboardAdapterGoReceiptAcceptedByIndependentPythonContract(t *testing.T) {
	python, err := filepath.Abs("../../.tools/venv/bin/python3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(python); err != nil {
		t.Skip("portable pinned-Python contract check; not native qualification")
	}
	module, err := filepath.Abs("../../tools/dashboard-resource-evidence.py")
	if err != nil {
		t.Fatal(err)
	}
	script := `import importlib.util,json,sys
spec=importlib.util.spec_from_file_location('validator',sys.argv[1]);module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
module.validate_phase(sys.argv[2],json.load(sys.stdin),False)
`
	for _, phase := range []string{"initial", "replacement"} {
		samples := adapterSamples()
		if phase == "replacement" {
			samples[0].Outcome = "expiry"
			samples[0].Status = 409
			samples[0].Predicate = "safe-expiry"
		}
		receipt, err := dashboardBuildPhaseReceipt(samples, phase, false)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command(python, "-c", script, module, phase)
		command.Stdin = strings.NewReader(string(encoded))
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatal("independent contract rejected Go JSON", err, string(output))
		}
	}
}
func TestDashboardAdapterOriginalChangesFailRatherThanRebaseline(t *testing.T) {
	original := adapterNativeInventory()
	prepared, err := newDashboardPreparedFixture("native-small", original, adapterWitness(original))
	if err != nil {
		t.Fatal(err)
	}
	changed, _ := inventory.CloneBounded(original, inventory.DefaultLimits())
	changed.Pools.Data[0].Allocated = 1
	changed.Pools.Data[0].Free--
	h, _, cookie := dashboardHTMLFixture(t, changed)
	response := dashboardHTMLRequest(h, cookie, "GET", prepared.plan[0].Path, "")
	if err := prepared.predicate(prepared.plan[0], response.Code, response.Body.Bytes()); err == nil {
		t.Fatal("changed source silently rebaselined")
	}
	if reflect.DeepEqual(original.Pools.Data, changed.Pools.Data) {
		t.Fatal("ineffective mutation")
	}
}

func TestDashboardAdapterCanonicalDigestHandlesOpaqueGEOMIdentities(t *testing.T) {
	s := adapterNativeInventory()
	ids := map[string]string{"1": "0xclass", "2": "0xb", "3": "0xa", "4": "0xd", "5": "0xc", "6": "0xf", "7": "0xe", "8": "0xh", "9": "0xg"}
	for i := range s.GEOM.Data.Classes {
		s.GEOM.Data.Classes[i].ID = ids[s.GEOM.Data.Classes[i].ID]
	}
	for i := range s.GEOM.Data.Geoms {
		s.GEOM.Data.Geoms[i].ID = ids[s.GEOM.Data.Geoms[i].ID]
		s.GEOM.Data.Geoms[i].ClassID = "0xclass"
	}
	for i := range s.GEOM.Data.Providers {
		s.GEOM.Data.Providers[i].ID = ids[s.GEOM.Data.Providers[i].ID]
		s.GEOM.Data.Providers[i].GeomID = ids[s.GEOM.Data.Providers[i].GeomID]
	}
	before, err := newDashboardPreparedFixture("native-small", s, adapterWitness(s))
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(s.GEOM.Data.Geoms)
	slices.Reverse(s.GEOM.Data.Providers)
	after, err := newDashboardPreparedFixture("native-small", s, adapterWitness(s))
	if err != nil || before.fixture != after.fixture {
		t.Fatal("opaque identities made order-dependent digest", err)
	}
}

func TestDashboardAdapterLoadRetainsFailedSamplesWithoutPassingReceipt(t *testing.T) {
	original := adapterNativeInventory()
	prepared, err := newDashboardPreparedFixture("native-small", original, adapterWitness(original))
	if err != nil {
		t.Fatal(err)
	}
	changed, _ := inventory.CloneBounded(original, inventory.DefaultLimits())
	changed.Pools.Data[0].Allocated = 1
	changed.Pools.Data[0].Free--
	h, _, cookie := dashboardHTMLFixture(t, changed)
	var requests atomic.Int32
	client := &http.Client{Transport: dashboardTransport(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		r := httptest.NewRequest(request.Method, request.URL.RequestURI(), nil)
		r.Host = "127.0.0.1:8088"
		r.RemoteAddr = "127.0.0.1:50000"
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Result(), nil
	})}
	receipt, samples, err := prepared.load(client, "http://127.0.0.1:8088", "initial")
	if receipt != nil || err == nil || len(samples) != 1000 || requests.Load() != 1000 {
		t.Fatal("failed workload lost or emitted a pass", err)
	}
	failures := 0
	for _, sample := range samples {
		if sample.Outcome == "failure" {
			failures++
			if sample.Predicate != "failure:table" || sample.BodyBytes <= 0 {
				t.Fatal("failed table predicate lost its bounded diagnostic")
			}
		}
	}
	if failures != 600 {
		t.Fatal("wrong failure classification", failures)
	}
	if receipt, samples, err := (*dashboardPreparedFixture)(nil).load(client, "http://127.0.0.1:8088", "initial"); receipt != nil || samples != nil || err == nil {
		t.Fatal("missing preparation accepted")
	}
}

func TestDashboardAdapterLoadSeparatesActualSafeExpiry(t *testing.T) {
	original := adapterNativeInventory()
	prepared, err := newDashboardPreparedFixture("native-small", original, adapterWitness(original))
	if err != nil {
		t.Fatal(err)
	}
	h, m, cookie := dashboardHTMLFixture(t, original)
	old := m.State().CurrentID
	for i := 0; i < 2; i++ {
		previous := m.State().CurrentID
		if err := m.Refresh(); err != nil {
			t.Fatal(err)
		}
		dashboardHTMLWait(t, func() bool { return m.State().CurrentID != previous })
	}
	expired := dashboardHTMLRequest(h, cookie, "GET", "/?generation="+old, "")
	if expired.Code != 409 {
		t.Fatal("expiry fixture")
	}
	body := expired.Body.String()
	var overviews atomic.Int32
	client := &http.Client{Transport: dashboardTransport(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/" && overviews.Add(1)%2 == 0 {
			return &http.Response{StatusCode: 409, Header: expired.Header().Clone(), Body: io.NopCloser(strings.NewReader(body))}, nil
		}
		r := httptest.NewRequest(request.Method, request.URL.RequestURI(), nil)
		r.Host = "127.0.0.1:8088"
		r.RemoteAddr = "127.0.0.1:50000"
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Result(), nil
	})}
	receipt, samples, err := prepared.load(client, "http://127.0.0.1:8088", "replacement")
	if err != nil || len(samples) != 1000 || receipt.Summary["all"].Expiry != 100 || receipt.Summary["overview"].Success != 100 {
		t.Fatal("safe actual expiry was combined with success", err)
	}
	for _, sample := range samples {
		if sample.Outcome == "expiry" && (sample.Predicate != "safe-expiry" || sample.BodyBytes != len(body)) {
			t.Fatal("expiry receipt lost original body size or predicate")
		}
	}
}
