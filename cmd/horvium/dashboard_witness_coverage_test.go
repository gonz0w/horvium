// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func witnessCoverageFixture(t *testing.T) (witnessCoverageContext, witnessCoverageReceipt, []dashboardRequest, witnessCoverageReaders) {
	t.Helper()
	snapshot, e := dashboardDecodeWitness(bytes.NewReader(witnessFrames(witnessGolden())), witnessTestID)
	if e != nil {
		t.Fatal(e)
	}
	handler, manager, cookie := dashboardHTMLFixture(t, snapshot)
	dashboardHTMLWait(t, func() bool { o, e := manager.Overview(context.Background(), ""); return e == nil && o.Complete })
	overview, e := manager.Overview(context.Background(), "")
	if e != nil {
		t.Fatal(e)
	}
	raw := bytes.ReplaceAll(witnessFrames(witnessGolden()), []byte(witnessTestID), []byte(overview.Generation))
	metadata, _ := json.Marshal(overview)
	hash := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	rawSHA := hash(raw)
	metadataSHA := hash(metadata)
	ref := dashboardWitnessRef{Name: "original-001.wire", ID: overview.Generation, Mode: "native-small", Management: "management-1", Helper: "helper-1", RawSHA256: rawSHA, RawBytes: int64(len(raw)), Complete: true, Delivered: true, ForwardSHA256: rawSHA}
	want := witnessCoverageContext{Source: strings.Repeat("1", 40), Base: strings.Repeat("2", 40), Title: "test(dashboards): native witnesses", Release: "15.1", Mode: "native-small", ArchiveSHA256: strings.Repeat("a", 64), CompilerSHA256: strings.Repeat("b", 64), ManagementSHA256: strings.Repeat("c", 64), HelperSHA256: strings.Repeat("d", 64), FixtureSHA256: strings.Repeat("e", 64), ProbeSHA256: strings.Repeat("f", 64)}
	plan, _ := dashboardPlan(1)
	record := witnessCoverageReceipt{Version: 1, Scope: "short", Context: want}
	bodies := map[int][]byte{}
	for _, request := range plan[:10] {
		body := dashboardHTMLRequest(handler, cookie, "GET", request.Path, "").Body.Bytes()
		bodies[request.Index] = bytes.Clone(body)
		record.Samples = append(record.Samples, witnessCoveredSample{Phase: "initial", Index: request.Index, Status: 200, Duration: time.Millisecond, BodyBytes: len(body), BodySHA256: hash(body), ID: overview.Generation, RawSHA256: rawSHA, ForwardSHA256: rawSHA, MetadataSHA256: metadataSHA})
	}
	readers := witnessCoverageReaders{Original: func(string) (witnessOriginalData, error) { return witnessOriginalData{bytes.Clone(raw), ref}, nil }, Metadata: func(string) ([]byte, error) { return bytes.Clone(metadata), nil }, Body: func(_ string, index int) ([]byte, error) { return bytes.Clone(bodies[index]), nil }}
	return want, record, plan, readers
}
func TestDashboardWitnessCoverageReopensOriginalsMetadataAndHTML(t *testing.T) {
	want, record, plan, read := witnessCoverageFixture(t)
	if e := dashboardValidateWitnessCoverage(want, "short", record, plan, read); e != nil {
		t.Fatal("independent source-bound complete response coverage", e)
	}
}
func TestDashboardWitnessCoverageRejectsCandidateLineageAndBodyForgery(t *testing.T) {
	for _, name := range []string{"release", "mode", "source", "probe", "legacy", "scope", "count", "duplicate", "wrong-id", "wrong-raw", "wrong-forward", "wrong-metadata", "forged-body", "private-reader-failure", "missing-publication"} {
		t.Run(name, func(t *testing.T) {
			want, row, plan, read := witnessCoverageFixture(t)
			switch name {
			case "release":
				row.Context.Release = "14.5"
			case "mode":
				row.Context.Mode = "projection-10000"
			case "source":
				row.Context.Source = strings.Repeat("3", 40)
			case "probe":
				row.Context.ProbeSHA256 = strings.Repeat("0", 64)
			case "legacy":
				row.Version = 0
			case "scope":
				row.Scope = "resource"
			case "count":
				row.Samples = row.Samples[:9]
			case "duplicate":
				row.Samples[1].Index = 0
			case "wrong-id":
				row.Samples[0].ID = witnessTestID
			case "wrong-raw":
				row.Samples[0].RawSHA256 = strings.Repeat("0", 64)
			case "wrong-forward":
				row.Samples[0].ForwardSHA256 = strings.Repeat("0", 64)
			case "wrong-metadata":
				row.Samples[0].MetadataSHA256 = strings.Repeat("0", 64)
			case "forged-body":
				original := read.Body
				read.Body = func(p string, i int) ([]byte, error) {
					b, e := original(p, i)
					return bytes.ReplaceAll(b, []byte(" B</td>"), []byte("9 B</td>")), e
				}
				for i := range row.Samples {
					b, _ := read.Body("initial", i)
					row.Samples[i].BodyBytes = len(b)
					row.Samples[i].BodySHA256 = fmt.Sprintf("%x", sha256.Sum256(b))
				}
			case "private-reader-failure":
				read.Original = func(string) (witnessOriginalData, error) {
					return witnessOriginalData{}, fmt.Errorf("PRIVATE raw path")
				}
			case "missing-publication":
				read.Metadata = func(string) ([]byte, error) { return []byte(`{"Generation":"missing"}`), nil }
			}
			e := dashboardValidateWitnessCoverage(want, "short", row, plan, read)
			if e == nil || strings.Contains(e.Error(), "PRIVATE") {
				t.Fatal("swapped/forged evidence accepted or private data leaked", e)
			}
		})
	}
}
