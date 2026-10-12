// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

type capacitySource struct{ snapshot inventory.Snapshot }

func (s capacitySource) Collect(context.Context, string) (inventory.Snapshot, error) {
	return s.snapshot, nil
}
func capacityPage(t *testing.T, h http.Handler, c *http.Cookie, path string, fragment bool) string {
	t.Helper()
	r := req("GET", path, "")
	r.AddCookie(c)
	if fragment {
		r.Header.Set("HX-Request", "true")
	}
	w := serve(h, r)
	if w.Code != 200 {
		t.Fatalf("capacity page status %d: %s", w.Code, w.Body.String())
	}
	return w.Body.String()
}
func capacitySection(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<section class="capacity-section"`)
	if start < 0 {
		t.Fatal("pool comparisons missing")
	}
	end := strings.Index(body[start:], "</section>")
	if end < 0 {
		t.Fatal("comparison section not closed")
	}
	return body[start : start+end+len("</section>")]
}
func capacityWidths(t *testing.T, section string) []string {
	t.Helper()
	var widths []string
	for _, svg := range regexp.MustCompile(`(?s)<svg\b.*?</svg>`).FindAllString(section, -1) {
		d := xml.NewDecoder(strings.NewReader(svg))
		for {
			token, e := d.Token()
			if e == io.EOF {
				break
			}
			if e != nil {
				t.Fatal(e)
			}
			if el, ok := token.(xml.StartElement); ok {
				if el.Name.Local == "svg" {
					attrs := map[string]string{}
					for _, a := range el.Attr {
						attrs[a.Name.Local] = a.Value
					}
					if attrs["aria-hidden"] != "true" || attrs["focusable"] != "false" {
						t.Fatal("decorative plot exposed as interactive")
					}
				}
				if el.Name.Local == "rect" {
					a := map[string]string{}
					for _, v := range el.Attr {
						a[v.Name.Local] = v.Value
					}
					if a["class"] == "capacity-allocated" || a["class"] == "capacity-free" {
						if a["x"] != "0" {
							t.Fatal("measurement is stacked instead of zero-origin")
						}
						widths = append(widths, a["width"])
					}
				}
			}
		}
	}
	return widths
}
func TestPoolCapacityExactAccounting(t *testing.T) {
	s := fixtureSnapshot()
	s.Pools.Data = []inventory.Pool{
		{Name: "accounting", GUID: "18446744073709551615", Size: 1000, Allocated: 999, Free: 700, Health: "ONLINE"},
		{Name: `exact-<img src=x onerror=alert(1)>`, GUID: "9007199254740993", Size: ^uint64(0), Allocated: ^uint64(0) - 1, Free: ^uint64(0), Health: "ONLINE"},
		{Name: "precision", GUID: "9007199254740991", Size: 9007199254740993, Allocated: 9007199254740991, Free: 9007199254740992, Health: "ONLINE"},
	}
	h, m, _ := fixtureWithSource(t, capacitySource{s})
	c, _ := login(t, h)
	path := "/storage/pools?generation=" + m.State().CurrentID
	full := capacityPage(t, h, c, path, false)
	section := capacitySection(t, full)
	if got := strings.Join(capacityWidths(t, section), ","); got != "999,700,1000,1000,1000,1000" {
		t.Fatalf("independent rounded ratios: %s", got)
	}
	for _, text := range []string{"99.9%", "70.0%", "Allocated", "Free", "reported size", "Approximate", "measured separately"} {
		if !strings.Contains(section, text) {
			t.Fatalf("missing accounting explanation %q", text)
		}
	}
	for _, text := range []string{"18446744073709551615 B", "18446744073709551614 B", "9007199254740993 B", "9007199254740992 B", "9007199254740991 B"} {
		if !strings.Contains(full, text) {
			t.Fatalf("lost exact uint64 %q", text)
		}
	}
	if strings.Contains(full, "<img src=x") || !strings.Contains(full, "&lt;img") || strings.Contains(section, "style=") {
		t.Fatal("unsafe observed markup or inline style")
	}
	if capacitySection(t, capacityPage(t, h, c, path, true)) != section {
		t.Fatal("fragment/full comparisons differ")
	}
}
func TestPoolCapacityPageScopeAndBound(t *testing.T) {
	s := fixtureSnapshot()
	s.Pools.Data = nil
	for i := 0; i < 55; i++ {
		s.Pools.Data = append(s.Pools.Data, inventory.Pool{Name: fmt.Sprintf("pool-%02d", i), GUID: strconv.Itoa(1000 + i), Size: 100, Allocated: uint64(i + 1), Free: 80, Health: "ONLINE"})
	}
	h, m, _ := fixtureWithSource(t, capacitySource{s})
	c, _ := login(t, h)
	id := m.State().CurrentID
	q := url.Values{"generation": {id}, "filter": {"pool-"}, "sort": {"allocated"}, "descending": {"true"}, "offset": {"5"}, "limit": {"30"}, "selected": {"1049"}}
	body := capacityPage(t, h, c, "/storage/pools?"+q.Encode(), false)
	section := capacitySection(t, body)
	if len(capacityWidths(t, section)) != 40 || !strings.Contains(section, "Showing 6–25 of 55 matching pools") || !strings.Contains(section, "pool-49") || !strings.Contains(section, "pool-30") || strings.Contains(section, "pool-29") {
		t.Fatal("comparison coverage is not the first 20 rows of this page")
	}
	if !strings.Contains(body, "pool-20") || !strings.Contains(body, `id="row-1049"`) {
		t.Fatal("comparison truncated table or changed selection")
	}
	for _, text := range []string{id, "pool-", "allocated", "Descending"} {
		if !strings.Contains(section, text) {
			t.Fatalf("scope missing %q", text)
		}
	}
	links := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(section, -1)
	if len(links) != 20 {
		t.Fatalf("got %d drill-down links", len(links))
	}
	for i, l := range links {
		u, e := url.Parse(html.UnescapeString(l[1]))
		if e != nil {
			t.Fatal(e)
		}
		v := u.Query()
		if u.Path != "/storage/pools" || v.Get("generation") != id || v.Get("filter") != "pool-" || v.Get("sort") != "allocated" || v.Get("descending") != "true" || v.Get("offset") != "5" || v.Get("limit") != "30" || v.Get("selected") != strconv.Itoa(1049-i) {
			t.Fatal("drill-down lost original query/identity")
		}
	}
	q.Del("selected")
	r := req("GET", "/api/pools?"+q.Encode(), "")
	r.AddCookie(c)
	w := serve(h, r)
	var page observation.Page[inventory.Pool]
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || page.Generation != id || len(page.Rows) != 30 || page.Rows[0].GUID != "1049" || page.Rows[29].GUID != "1020" {
		t.Fatal("JSON read contract changed")
	}
	max := capacitySection(t, capacityPage(t, h, c, "/storage/pools?limit=500", false))
	if len(capacityWidths(t, max)) != 40 {
		t.Fatal("maximum page bypassed comparison bound")
	}
}
func TestPoolCapacityZeroEmptyAndFreshness(t *testing.T) {
	for _, status := range []inventory.Status{inventory.OK, inventory.Unavailable} {
		t.Run(string(status), func(t *testing.T) {
			s := fixtureSnapshot()
			s.Pools.Status = status
			if status != inventory.OK {
				s.Pools.Problems = []inventory.Problem{{Source: "pools", Code: "permission"}}
			}
			s.Pools.Data = nil
			h, _, _ := fixtureWithSource(t, capacitySource{s})
			c, _ := login(t, h)
			body := capacityPage(t, h, c, "/storage/pools", false)
			if status == inventory.OK {
				section := capacitySection(t, body)
				if len(capacityWidths(t, section)) != 0 || !strings.Contains(section, "No comparisons on this page") {
					t.Fatal("empty success invented data")
				}
			} else if strings.Contains(body, `class="capacity-section"`) || strings.Contains(body, "0 matching pools") || !strings.Contains(body, "Resource count unavailable") {
				t.Fatal("failed/partial section appears complete")
			}
		})
	}
	s := fixtureSnapshot()
	s.Pools.Data = []inventory.Pool{{Name: "empty", GUID: "1", Health: "ONLINE"}}
	old := time.Now().Add(-2 * time.Minute).UTC()
	s.StartedAt = old
	s.Pools.StartedAt = old
	s.Pools.FinishedAt = old
	h, _, _ := fixtureWithSource(t, capacitySource{s})
	c, _ := login(t, h)
	section := capacitySection(t, capacityPage(t, h, c, "/storage/pools", false))
	if len(capacityWidths(t, section)) != 0 || !strings.Contains(section, "Ratio unavailable: reported size is zero") || !strings.Contains(section, "Stale") || !strings.Contains(section, old.Format("2006-01-02 15:04:05 UTC")) || strings.Contains(section, "0.0%") {
		t.Fatal("zero/stale pool invented ratio or fresh time")
	}
}
func TestPoolCapacityGenerationAndDenials(t *testing.T) {
	h, m, _ := fixture(t)
	c, _ := login(t, h)
	id := m.State().CurrentID
	for i := 0; i < 2; i++ {
		prior := m.State().CurrentID
		deadline := time.Now().Add(time.Second)
		for m.State().Collecting {
			if time.Now().After(deadline) {
				t.Fatal("collection stuck")
			}
			time.Sleep(time.Millisecond)
		}
		if e := m.Refresh(); e != nil {
			t.Fatal(e)
		}
		for m.State().CurrentID == prior {
			if time.Now().After(deadline) {
				t.Fatal("replacement missing")
			}
			time.Sleep(time.Millisecond)
		}
	}
	r := req("GET", "/storage/pools?generation="+id, "")
	r.AddCookie(c)
	w := serve(h, r)
	if w.Code != 409 || strings.Contains(w.Body.String(), `class="capacity-section"`) || !strings.Contains(w.Body.String(), "Observation expired") {
		t.Fatal("expired observation silently changed comparison")
	}
	for _, path := range []string{"/storage/pools", "/storage/pools?selected=missing"} {
		r := req("GET", path, "")
		if path == "/storage/pools" {
			w = serve(h, r)
			if w.Code != 303 || strings.Contains(w.Body.String(), "Pool capacity comparisons") {
				t.Fatal("anonymous comparison leaked")
			}
		} else {
			r.AddCookie(c)
			w = serve(h, r)
			if w.Code != 400 || strings.Contains(w.Body.String(), `class="capacity-section"`) {
				t.Fatal("invalid selection returned comparisons")
			}
		}
	}
}

func TestPoolCapacityRatioBoundaries(t *testing.T) {
	for _, tc := range []struct {
		value, size, width uint64
		percent            string
	}{
		{0, 1, 0, "0.0%"}, {1, 1, 1000, "100.0%"}, {1, 3, 333, "33.3%"}, {2, 3, 667, "66.7%"},
		{1, 2000, 1, "0.1%"}, {1, 2001, 0, "0.0%"}, {1, 1999, 1, "0.1%"},
		{^uint64(0), ^uint64(0), 1000, "100.0%"}, {^uint64(0) / 2, ^uint64(0), 500, "50.0%"},
		{0, 0, 0, "Unavailable"}, {2, 1, 0, "Unavailable"},
	} {
		t.Run(fmt.Sprintf("%d/%d", tc.value, tc.size), func(t *testing.T) {
			r := capacityRatio(tc.value, tc.size)
			if r.Width != tc.width || r.Percent != tc.percent {
				t.Fatalf("ratio %d/%d = %+v", tc.value, tc.size, r)
			}
		})
	}
	// The runtime inventory contract rejects partial pool observations. The display
	// boundary also denies one rather than giving a future partial page a chart.
	for _, status := range []inventory.Status{inventory.Partial, inventory.Unavailable, ""} {
		p := observation.Page[inventory.Pool]{Freshness: observation.Freshness{Status: status}, Rows: fixtureSnapshot().Pools.Data}
		if poolComparisons(p, observation.Query{}) != nil {
			t.Fatal("non-success comparison admitted")
		}
	}
}
