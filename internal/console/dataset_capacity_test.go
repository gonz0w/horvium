// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

func TestDatasetCapacityExactAccounting(t *testing.T) {
	s := fixtureSnapshot()
	s.Datasets.Data = []inventory.Dataset{
		{Name: "a-parent", GUID: "18446744073709551615", Used: ^uint64(0), Available: 7, Referenced: 9},
		{Name: "a-parent/child", GUID: "9007199254740993", Used: ^uint64(0) / 2, Available: ^uint64(0), Referenced: 9007199254740993},
		{Name: `b-<img src=x onerror=alert(1)>`, GUID: "9007199254740991", Used: 9007199254740992, Available: 9007199254740991, Referenced: 9007199254740992},
	}
	h, m, _ := fixtureWithSource(t, capacitySource{s})
	c, _ := login(t, h)
	path := "/storage/datasets?generation=" + m.State().CurrentID
	body := capacityPage(t, h, c, path, false)
	section := capacitySection(t, body)
	if got := strings.Join(capacityWidths(t, section), ","); got != "1000,500,0" {
		t.Fatalf("page-relative Used widths: %s", got)
	}
	for _, text := range []string{"Dataset Used comparisons", "largest Used on this page", "18446744073709551615 B", "overlap", "quota headroom", "Approximate", "100.0%", "50.0%"} {
		if !strings.Contains(section, text) {
			t.Fatalf("missing honest scale/accounting %q", text)
		}
	}
	for _, text := range []string{"18446744073709551615 B", "9223372036854775807 B", "9007199254740993 B", "9007199254740992 B", "9007199254740991 B", "7 B", "9 B", "Referenced", "Available"} {
		if !strings.Contains(body, text) {
			t.Fatalf("lost exact independent value %q", text)
		}
	}
	if strings.Contains(body, "<img src=x") || !strings.Contains(section, "&lt;img") || strings.Contains(section, "style=") {
		t.Fatal("unsafe observed label or inline style")
	}
	if capacitySection(t, capacityPage(t, h, c, path, true)) != section {
		t.Fatal("full/fragment mismatch")
	}
	r := req("GET", "/api/datasets?generation="+m.State().CurrentID, "")
	r.AddCookie(c)
	w := serve(h, r)
	var p observation.Page[inventory.Dataset]
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &p) != nil || len(p.Rows) != 3 || p.Rows[0].Used != ^uint64(0) || p.Rows[1].GUID != "9007199254740993" || p.Rows[1].Referenced != 9007199254740993 {
		t.Fatal("exact JSON contract changed")
	}
}
func TestDatasetCapacityPageScaleAndBound(t *testing.T) {
	s := fixtureSnapshot()
	s.Datasets.Data = nil
	for i := 0; i < 40; i++ {
		used := uint64(i + 1)
		if i == 34 {
			used = 1000
		}
		if i == 39 {
			used = ^uint64(0)
		}
		s.Datasets.Data = append(s.Datasets.Data, inventory.Dataset{Name: fmt.Sprintf("data-%02d", i), GUID: strconv.Itoa(1000 + i), Used: used})
	}
	h, m, _ := fixtureWithSource(t, capacitySource{s})
	c, _ := login(t, h)
	id := m.State().CurrentID
	q := url.Values{"generation": {id}, "filter": {"data-"}, "sort": {"name"}, "descending": {"false"}, "offset": {"5"}, "limit": {"30"}, "selected": {"1005"}}
	body := capacityPage(t, h, c, "/storage/datasets?"+q.Encode(), false)
	section := capacitySection(t, body)
	widths := capacityWidths(t, section)
	if len(widths) != 20 || widths[0] != "6" || widths[19] != "25" || !strings.Contains(section, "largest Used on this page · 1000 B") || !strings.Contains(section, "Showing 6–25 of 40 matching datasets") || strings.Contains(section, "data-25") {
		t.Fatal("scope/scale is not the original bounded page and first 20 comparisons")
	}
	if !strings.Contains(body, "data-34") || !strings.Contains(body, `id="row-1005"`) {
		t.Fatal("table or selection truncated")
	}
	links := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(section, -1)
	if len(links) != 20 {
		t.Fatalf("links %d", len(links))
	}
	for i, l := range links {
		u, e := url.Parse(html.UnescapeString(l[1]))
		if e != nil {
			t.Fatal(e)
		}
		v := u.Query()
		if u.Path != "/storage/datasets" || v.Get("generation") != id || v.Get("filter") != "data-" || v.Get("sort") != "name" || v.Get("descending") != "false" || v.Get("offset") != "5" || v.Get("limit") != "30" || v.Get("selected") != strconv.Itoa(1005+i) {
			t.Fatal("comparison link changed original query/GUID")
		}
	}
	max := capacitySection(t, capacityPage(t, h, c, "/storage/datasets?limit=500&sort=used&descending=true", false))
	if len(capacityWidths(t, max)) != 20 || !strings.Contains(max, "18446744073709551615 B") || !strings.Contains(max, "data-39") {
		t.Fatal("maximum page or exact numeric order changed")
	}
	empty := capacitySection(t, capacityPage(t, h, c, "/storage/datasets?offset=40", false))
	if len(capacityWidths(t, empty)) != 0 || !strings.Contains(empty, "No comparisons on this page") {
		t.Fatal("off-end page invented comparison")
	}
}
func TestDatasetCapacityZeroEmptyAndFreshness(t *testing.T) {
	for _, status := range []inventory.Status{inventory.OK, inventory.Unavailable} {
		t.Run(string(status), func(t *testing.T) {
			s := fixtureSnapshot()
			s.Datasets.Status = status
			s.Datasets.Data = nil
			if status != inventory.OK {
				s.Datasets.Problems = []inventory.Problem{{Source: "datasets", Code: "permission"}}
			}
			h, _, _ := fixtureWithSource(t, capacitySource{s})
			c, _ := login(t, h)
			body := capacityPage(t, h, c, "/storage/datasets", false)
			if status == inventory.OK {
				section := capacitySection(t, body)
				if len(capacityWidths(t, section)) != 0 || !strings.Contains(section, "No comparisons on this page") {
					t.Fatal("empty invented data")
				}
			} else if strings.Contains(body, `class="capacity-section"`) || !strings.Contains(body, "Resource count unavailable") {
				t.Fatal("failed section appears complete")
			}
		})
	}
	s := fixtureSnapshot()
	s.Datasets.Data = []inventory.Dataset{{Name: "zero", GUID: "1", Available: 100}}
	old := time.Now().Add(-2 * time.Minute).UTC()
	s.StartedAt = old
	s.Datasets.StartedAt = old
	s.Datasets.FinishedAt = old
	h, _, _ := fixtureWithSource(t, capacitySource{s})
	c, _ := login(t, h)
	section := capacitySection(t, capacityPage(t, h, c, "/storage/datasets", false))
	if len(capacityWidths(t, section)) != 0 || !strings.Contains(section, "Scale unavailable: every Used value on this page is zero") || !strings.Contains(section, "Stale") || !strings.Contains(section, old.Format("2006-01-02 15:04:05 UTC")) || strings.Contains(section, "0.0%") {
		t.Fatal("zero scale/stale observation misrepresented")
	}
}
func TestDatasetCapacityGenerationAndDenials(t *testing.T) {
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
	for _, tc := range []struct {
		path   string
		cookie bool
		code   int
	}{{"/storage/datasets?generation=" + id, true, 409}, {"/storage/datasets", false, 303}, {"/storage/datasets?selected=missing", true, 400}} {
		r := req("GET", tc.path, "")
		if tc.cookie {
			r.AddCookie(c)
		}
		w := serve(h, r)
		if w.Code != tc.code || strings.Contains(w.Body.String(), `class="capacity-section"`) {
			t.Fatalf("denied/expired query displayed comparisons: %s %d", tc.path, w.Code)
		}
	}
}

func TestDatasetCapacityNonSuccessProjection(t *testing.T) {
	for _, status := range []inventory.Status{inventory.Partial, inventory.Unavailable, ""} {
		p := observation.Page[inventory.Dataset]{Freshness: observation.Freshness{Status: status}, Rows: []inventory.Dataset{{Name: "hidden", GUID: "1", Used: 100}}}
		if datasetComparisons(p, observation.Query{}) != nil {
			t.Fatal("non-success projection inferred complete data")
		}
	}
}
