// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func healthSource(read poolReader) *PoolHealthSource {
	return &PoolHealthSource{read: read, bytes: 4 << 20, commandTimeout: time.Second, totalTimeout: time.Second}
}

func TestPoolHealthReportsFactsWithoutEligibility(t *testing.T) {
	for _, degraded := range []bool{false, true} {
		rows := propertyCaptures()
		if degraded {
			rows[poolState] = strings.ReplaceAll(rows[poolState], "ONLINE", "DEGRADED")
			for _, p := range []poolProbe{poolTopologyGUID, poolTopologyPath} {
				rows[p] = strings.ReplaceAll(rows[p], "ONLINE", "DEGRADED")
				rows[p] = strings.ReplaceAll(rows[p], "0 0 0", "2 3 4")
				rows[p] = strings.ReplaceAll(rows[p], "No known data errors", "5 data errors, use '-v' for a list")
			}
		}
		calls := []poolProbe{}
		s := healthSource(func(_ context.Context, p poolProbe, name string, limit int) (readOutput, error) {
			if name != "tank" || limit <= 0 {
				t.Fatal("invalid selection/budget")
			}
			calls = append(calls, p)
			return testOutput(rows[p]), nil
		})
		o, err := s.Collect(context.Background(), "100", "tank")
		if err != nil || o.Pool.GUID != "100" || o.Pool.LoadGUID != "200" || len(o.Topology.Nodes) != 3 || o.Scan.Kind != "none" {
			t.Fatal("missing observation", err)
		}
		if !reflect.DeepEqual(calls, []poolProbe{poolVersions, poolState, poolTopologyGUID, poolTopologyPath, poolState, poolVersions}) {
			t.Fatal("missing identity/release bracket", calls)
		}
		if degraded && (o.Topology.Health != "DEGRADED" || o.Topology.ReadErrors != 2 || o.Topology.WriteErrors != 3 || o.Topology.ChecksumErrors != 4 || o.Topology.DataErrors != 5) {
			t.Fatal("lost unhealthy facts", o)
		}
		o.Topology.Nodes[0].GUID = "changed"
		again, err := s.Collect(context.Background(), "100", "tank")
		if err != nil || again.Topology.Nodes[0].GUID == "changed" {
			t.Fatal("aliased observation", err)
		}
	}
}

func TestPoolHealthFailuresExposeZero(t *testing.T) {
	for position := 0; position < 6; position++ {
		for _, kind := range []string{"failed", "malformed", "cancelled", "underreported", "overbudget"} {
			t.Run(string(rune('0'+position))+"/"+kind, func(t *testing.T) {
				rows := propertyCaptures()
				calls := 0
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				s := healthSource(func(_ context.Context, p poolProbe, _ string, limit int) (readOutput, error) {
					at := calls
					calls++
					if at == position {
						switch kind {
						case "failed":
							return testOutput("private capture"), ErrExit
						case "malformed":
							return testOutput("private capture"), nil
						case "cancelled":
							cancel()
						case "underreported":
							return readOutput{data: []byte("x"), consumed: 0}, nil
						case "overbudget":
							return readOutput{consumed: limit + 1}, nil
						}
					}
					return testOutput(rows[p]), nil
				})
				o, err := s.Collect(ctx, "100", "tank")
				if err == nil || !reflect.DeepEqual(o, PoolHealthObservation{}) || strings.Contains(err.Error(), "private capture") {
					t.Fatal("exposed partial capture", err, o)
				}
			})
		}
	}
}

func TestPoolHealthRejectsDriftAndUnknownFormats(t *testing.T) {
	for _, kind := range []string{"load", "health", "release", "mixed-release", "unsupported", "scan", "topology", "selection"} {
		t.Run(kind, func(t *testing.T) {
			rows := propertyCaptures()
			counts := map[poolProbe]int{}
			s := healthSource(func(_ context.Context, p poolProbe, _ string, _ int) (readOutput, error) {
				counts[p]++
				v := rows[p]
				switch kind {
				case "load":
					if p == poolState && counts[p] == 2 {
						v = strings.ReplaceAll(v, "load_guid\t200", "load_guid\t201")
					}
				case "health":
					if p == poolState {
						v = strings.ReplaceAll(v, "ONLINE", "DEGRADED")
					}
				case "release":
					if p == poolVersions && counts[p] == 2 {
						v = strings.ReplaceAll(v, "14.5", "15.1")
					}
				case "mixed-release":
					if p == poolVersions {
						v = strings.Replace(v, "14.5", "15.1", 1)
					}
				case "unsupported":
					if p == poolVersions {
						v = strings.ReplaceAll(v, "14.5", "16.0")
					}
				case "scan":
					v = strings.ReplaceAll(v, "none requested", "unknown scan")
				case "topology":
					if p == poolTopologyPath {
						v = strings.ReplaceAll(v, "0 0 0", "1 0 0")
					}
				}
				return testOutput(v), nil
			})
			guid := "100"
			if kind == "selection" {
				guid = "999"
			}
			o, err := s.Collect(context.Background(), guid, "tank")
			if err == nil || !reflect.DeepEqual(o, PoolHealthObservation{}) {
				t.Fatal("accepted mixed or unsupported facts", err, o)
			}
		})
	}
}

func TestPoolHealthSelectionAndBounds(t *testing.T) {
	calls := 0
	s := healthSource(func(context.Context, poolProbe, string, int) (readOutput, error) { calls++; return readOutput{}, nil })
	for _, selection := range [][2]string{{"0", "tank"}, {"0100", "tank"}, {"100", "tank;id"}, {"100", "tank/data"}} {
		if _, err := s.Collect(context.Background(), selection[0], selection[1]); err == nil || calls != 0 {
			t.Fatal("unsafe selection reached reader")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Collect(ctx, "100", "tank"); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("cancelled read launched", err)
	}
	for _, kind := range []string{"nil-source", "nil-context", "nil-reader", "bytes", "timeout", "total"} {
		candidate := healthSource(s.read)
		ctx := context.Background()
		switch kind {
		case "nil-source":
			candidate = nil
		case "nil-context":
			ctx = nil
		case "nil-reader":
			candidate.read = nil
		case "bytes":
			candidate.bytes = 4<<20 + 1
		case "timeout":
			candidate.commandTimeout = 0
		case "total":
			candidate.totalTimeout = 0
		}
		o, err := candidate.Collect(ctx, "100", "tank")
		if err == nil || !reflect.DeepEqual(o, PoolHealthObservation{}) || calls != 0 {
			t.Fatal("invalid setup reached reader", kind, err)
		}
	}
}

func TestPoolHealthBindsReportedScanFacts(t *testing.T) {
	scans := []string{
		"scrub repaired 0B in 00:00:03 with 7 errors on Wed Oct  7 20:00:00 2026",
		"scrub in progress since Wed Oct  7 20:00:00 2026\n1.00M / 2.00M scanned, 1.00M / 2.00M issued\n0B repaired, 50.00% done, no estimated completion time",
	}
	for _, release := range []string{"14.5", "15.1"} {
		for _, scan := range scans {
			rows := propertyCaptures()
			rows[poolVersions] = strings.ReplaceAll(rows[poolVersions], "14.5", release)
			for _, p := range []poolProbe{poolTopologyGUID, poolTopologyPath} {
				rows[p] = strings.ReplaceAll(rows[p], "none requested", strings.ReplaceAll(scan, "\n", "\n\t"))
			}
			s := healthSource(func(_ context.Context, p poolProbe, _ string, _ int) (readOutput, error) {
				return testOutput(rows[p]), nil
			})
			o, err := s.Collect(context.Background(), "100", "tank")
			if err != nil || o.Scan.Kind != "scrub" || o.Topology.Scan != scan {
				t.Fatal("scan facts not bound", err, o)
			}
			if o.Scan.State == "completed" && (!o.Scan.ErrorsReported || o.Scan.Errors != 7) {
				t.Fatal("lost scan errors")
			}
		}
	}
}

func TestPoolHealthChildTimeoutDiscardsSuccessfulRead(t *testing.T) {
	s := healthSource(func(ctx context.Context, _ poolProbe, _ string, _ int) (readOutput, error) {
		<-ctx.Done()
		return testOutput(propertyCaptures()[poolVersions]), nil
	})
	s.commandTimeout = time.Millisecond
	o, err := s.Collect(context.Background(), "100", "tank")
	if !errors.Is(err, context.DeadlineExceeded) || !reflect.DeepEqual(o, PoolHealthObservation{}) {
		t.Fatal("late capture returned facts", err)
	}
}

func TestPoolHealthUnsupportedConstructor(t *testing.T) {
	if nativeSupported {
		return
	}
	if source, err := NewPoolHealthSource(); !errors.Is(err, ErrUnsupported) || source != nil {
		t.Fatal("unsupported native source exposed", err)
	}
}

func TestPoolHealthQualifiedScanOmissionStaysUnreported(t *testing.T) {
	rows := propertyCaptures()
	for _, p := range []poolProbe{poolTopologyGUID, poolTopologyPath} {
		rows[p] = strings.ReplaceAll(rows[p], "  scan: none requested\n", "")
	}
	s := healthSource(func(_ context.Context, p poolProbe, _ string, _ int) (readOutput, error) {
		return testOutput(rows[p]), nil
	})
	o, err := s.Collect(context.Background(), "100", "tank")
	if err != nil || o.Topology.ScanReported || o.Topology.Scan != "" || o.Scan.Kind != "unreported" || o.Scan.State != "unreported" || o.Scan.ProgressReported || o.Scan.ErrorsReported {
		t.Fatal("qualified omission unavailable or fabricated scan", err, o)
	}
}
