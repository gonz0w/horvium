//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type witnessNativeCoverage struct {
	mu       sync.Mutex
	dir      string
	root     *os.Root
	files    map[string]os.FileInfo
	metadata map[string]string
	refs     map[string]dashboardWitnessRef
	receipt  witnessCoverageReceipt
	observer *witnessFixtureObserver
	registry *dashboardWitnessRegistry
	plan     []dashboardRequest
}

func witnessNativeCoverageContext(t *testing.T, mode string, owner *dashboardGuestOwner) witnessCoverageContext {
	t.Helper()
	var c witnessCoverageContext
	b, e := base64.StdEncoding.DecodeString(os.Getenv("HORVIUM_WITNESS_CONTEXT"))
	if e != nil || len(b) > 4096 {
		t.Fatal("source-bound witness context")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		t.Fatal("candidate context decode")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		t.Fatal("context trailing data")
	}
	c.Mode = mode
	c.ManagementSHA256 = owner.artifacts.Artifacts["management"]
	c.HelperSHA256 = owner.artifacts.Artifacts["helper"]
	c.FixtureSHA256 = owner.artifacts.Artifacts["fixture"]
	probe, e := os.ReadFile("../../.tools/witness-probe-original.json")
	if e != nil {
		t.Fatal("original probe context")
	}
	c.ProbeSHA256 = fmt.Sprintf("%x", sha256.Sum256(probe))
	if !witnessCoverageIdentity(c) {
		t.Fatal("incomplete original candidate identity")
	}
	return c
}
func newWitnessNativeCoverage(t *testing.T, c witnessCoverageContext, o *witnessFixtureObserver) *witnessNativeCoverage {
	t.Helper()
	dir := filepath.Join("../../.tools", "witness-coverage-"+c.Mode)
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("exclusive coverage directory")
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		t.Fatal("original coverage root")
	}
	t.Cleanup(func() { root.Close() })
	return &witnessNativeCoverage{dir: dir, root: root, files: map[string]os.FileInfo{}, metadata: map[string]string{}, refs: map[string]dashboardWitnessRef{}, receipt: witnessCoverageReceipt{Version: 1, Scope: "short", Context: c}, observer: o}
}
func (c *witnessNativeCoverage) write(name string, b []byte, limit int) error {
	if len(b) > limit {
		return errDashboardWitness
	}
	f, e := c.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return errDashboardWitness
	}
	n, we := f.Write(b)
	se := f.Sync()
	me := f.Chmod(0400)
	i, ie := f.Stat()
	ce := f.Close()
	if n != len(b) || we != nil || se != nil || me != nil || ie != nil || ce != nil {
		return errDashboardWitness
	}
	c.files[name] = i
	return nil
}
func (c *witnessNativeCoverage) read(name string, limit int) ([]byte, error) {
	before := c.files[name]
	path, e := c.root.Lstat(name)
	if e != nil || !witnessRawUnchanged(before, path) {
		return nil, errDashboardWitness
	}
	f, e := c.root.Open(name)
	if e != nil {
		return nil, errDashboardWitness
	}
	b, re := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	after, se := f.Stat()
	ce := f.Close()
	if re != nil || se != nil || ce != nil || len(b) > limit || !witnessRawUnchanged(before, after) {
		return nil, errDashboardWitness
	}
	return b, nil
}
func (c *witnessNativeCoverage) captureMetadata(id string, b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !dashboardGenerationPattern.MatchString(id) || len(c.metadata) >= 2*witnessAttemptCap || c.metadata[id] != "" {
		return errDashboardWitness
	}
	name := fmt.Sprintf("control-%03d.json", len(c.metadata)+1)
	if c.write(name, b, witnessFrameCap) != nil {
		return errDashboardWitness
	}
	c.metadata[id] = name
	return nil
}
func (c *witnessNativeCoverage) sample(request dashboardRequest, status int, body []byte, elapsed time.Duration) error {
	if request.Index != len(c.receipt.Samples) || request.Index >= 10 || status != 200 {
		return errDashboardWitness
	}
	root, e := dashboardParseHTML(body)
	if e != nil {
		return errDashboardWitness
	}
	main, e := root.one("main", "id", "workspace")
	if e != nil {
		return errDashboardWitness
	}
	generation, e := main.one("p", "class", "generation")
	if e != nil {
		return errDashboardWitness
	}
	code, e := generation.one("code", "", "")
	if e != nil {
		return errDashboardWitness
	}
	id := code.rawText()
	_, ref, e := c.registry.confirm(context.Background(), id)
	if e != nil {
		return witnessFailureFrom(e, "publication-original")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	name := fmt.Sprintf("initial-%03d.response", request.Index)
	if c.write(name, body, dashboardBodyLimit) != nil {
		return errDashboardWitness
	}
	meta, e := c.read(c.metadata[id], witnessFrameCap)
	if e != nil {
		return errDashboardWitness
	}
	c.refs[id] = ref
	c.receipt.Samples = append(c.receipt.Samples, witnessCoveredSample{Phase: "initial", Index: request.Index, Status: status, Duration: elapsed, BodyBytes: len(body), BodySHA256: fmt.Sprintf("%x", sha256.Sum256(body)), ID: id, RawSHA256: ref.RawSHA256, ForwardSHA256: ref.ForwardSHA256, MetadataSHA256: fmt.Sprintf("%x", sha256.Sum256(meta))})
	return nil
}
func (c *witnessNativeCoverage) finish(t *testing.T) {
	t.Helper()
	read := witnessCoverageReaders{Original: func(id string) (witnessOriginalData, error) {
		_, ref, e := c.observer.original(id)
		if e != nil {
			return witnessOriginalData{}, e
		}
		f, e := c.observer.root.Open(ref.Name)
		if e != nil {
			return witnessOriginalData{}, errDashboardWitness
		}
		b, re := io.ReadAll(io.LimitReader(f, witnessRawCap+1))
		ce := f.Close()
		if re != nil || ce != nil {
			return witnessOriginalData{}, errDashboardWitness
		}
		return witnessOriginalData{b, ref}, nil
	}, Metadata: func(id string) ([]byte, error) { return c.read(c.metadata[id], witnessFrameCap) }, Body: func(phase string, index int) ([]byte, error) {
		if phase != "initial" || index < 0 || index >= 10 {
			return nil, errDashboardWitness
		}
		return c.read(fmt.Sprintf("initial-%03d.response", index), dashboardBodyLimit)
	}}
	if e := dashboardValidateWitnessCoverage(c.receipt.Context, "short", c.receipt, c.plan, read); e != nil {
		t.Fatal("original supplemental coverage rejected", e)
	}
	if witnessWritePrivateIndex(c.dir, "receipt.json", c.receipt) != nil || witnessWritePrivateIndex(c.dir, "index.json", struct {
		Metadata  map[string]string              `json:"metadata"`
		Originals map[string]dashboardWitnessRef `json:"originals"`
	}{c.metadata, c.refs}) != nil {
		t.Fatal("exclusive coverage receipt")
	}
	t.Logf("NATIVE_WITNESS_COVERAGE mode=%s version=1 scope=short responses=%d metadata=%d independently_reopened=true resource_acceptance=false", c.receipt.Context.Mode, len(c.receipt.Samples), len(c.metadata))
}
