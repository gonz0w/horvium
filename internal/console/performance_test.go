// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
	"net/http"
	"net/http/httptest"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type generatedSource struct{ n int }

func (s generatedSource) Collect(context.Context, string) (inventory.Snapshot, error) {
	snap := fixtureSnapshot()
	snap.Datasets.Data = make([]inventory.Dataset, s.n)
	for i := range snap.Datasets.Data {
		snap.Datasets.Data[i] = inventory.Dataset{Name: fmt.Sprintf("vault/dataset-%06d", i), GUID: strconv.Itoa(i + 100), Used: uint64(i), Available: 1 << 40, Referenced: uint64(i)}
	}
	snap.GEOM.Data.Classes = []inventory.Class{{ID: "1", Name: "DISK"}}
	for i := 0; i < 4; i++ {
		gid, pid := strconv.Itoa(2+i), strconv.Itoa(6+i)
		snap.GEOM.Data.Geoms = append(snap.GEOM.Data.Geoms, inventory.Geom{ID: gid, ClassID: "1", Name: fmt.Sprintf("fixture%d", i)})
		snap.GEOM.Data.Providers = append(snap.GEOM.Data.Providers, inventory.Provider{ID: pid, GeomID: gid, Name: fmt.Sprintf("fixture%d", i), Class: "DISK", Kind: inventory.DiskCandidate, MediaSize: 3e12, SectorSize: 512, Identity: inventory.IdentityMissing})
	}
	return snap, nil
}

// This diagnostic measures authenticated handler/query/render behavior, with
// ten concurrent readers and 1,000 pages per fixture. It is deliberately not
// the TDD's warm-up/idle/combined-helper RSS acceptance procedure.
func BenchmarkAuthenticatedDatasetPages(b *testing.B) {
	for _, fixture := range []struct{ count, limit int }{{100, 100}, {100, 500}, {10000, 100}, {10000, 500}} {
		count, limit := fixture.count, fixture.limit
		b.Run(fmt.Sprintf("%d/limit-%d", count, limit), func(b *testing.B) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cfg := observation.DefaultConfig()
			cfg.Interval = time.Hour
			m, e := observation.New(cfg, generatedSource{count})
			if e != nil {
				b.Fatal(e)
			}
			done := make(chan struct{})
			go func() { defer close(done); _ = m.Run(ctx) }()
			defer func() { cancel(); <-done }()
			for until := time.Now().Add(5 * time.Second); m.State().CurrentID == ""; {
				if time.Now().After(until) {
					b.Fatal("fixture did not publish")
				}
				time.Sleep(time.Millisecond)
			}
			a := admin.NewAuth(sha256.Sum256([]byte(strings.Repeat("a", 64))))
			defer a.Close()
			h, e := New(fixtureConfig("127.0.0.1:8088"), a, m)
			if e != nil {
				b.Fatal(e)
			}
			ch, e := a.Challenge("127.0.0.1")
			if e != nil {
				b.Fatal(e)
			}
			session, e := a.Login("127.0.0.1", ch.ID, ch.CSRF, strings.Repeat("a", 64))
			if e != nil {
				b.Fatal(e)
			}
			cookie := &http.Cookie{Name: admin.SessionCookie, Value: session.ID}
			durations := make([]time.Duration, 1000)
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				var wg sync.WaitGroup
				for worker := 0; worker < 10; worker++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for i := worker * 100; i < (worker+1)*100; i++ {
							r := req("GET", fmt.Sprintf("/storage/datasets?offset=%d&limit=%d", (i*limit)%count, limit), "")
							r.AddCookie(cookie)
							w := httptest.NewRecorder()
							start := time.Now()
							h.ServeHTTP(w, r)
							durations[i] = time.Since(start)
							if w.Code != 200 {
								b.Errorf("page rejected: %d", w.Code)
							}
						}
					}()
				}
				wg.Wait()
			}
			b.StopTimer()
			slices.Sort(durations)
			b.ReportMetric(float64(durations[499].Microseconds())/1000, "p50-ms/page")
			b.ReportMetric(float64(durations[949].Microseconds())/1000, "p95-ms/page")
			b.ReportMetric(float64(durations[989].Microseconds())/1000, "p99-ms/page")
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			b.ReportMetric(float64(memory.HeapInuse), "end-heap-bytes")
			if rss := peakRSS(); rss != 0 {
				b.ReportMetric(float64(rss), "process-peak-rss-bytes")
			}
		})
	}
}
