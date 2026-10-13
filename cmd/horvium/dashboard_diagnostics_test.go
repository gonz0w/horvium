// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardFailureDiagnosticsRemainBoundedAndFailClosed(t *testing.T) {
	plan, err := dashboardPlan(100)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name      string
		status    int
		body      string
		predicate error
		want      string
	}{
		{"original table", 200, "fixture", errors.New("wrong original dashboard table"), "failure:table"},
		{"unknown table check", 200, "fixture", dashboardTableMismatch("private-token"), "failure:table"},
		{"original comparisons", 200, "fixture", errors.New("wrong original dashboard comparisons"), "failure:comparisons"},
		{"freshness", 200, "fixture", errors.New("wrong storage query or freshness metadata"), "failure:storage-metadata"},
		{"unknown sensitive error", 200, "fixture", errors.New("session=private-token inventory=private-name"), "failure:predicate"},
		{"denied status", 403, "fixture", nil, "failure:status"},
		{"incomplete document", 200, "fixture", errors.New("malformed or unbounded dashboard document"), "failure:document"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(c.status); _, _ = w.Write([]byte(c.body)) }))
			defer server.Close()
			samples, err := dashboardLoad(server.Client(), server.URL, 100, plan, "initial", func(dashboardRequest, int, []byte) error { return c.predicate })
			if err == nil || len(samples) != 1000 {
				t.Fatal("failed workload accepted or incomplete diagnostic retention")
			}
			for _, sample := range samples {
				if sample.Outcome != "failure" || sample.Predicate != c.want || strings.Contains(sample.Predicate, "private") {
					t.Fatalf("unsafe/missing classification: %q", sample.Predicate)
				}
			}
			if receipt, err := dashboardBuildPhaseReceipt(samples, "initial", false); err == nil || receipt != nil {
				t.Fatal("failure diagnostic accepted as successful receipt")
			}
		})
	}
}

func TestDashboardFailureDiagnosticsPreserveSuccessAndSafeExpiry(t *testing.T) {
	request := dashboardRequest{Case: "overview"}
	for _, c := range []struct {
		status                int
		phase, outcome, label string
	}{{200, "initial", "success", "original-inventory"}, {409, "replacement", "expiry", "safe-expiry"}, {409, "initial", "failure", "failure:status"}} {
		outcome, label := dashboardClassifyResponse(request, c.phase, c.status, []byte("complete"), nil, nil, func(dashboardRequest, int, []byte) error { return nil })
		if outcome != c.outcome || label != c.label {
			t.Fatal("changed acceptance outcome", outcome, label)
		}
	}
	for _, c := range []struct {
		transport, body error
		want            string
	}{{errors.New("secret transport"), nil, "failure:transport"}, {nil, errors.New("secret body"), "failure:body"}} {
		outcome, label := dashboardClassifyResponse(request, "initial", 200, nil, c.transport, c.body, func(dashboardRequest, int, []byte) error {
			t.Fatal("predicate invoked for incomplete response")
			return nil
		})
		if outcome != "failure" || label != c.want {
			t.Fatal("unsafe failure")
		}
	}
}

func TestDashboardDiagnosticOracleKeepsOriginalPoolValues(t *testing.T) {
	original := performanceSnapshot(100, 0)
	plan, err := dashboardPlan(100)
	if err != nil {
		t.Fatal(err)
	}
	predicate, err := newDashboardHTMLPredicate(original, plan)
	if err != nil {
		t.Fatal(err)
	}
	handler, _, cookie := dashboardHTMLFixture(t, original)
	for _, request := range plan[:5] {
		response := dashboardHTMLRequest(handler, cookie, "GET", request.Path, "")
		if err := predicate(request, response.Code, response.Body.Bytes()); err != nil {
			t.Fatal("unchanged original fixture rejected", request.Case, err)
		}
	}
	changed, err := inventory.CloneBounded(original, inventory.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	changed.Pools.Data[0].Allocated++
	changed.Pools.Data[0].Free--
	other, _, otherCookie := dashboardHTMLFixture(t, changed)
	for _, request := range plan[:5] {
		response := dashboardHTMLRequest(other, otherCookie, "GET", request.Path, "")
		err := predicate(request, response.Code, response.Body.Bytes())
		if strings.HasPrefix(request.Case, "datasets-") {
			if err != nil {
				t.Fatal("unchanged dataset facts rejected", err)
			}
		} else {
			if err == nil || dashboardFailureClassification(err) != "failure:table-cell-allocated" {
				t.Fatal("changed original pool accounting accepted or misclassified", request.Case, err)
			}
		}
	}
	if original.Pools.Data[0].Allocated == changed.Pools.Data[0].Allocated {
		t.Fatal("original oracle mutated")
	}
}

// Mutate parsed real responses, retaining the original independent expectation.
// Each failure must identify a fixed check, never a row identity or observed value.
func TestDashboardTableDiagnosticsIdentifyExactChecks(t *testing.T) {
	original := performanceSnapshot(100, 0)
	h, _, cookie := dashboardHTMLFixture(t, original)
	pool := original.Pools.Data[0]
	expected := dashboardHTMLPage{path: "/storage/pools", title: "Pools", caption: "Pools in this observation", limit: 100, rows: []dashboardHTMLRow{{name: pool.Name, guid: pool.GUID, cells: []string{pool.GUID, dashboardBytes(pool.Size), dashboardBytes(pool.Allocated), dashboardBytes(pool.Free), pool.Health}}}}
	cases := []struct {
		name, want string
		mutate     func(*dashboardHTMLNode)
	}{
		{"structure", "structure", func(n *dashboardHTMLNode) { n.children = nil }},
		{"header count", "header-count", func(n *dashboardHTMLNode) { n.direct("thead")[0].children = nil }},
		{"header text", "header", func(n *dashboardHTMLNode) { n.direct("thead")[0].find("th", "", "")[0].content = nil }},
		{"row count", "row-count", func(n *dashboardHTMLNode) { n.direct("tbody")[0].children = nil }},
		{"row shape", "row-shape", func(n *dashboardHTMLNode) { n.direct("tbody")[0].direct("tr")[0].attrs["class"] = "private-name" }},
		{"row name", "row-name", func(n *dashboardHTMLNode) {
			n.direct("tbody")[0].find("a", "", "")[0].content = []dashboardHTMLContent{{text: "private-name"}}
		}},
		{"row link", "row-link", func(n *dashboardHTMLNode) { n.direct("tbody")[0].find("a", "", "")[0].attrs["href"] = "private-token" }},
		{"row selection", "row-selection", func(n *dashboardHTMLNode) { n.direct("tbody")[0].find("a", "", "")[0].attrs["id"] = "private-id" }},
		{"cell style", "cell-style", func(n *dashboardHTMLNode) {
			n.direct("tbody")[0].find("td", "", "")[0].attrs["class"] = "private-class"
		}},
	}
	for i, field := range []string{"guid", "size", "allocated", "free", "pool-state"} {
		cases = append(cases, struct {
			name, want string
			mutate     func(*dashboardHTMLNode)
		}{field, "cell-" + field, func(n *dashboardHTMLNode) {
			n.direct("tbody")[0].find("td", "", "")[i].content = []dashboardHTMLContent{{text: "private-observed-value"}}
		}})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			response := dashboardHTMLRequest(h, cookie, "GET", "/storage/pools", "")
			root, err := dashboardParseHTML(response.Body.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			table, err := dashboardTable(root, expected.caption)
			if err != nil {
				t.Fatal(err)
			}
			generation, err := root.one("p", "class", "generation")
			if err != nil {
				t.Fatal(err)
			}
			code, err := generation.one("code", "", "")
			if err != nil {
				t.Fatal(err)
			}
			if err := dashboardTableHTML(table, expected, code.rawText()); err != nil {
				t.Fatal("unmodified table rejected", err)
			}
			c.mutate(table)
			err = dashboardTableHTML(table, expected, code.rawText())
			if err == nil || dashboardFailureClassification(err) != "failure:table-"+c.want {
				t.Fatal("missing fixed check", c.want, err)
			}
			if strings.Contains(err.Error(), "private") {
				t.Fatal("observed data leaked")
			}
		})
	}
}
