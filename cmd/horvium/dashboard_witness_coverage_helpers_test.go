// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gonz0w/horvium/internal/inventory"
	"io"
	"regexp"
	"time"
)

type witnessCoverageContext struct{ Source, Base, Title, Release, Mode, ArchiveSHA256, CompilerSHA256, ManagementSHA256, HelperSHA256, FixtureSHA256, ProbeSHA256 string }
type witnessCoveredSample struct {
	Phase          string        `json:"phase"`
	Index          int           `json:"index"`
	Status         int           `json:"status"`
	Duration       time.Duration `json:"duration_ns"`
	BodyBytes      int           `json:"body_bytes"`
	BodySHA256     string        `json:"body_sha256"`
	ID             string        `json:"generation"`
	RawSHA256      string        `json:"raw_sha256"`
	ForwardSHA256  string        `json:"forward_sha256"`
	MetadataSHA256 string        `json:"metadata_sha256"`
}
type witnessCoverageReceipt struct {
	Version int                    `json:"version"`
	Scope   string                 `json:"scope"`
	Context witnessCoverageContext `json:"context"`
	Samples []witnessCoveredSample `json:"samples"`
}
type witnessOriginalData struct {
	Raw []byte
	Ref dashboardWitnessRef
}
type witnessCoverageReaders struct {
	Original func(string) (witnessOriginalData, error)
	Metadata func(string) ([]byte, error)
	Body     func(string, int) ([]byte, error)
}

func dashboardValidateWitnessCoverage(want witnessCoverageContext, scope string, receipt witnessCoverageReceipt, plan []dashboardRequest, read witnessCoverageReaders) error {
	if receipt.Version != 1 || receipt.Scope != scope || receipt.Context != want || scope != "short" && scope != "resource" || !witnessCoverageIdentity(want) || len(plan) != 1000 || read.Original == nil || read.Metadata == nil || read.Body == nil {
		return errDashboardWitness
	}
	count := 10
	phases := []string{"initial"}
	if scope == "resource" {
		count = 1000
		phases = append(phases, "replacement")
	}
	if len(receipt.Samples) != count*len(phases) {
		return errDashboardWitness
	}
	seen := map[string]bool{}
	predicates := map[string]dashboardPredicate{}
	hash := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	for _, row := range receipt.Samples {
		allowed := row.Phase == "initial" || scope == "resource" && row.Phase == "replacement"
		key := fmt.Sprintf("%s/%d", row.Phase, row.Index)
		if !allowed || row.Index < 0 || row.Index >= count || seen[key] || row.Duration <= 0 || row.BodyBytes <= 0 || row.BodyBytes > dashboardBodyLimit {
			return errDashboardWitness
		}
		seen[key] = true
		body, e := read.Body(row.Phase, row.Index)
		if e != nil || len(body) != row.BodyBytes || hash(body) != row.BodySHA256 {
			return errDashboardWitness
		}
		request := plan[row.Index]
		if row.Status == 409 && row.Phase == "replacement" && request.Case == "overview" {
			if row.ID != "" || row.RawSHA256 != "" || row.ForwardSHA256 != "" || row.MetadataSHA256 != "" {
				return errDashboardWitness
			}
			// Reuse only the independent safe-expiry shape checker, never publication.
			root, e := dashboardParseHTML(body)
			if e != nil {
				return errDashboardWitness
			}
			workspace, e := root.one("main", "id", "workspace")
			if e != nil {
				return errDashboardWitness
			}
			heading, e := workspace.one("h1", "", "")
			if e != nil || heading.text() != "Overview" || len(workspace.find("script", "", "")) != 0 || len(workspace.find("aside", "class", "detail")) != 0 || dashboardExpiryHTML(workspace) != nil {
				return errDashboardWitness
			}
			continue
		}
		if row.Status != 200 || !dashboardGenerationPattern.MatchString(row.ID) {
			return errDashboardWitness
		}
		original, e := read.Original(row.ID)
		if e != nil {
			return errDashboardWitness
		}
		ref := original.Ref
		if ref.ID != row.ID || ref.Mode != want.Mode || ref.Management == "" || ref.Helper == "" || len(ref.Management) > 256 || len(ref.Helper) > 256 || !ref.Complete || !ref.Delivered || ref.CaptureFailure != "" || ref.RawBytes <= 0 || ref.RawBytes > witnessRawCap || int64(len(original.Raw)) != ref.RawBytes || hash(original.Raw) != ref.RawSHA256 || row.RawSHA256 != ref.RawSHA256 || ref.ForwardSHA256 != ref.RawSHA256 || row.ForwardSHA256 != ref.ForwardSHA256 {
			return errDashboardWitness
		}
		snapshot, e := dashboardDecodeWitness(bytes.NewReader(original.Raw), row.ID)
		if e != nil || !snapshot.Complete() {
			return errDashboardWitness
		}
		metadata, e := read.Metadata(row.ID)
		if e != nil || len(metadata) == 0 || len(metadata) > witnessFrameCap || hash(metadata) != row.MetadataSHA256 {
			return errDashboardWitness
		}
		published, e := witnessPublicationBytes(metadata)
		if e != nil || published != witnessPublicationOfSnapshot(snapshot, row.ID) {
			return errDashboardWitness
		}
		predicate := predicates[row.ID]
		if predicate == nil {
			predicate, e = newDashboardHTMLPredicate(snapshot, plan)
			if e != nil {
				return errDashboardWitness
			}
			predicates[row.ID] = predicate
		}
		if predicate(request, row.Status, body) != nil {
			return errDashboardWitness
		}
	}
	return nil
}
func witnessCoverageIdentity(c witnessCoverageContext) bool {
	valid := func(s string, n int) bool {
		b, e := hex.DecodeString(s)
		return e == nil && len(b) == n/2 && hex.EncodeToString(b) == s
	}
	if !valid(c.Source, 40) || !valid(c.Base, 40) || c.Source == c.Base || c.Release != "15.1" && c.Release != "14.5" || c.Mode != "native-small" && c.Mode != "projection-10000" || len(c.Title) > 512 || !regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([^\n]+\))?!?: [^\n]+$`).MatchString(c.Title) {
		return false
	}
	for _, v := range []string{c.ArchiveSHA256, c.CompilerSHA256, c.ManagementSHA256, c.HelperSHA256, c.FixtureSHA256, c.ProbeSHA256} {
		if !valid(v, 64) {
			return false
		}
	}
	return true
}
func witnessPublicationOfSnapshot(s inventory.Snapshot, id string) witnessPublication {
	return witnessPublication{ID: id, Complete: s.Complete(), Host: s.Host.Status, GEOM: s.GEOM.Status, Pools: s.Pools.Status, Datasets: s.Datasets.Status, Classes: len(s.GEOM.Data.Classes), Geoms: len(s.GEOM.Data.Geoms), Providers: len(s.GEOM.Data.Providers), PoolCount: len(s.Pools.Data), DatasetCount: len(s.Datasets.Data)}
}

// Decode required publication metadata from actual bytes. Host/accounting
// values are never numerical expectations. Missing zero counts also fail.
func witnessPublicationBytes(b []byte) (witnessPublication, error) {
	var out witnessPublication
	if len(b) == 0 || len(b) > witnessFrameCap {
		return out, errDashboardWitness
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	value, e := witnessJSONValue(d, 0)
	_, end := d.Token()
	if e != nil || end != io.EOF {
		return out, errDashboardWitness
	}
	object, ok := value.(map[string]any)
	if !ok {
		return out, errDashboardWitness
	}
	for _, key := range []string{"Generation", "Complete", "HostState", "GEOM", "Pools", "Datasets", "Classes", "Geoms", "Providers", "PoolCount", "DatasetCount"} {
		if _, ok := object[key]; !ok {
			return out, errDashboardWitness
		}
	}
	var meta struct {
		Generation                                         string
		Complete                                           bool
		HostState, GEOM, Pools, Datasets                   struct{ Status inventory.Status }
		Classes, Geoms, Providers, PoolCount, DatasetCount int
	}
	if json.Unmarshal(b, &meta) != nil {
		return out, errDashboardWitness
	}
	for _, key := range []string{"HostState", "GEOM", "Pools", "Datasets"} {
		section, ok := object[key].(map[string]any)
		if !ok {
			return out, errDashboardWitness
		}
		if _, ok := section["Status"]; !ok {
			return out, errDashboardWitness
		}
	}
	out = witnessPublication{meta.Generation, meta.Complete, meta.HostState.Status, meta.GEOM.Status, meta.Pools.Status, meta.Datasets.Status, meta.Classes, meta.Geoms, meta.Providers, meta.PoolCount, meta.DatasetCount}
	return out, nil
}
