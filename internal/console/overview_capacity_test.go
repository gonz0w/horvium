// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"context"
	"errors"

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

func TestOverviewCapacityBoundAndExactLinks(t *testing.T) {
	s := fixtureSnapshot()
	s.Pools.Data = nil
	for i := 0; i < 125; i++ {
		s.Pools.Data = append(s.Pools.Data, inventory.Pool{Name: fmt.Sprintf("pool-%03d", i), GUID: strconv.Itoa(1000 + i), Size: 1000, Allocated: 999, Free: 700, Health: "ONLINE"})
	}
	s.Pools.Data[0] = inventory.Pool{Name: `pool-000-<img src=x onerror=alert(1)>`, GUID: "18446744073709551615", Size: ^uint64(0), Allocated: 9007199254740993, Free: ^uint64(0), Health: "ONLINE"}
	h, m, _ := fixtureWithSource(t, capacitySource{s})
	c, _ := login(t, h)
	id := m.State().CurrentID
	body := capacityPage(t, h, c, "/?generation="+id, false)
	section := capacitySection(t, body)
	if len(capacityWidths(t, section)) != 40 || !strings.Contains(section, "First 20 pools by name") || !strings.Contains(section, "125 observed pools; more available") || strings.Contains(section, "pool-020") {
		t.Fatal("Overview coverage/bound differs from adopted first-by-name scope")
	}
	for _, text := range []string{"18446744073709551615 B", "9007199254740993 B", "999 B", "700 B", "99.9%", "70.0%", "Pool capacity in this observation", "GUID", "Allocated", "Free"} {
		if !strings.Contains(section, text) {
			t.Fatalf("missing exact fallback/accounting %q", text)
		}
	}
	if strings.Contains(body, "<img src=x") || !strings.Contains(section, "&lt;img") || strings.Contains(section, "style=") {
		t.Fatal("observed markup or inline style leaked")
	}
	if capacitySection(t, capacityPage(t, h, c, "/?generation="+id, true)) != section {
		t.Fatal("Overview fragment/full differs")
	}
	links := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(section, -1)
	if len(links) != 40 {
		t.Fatalf("comparison and exact-table links %d", len(links))
	}
	for i, l := range links {
		u, e := url.Parse(html.UnescapeString(l[1]))
		if e != nil {
			t.Fatal(e)
		}
		q := u.Query()
		guid := strconv.Itoa(1000 + i%20)
		if i%20 == 0 {
			guid = "18446744073709551615"
		}
		if u.Path != "/storage/pools" || q.Get("generation") != id || q.Get("sort") != "name" || q.Get("descending") != "false" || q.Get("filter") != "" || q.Get("offset") != "0" || q.Get("limit") != "100" || q.Get("selected") != guid {
			t.Fatal("Overview link changed bounded page or GUID")
		}
	}
	first := html.UnescapeString(links[0][1])
	pool := capacityPage(t, h, c, first, false)
	if !strings.Contains(pool, `id="row-18446744073709551615"`) || !strings.Contains(pool, `aria-current="true"`) {
		t.Fatal("original resource not selectable on linked page")
	}
}
func TestOverviewCapacityEmptyUnavailableAndStale(t *testing.T) {
	for _, status := range []inventory.Status{inventory.OK, inventory.Unavailable} {
		t.Run(string(status), func(t *testing.T) {
			s := fixtureSnapshot()
			s.Pools.Status = status
			s.Pools.Data = nil
			if status != inventory.OK {
				s.Pools.Problems = []inventory.Problem{{Source: "pools", Code: "permission"}}
			}
			h, _, _ := fixtureWithSource(t, capacitySource{s})
			c, _ := login(t, h)
			body := capacityPage(t, h, c, "/", false)
			if status == inventory.OK {
				section := capacitySection(t, body)
				if len(capacityWidths(t, section)) != 0 || !strings.Contains(section, "0 observed pools") || !strings.Contains(section, "No pool comparisons in this observation") {
					t.Fatal("empty invented pools")
				}
			} else if strings.Contains(body, `class="capacity-section"`) || strings.Contains(body, "0 observed pools") || !strings.Contains(body, "Unavailable") {
				t.Fatal("unavailable appears complete")
			}
		})
	}
	s := fixtureSnapshot()
	old := time.Now().Add(-2 * time.Minute).UTC()
	s.StartedAt = old
	s.Pools.StartedAt = old
	s.Pools.FinishedAt = old
	h, _, _ := fixtureWithSource(t, capacitySource{s})
	c, _ := login(t, h)
	section := capacitySection(t, capacityPage(t, h, c, "/", false))
	if !strings.Contains(section, "Stale") || !strings.Contains(section, old.Format("2006-01-02 15:04:05 UTC")) || !strings.Contains(section, "First 1 pools by name") || strings.Contains(section, "more available") {
		t.Fatal("Overview pool freshness/coverage follows another section")
	}
}
func TestOverviewCapacityDenials(t *testing.T) {
	h, _, _ := fixture(t)
	c, _ := login(t, h)
	for _, tc := range []struct {
		path   string
		cookie bool
		code   int
	}{{"/", false, 303}, {"/?generation=expired", true, 409}, {"/?sort=name", true, 400}, {"/?selected=1", true, 400}, {"/?limit=500", true, 400}} {
		r := req("GET", tc.path, "")
		if tc.cookie {
			r.AddCookie(c)
		}
		w := serve(h, r)
		if w.Code != tc.code || strings.Contains(w.Body.String(), `class="capacity-section"`) {
			t.Fatalf("denied query exposes comparisons %s: %d", tc.path, w.Code)
		}
	}
}

type interleavedOverviewReader struct {
	manager                  *observation.Manager
	after                    func()
	overviewCalls, poolCalls int
	query                    observation.Query
}

func (r *interleavedOverviewReader) Overview(ctx context.Context, id string) (observation.Overview, error) {
	r.overviewCalls++
	o, e := r.manager.Overview(ctx, id)
	if e == nil && r.after != nil {
		r.after()
	}
	return o, e
}
func (r *interleavedOverviewReader) Pools(ctx context.Context, q observation.Query) (observation.Page[inventory.Pool], error) {
	r.poolCalls++
	r.query = q
	return r.manager.Pools(ctx, q)
}
func replaceOverviewGeneration(t *testing.T, m *observation.Manager) {
	t.Helper()
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
func TestOverviewCapacityInterleavedReads(t *testing.T) {
	for _, replacements := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(replacements), func(t *testing.T) {
			_, m, _ := fixture(t)
			id := m.State().CurrentID
			reader := &interleavedOverviewReader{manager: m, after: func() {
				for i := 0; i < replacements; i++ {
					replaceOverviewGeneration(t, m)
				}
			}}
			o, p, e := overviewPoolPage(context.Background(), reader, id)
			if reader.overviewCalls != 1 || reader.poolCalls != 1 || reader.query != (observation.Query{Generation: id, Sort: "name", Limit: 100}) {
				t.Fatal("composition re-resolved or changed bounded same-generation query")
			}
			if replacements < 2 {
				if e != nil || o.Generation != id || p.Generation != id {
					t.Fatal("retained generation changed")
				}
			} else if !errors.Is(e, observation.ErrExpired) || o.Generation != "" || p.Generation != "" || len(p.Rows) != 0 {
				t.Fatal("eviction combined observations or retried current generation")
			}
		})
	}
	t.Run("initial-expiry", func(t *testing.T) {
		_, m, _ := fixture(t)
		reader := &interleavedOverviewReader{manager: m}
		_, _, e := overviewPoolPage(context.Background(), reader, "expired")
		if !errors.Is(e, observation.ErrExpired) || reader.poolCalls != 0 {
			t.Fatal("pool read followed failed Overview")
		}
	})
	t.Run("cancel-between", func(t *testing.T) {
		_, m, _ := fixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		reader := &interleavedOverviewReader{manager: m, after: cancel}
		o, p, e := overviewPoolPage(ctx, reader, "")
		if !errors.Is(e, context.Canceled) || o.Generation != "" || p.Generation != "" {
			t.Fatal("cancelled composition exposed observation")
		}
	})
}
