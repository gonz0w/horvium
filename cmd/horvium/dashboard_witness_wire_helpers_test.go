// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gonz0w/horvium/internal/inventory"
)

var errDashboardWitness = errors.New("invalid original dashboard witness")

func witnessOneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

const (
	witnessFrameCap   = 64 << 10
	witnessPayloadCap = 64 << 20
	witnessRecordCap  = 100000
	witnessRawCap     = witnessPayloadCap + 4*witnessRecordCap
	witnessArchiveCap = 256 << 20
	witnessAttemptCap = 32
)

// Explicit independent grammar. This controller never calls helper framing,
// helper validation/decoding, cached queries or numeric presentation helpers.
// '?' means an optional key, '[]' a nullable array (the producer emits null for
// empty Go slices). Required scalar fields cannot be null.
var witnessShapes = map[string]map[string]string{
	"request":       {"version": "uint", "request_id": "text", "operation": "text"},
	"record":        {"version": "uint", "request_id": "text", "type": "text", "data": "any"},
	"begin":         {"started_at": "time", "finished_at": "time", "host": "host-section", "geom": "section", "pools": "section", "datasets": "section"},
	"section":       {"status": "text", "started_at": "time", "finished_at": "time", "problems?": "[]problem"},
	"host-section":  {"status": "text", "started_at": "time", "finished_at": "time", "problems?": "[]problem", "data": "host"},
	"host":          {"installed_kernel?": "text", "running_kernel?": "text", "userland?": "text", "architecture?": "text", "physical_memory_bytes?": "uint", "zfs_userland?": "text", "zfs_kernel?": "text"},
	"problem":       {"source": "text", "code": "text"},
	"class":         {"local_id": "text", "name": "text"},
	"geom":          {"local_id": "text", "class_local_id": "text", "name": "text"},
	"consumer-item": {"geom_local_id": "text", "consumer": "consumer"},
	"consumer":      {"local_id": "text", "provider_local_id?": "text"},
	"provider":      {"local_id": "text", "geom_local_id": "text", "name": "text", "class": "text", "kind": "text", "media_size_bytes": "uint", "sector_size_bytes": "uint", "identifiers": "[]identifier", "identity": "text"},
	"identifier":    {"kind": "text", "value": "text", "source": "text"},
	"pool":          {"name": "text", "guid": "text", "size_bytes": "uint", "allocated_bytes": "uint", "free_bytes": "uint", "health": "text"},
	"dataset":       {"name": "text", "guid": "text", "used_bytes": "uint", "available_bytes": "uint", "referenced_bytes": "uint"},
	"end":           {"records": "uint", "complete": "bool"},
}

func witnessJSONValue(d *json.Decoder, depth int) (any, error) {
	if depth > 32 {
		return nil, errDashboardWitness
	}
	v, e := d.Token()
	if e != nil {
		return nil, errDashboardWitness
	}
	switch x := v.(type) {
	case json.Delim:
		if x == '{' {
			out := map[string]any{}
			for d.More() {
				k, e := d.Token()
				name, ok := k.(string)
				if e != nil || !ok || !witnessText(name) {
					return nil, errDashboardWitness
				}
				if _, exists := out[name]; exists {
					return nil, errDashboardWitness
				}
				item, e := witnessJSONValue(d, depth+1)
				if e != nil {
					return nil, e
				}
				out[name] = item
			}
			end, e := d.Token()
			if e != nil || end != json.Delim('}') {
				return nil, errDashboardWitness
			}
			return out, nil
		}
		if x == '[' {
			out := []any{}
			for d.More() {
				item, e := witnessJSONValue(d, depth+1)
				if e != nil {
					return nil, e
				}
				out = append(out, item)
			}
			end, e := d.Token()
			if e != nil || end != json.Delim(']') {
				return nil, errDashboardWitness
			}
			return out, nil
		}
		return nil, errDashboardWitness
	case string:
		if !witnessText(x) {
			return nil, errDashboardWitness
		}
	case json.Number:
		if _, e := witnessUint(string(x)); e != nil {
			return nil, e
		}
	case bool, nil:
	default:
		return nil, errDashboardWitness
	}
	return v, nil
}

func witnessText(s string) bool {
	if len(s) > 4096 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func witnessUint(s string) (uint64, error) {
	n, e := strconv.ParseUint(s, 10, 64)
	if e != nil || strconv.FormatUint(n, 10) != s {
		return 0, errDashboardWitness
	}
	return n, nil
}
func witnessShape(v any, kind string) error {
	if kind == "any" {
		return nil
	}
	if strings.HasPrefix(kind, "[]") {
		if v == nil {
			return nil
		}
		list, ok := v.([]any)
		if !ok {
			return errDashboardWitness
		}
		for _, item := range list {
			if witnessShape(item, kind[2:]) != nil {
				return errDashboardWitness
			}
		}
		return nil
	}
	switch kind {
	case "text":
		if _, ok := v.(string); ok {
			return nil
		}
	case "uint":
		if _, ok := v.(json.Number); ok {
			return nil
		}
	case "bool":
		if _, ok := v.(bool); ok {
			return nil
		}
	case "time":
		if s, ok := v.(string); ok {
			if t, e := time.Parse(time.RFC3339Nano, s); e == nil && !t.IsZero() {
				return nil
			}
		}
	default:
		shape, exists := witnessShapes[kind]
		object, ok := v.(map[string]any)
		if !exists || !ok {
			return errDashboardWitness
		}
		for name, value := range object {
			child, found := shape[name]
			if !found {
				child, found = shape[name+"?"]
			}
			if !found || witnessShape(value, child) != nil {
				return errDashboardWitness
			}
		}
		for name := range shape {
			if !strings.HasSuffix(name, "?") {
				if _, found := object[name]; !found {
					return errDashboardWitness
				}
			}
		}
		return nil
	}
	return errDashboardWitness
}
func witnessJSON(b []byte, kind string, dst any) error {
	if !utf8.Valid(b) {
		return errDashboardWitness
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	v, e := witnessJSONValue(d, 0)
	if e != nil || witnessShape(v, kind) != nil {
		return errDashboardWitness
	}
	if _, e := d.Token(); e != io.EOF {
		return errDashboardWitness
	}
	if json.Unmarshal(b, dst) != nil {
		return errDashboardWitness
	}
	return nil
}

type witnessWireBudget struct{ bytes, records int }

func (b *witnessWireBudget) frame(r io.Reader) ([]byte, error) {
	var h [4]byte
	if _, e := io.ReadFull(r, h[:]); e != nil {
		return nil, errDashboardWitness
	}
	n := uint64(binary.BigEndian.Uint32(h[:]))
	if n == 0 || n > witnessFrameCap || n > uint64(witnessPayloadCap-b.bytes) || b.records >= witnessRecordCap {
		return nil, errDashboardWitness
	}
	b.bytes += int(n)
	b.records++
	p := make([]byte, int(n))
	if _, e := io.ReadFull(r, p); e != nil {
		return nil, errDashboardWitness
	}
	return p, nil
}
func witnessEOF(r io.Reader) error {
	var extra [1]byte
	n, e := r.Read(extra[:])
	if n != 0 || e != io.EOF {
		return errDashboardWitness
	}
	return nil
}
func dashboardWitnessRequest(r io.Reader) (string, error) {
	var b witnessWireBudget
	raw, e := b.frame(r)
	if e != nil {
		return "", e
	}
	var v struct {
		Version   int    `json:"version"`
		ID        string `json:"request_id"`
		Operation string `json:"operation"`
	}
	if witnessJSON(raw, "request", &v) != nil || v.Version != 1 || v.Operation != "inventory" || !dashboardGenerationPattern.MatchString(v.ID) || witnessEOF(r) != nil {
		return "", errDashboardWitness
	}
	if _, e := witnessUint(strings.Split(v.ID, "-")[1]); e != nil {
		return "", e
	}
	return v.ID, nil
}

type witnessSection struct {
	Status     inventory.Status    `json:"status"`
	StartedAt  time.Time           `json:"started_at"`
	FinishedAt time.Time           `json:"finished_at"`
	Problems   []inventory.Problem `json:"problems,omitempty"`
}

func witnessSectionValue[T any](v witnessSection) inventory.Section[T] {
	return inventory.Section[T]{Status: v.Status, StartedAt: v.StartedAt, FinishedAt: v.FinishedAt, Problems: v.Problems}
}
func dashboardDecodeWitness(r io.Reader, id string) (inventory.Snapshot, error) {
	var s inventory.Snapshot
	if !dashboardGenerationPattern.MatchString(id) {
		return s, errDashboardWitness
	}
	var budget witnessWireBudget
	stage := 0
	currentGeom := ""
	for {
		b, e := budget.frame(r)
		if e != nil {
			return inventory.Snapshot{}, e
		}
		var rec struct {
			Version int             `json:"version"`
			ID      string          `json:"request_id"`
			Type    string          `json:"type"`
			Data    json.RawMessage `json:"data"`
		}
		if witnessJSON(b, "record", &rec) != nil || rec.Version != 1 || rec.ID != id {
			return inventory.Snapshot{}, errDashboardWitness
		}
		order := map[string]int{"begin": 0, "class": 1, "geom": 2, "consumer": 2, "provider": 3, "pool": 4, "dataset": 5, "end": 6}
		n, known := order[rec.Type]
		if !known || n < stage || (budget.records == 1) != (rec.Type == "begin") {
			return inventory.Snapshot{}, errDashboardWitness
		}
		stage = n
		kind := rec.Type
		if kind == "consumer" {
			kind = "consumer-item"
		}
		decode := func(v any) error { return witnessJSON(rec.Data, kind, v) }
		switch rec.Type {
		case "begin":
			var v struct {
				StartedAt  time.Time                         `json:"started_at"`
				FinishedAt time.Time                         `json:"finished_at"`
				Host       inventory.Section[inventory.Host] `json:"host"`
				GEOM       witnessSection                    `json:"geom"`
				Pools      witnessSection                    `json:"pools"`
				Datasets   witnessSection                    `json:"datasets"`
			}
			if decode(&v) != nil {
				return inventory.Snapshot{}, errDashboardWitness
			}
			s = inventory.Snapshot{StartedAt: v.StartedAt, FinishedAt: v.FinishedAt, Host: v.Host, GEOM: witnessSectionValue[inventory.GEOM](v.GEOM), Pools: witnessSectionValue[[]inventory.Pool](v.Pools), Datasets: witnessSectionValue[[]inventory.Dataset](v.Datasets)}
		case "class":
			var v inventory.Class
			if decode(&v) != nil {
				return inventory.Snapshot{}, errDashboardWitness
			}
			s.GEOM.Data.Classes = append(s.GEOM.Data.Classes, v)
		case "geom":
			var v struct {
				ID    string `json:"local_id"`
				Class string `json:"class_local_id"`
				Name  string `json:"name"`
			}
			if decode(&v) != nil {
				return inventory.Snapshot{}, errDashboardWitness
			}
			currentGeom = v.ID
			s.GEOM.Data.Geoms = append(s.GEOM.Data.Geoms, inventory.Geom{ID: v.ID, ClassID: v.Class, Name: v.Name})
		case "consumer":
			var v struct {
				Geom     string             `json:"geom_local_id"`
				Consumer inventory.Consumer `json:"consumer"`
			}
			if decode(&v) != nil || currentGeom == "" || v.Geom != currentGeom || len(s.GEOM.Data.Geoms) == 0 {
				return inventory.Snapshot{}, errDashboardWitness
			}
			i := len(s.GEOM.Data.Geoms) - 1
			s.GEOM.Data.Geoms[i].Consumers = append(s.GEOM.Data.Geoms[i].Consumers, v.Consumer)
		case "provider":
			var v inventory.Provider
			if decode(&v) != nil {
				return inventory.Snapshot{}, errDashboardWitness
			}
			s.GEOM.Data.Providers = append(s.GEOM.Data.Providers, v)
		case "pool":
			var v inventory.Pool
			if decode(&v) != nil {
				return inventory.Snapshot{}, errDashboardWitness
			}
			s.Pools.Data = append(s.Pools.Data, v)
		case "dataset":
			var v inventory.Dataset
			if decode(&v) != nil {
				return inventory.Snapshot{}, errDashboardWitness
			}
			s.Datasets.Data = append(s.Datasets.Data, v)
		case "end":
			var v struct {
				Records  int  `json:"records"`
				Complete bool `json:"complete"`
			}
			if decode(&v) != nil || v.Records != budget.records || v.Complete != s.Complete() || witnessEOF(r) != nil || witnessValidateSnapshot(s) != nil {
				return inventory.Snapshot{}, errDashboardWitness
			}
			return s, nil
		}
	}
}

func witnessValidateSnapshot(s inventory.Snapshot) error {
	if s.StartedAt.IsZero() || s.FinishedAt.Before(s.StartedAt) {
		return errDashboardWitness
	}
	meta := []witnessSection{{s.Host.Status, s.Host.StartedAt, s.Host.FinishedAt, s.Host.Problems}, {s.GEOM.Status, s.GEOM.StartedAt, s.GEOM.FinishedAt, s.GEOM.Problems}, {s.Pools.Status, s.Pools.StartedAt, s.Pools.FinishedAt, s.Pools.Problems}, {s.Datasets.Status, s.Datasets.StartedAt, s.Datasets.FinishedAt, s.Datasets.Problems}}
	for i, m := range meta {
		if m.StartedAt.IsZero() || m.FinishedAt.Before(m.StartedAt) || m.StartedAt.Before(s.StartedAt) || m.FinishedAt.After(s.FinishedAt) {
			return errDashboardWitness
		}
		if m.Status != inventory.OK && m.Status != inventory.Partial && m.Status != inventory.Unavailable || (m.Status == inventory.OK) != (len(m.Problems) == 0) || i > 0 && m.Status == inventory.Partial {
			return errDashboardWitness
		}
		seen := map[string]bool{}
		for _, p := range m.Problems {
			if seen[p.Source] || !witnessOneOf(p.Source, "freebsd-version", "architecture", "physical-memory", "zfs-version", "geom", "pools", "datasets") || !witnessOneOf(p.Code, "limit", "malformed", "timeout", "cancelled", "permission", "unavailable", "failed") {
				return errDashboardWitness
			}
			seen[p.Source] = true
		}
	}
	h := s.Host.Data
	if s.Host.Status == inventory.OK && (h.InstalledKernel == "" || h.RunningKernel == "" || h.Userland == "" || h.Architecture == "" || h.PhysicalMemory == nil || *h.PhysicalMemory == 0 || h.ZFSUserland == "" || h.ZFSKernel == "") {
		return errDashboardWitness
	}
	if s.Host.Status == inventory.Unavailable && h != (inventory.Host{}) {
		return errDashboardWitness
	}
	if s.GEOM.Status == inventory.Unavailable && len(s.GEOM.Data.Classes)+len(s.GEOM.Data.Geoms)+len(s.GEOM.Data.Providers) > 0 || s.Pools.Status == inventory.Unavailable && len(s.Pools.Data) > 0 || s.Datasets.Status == inventory.Unavailable && len(s.Datasets.Data) > 0 {
		return errDashboardWitness
	}
	ids := map[string]bool{}
	unique := func(id string) bool {
		if id == "" || ids[id] {
			return false
		}
		ids[id] = true
		return true
	}
	classes := map[string]string{}
	geoms := map[string]string{}
	providers := map[string]bool{}
	for _, v := range s.GEOM.Data.Classes {
		if v.Name == "" || !unique(v.ID) {
			return errDashboardWitness
		}
		classes[v.ID] = v.Name
	}
	for _, v := range s.GEOM.Data.Geoms {
		if v.Name == "" || !unique(v.ID) || classes[v.ClassID] == "" {
			return errDashboardWitness
		}
		geoms[v.ID] = classes[v.ClassID]
		for _, c := range v.Consumers {
			if !unique(c.ID) {
				return errDashboardWitness
			}
		}
	}
	for _, v := range s.GEOM.Data.Providers {
		if v.Name == "" || !unique(v.ID) || geoms[v.GeomID] != v.Class || v.Class == "" || v.SectorSize == 0 {
			return errDashboardWitness
		}
		providers[v.ID] = true
		kind := inventory.Logical
		switch v.Class {
		case "DISK":
			kind = inventory.DiskCandidate
		case "PART":
			kind = inventory.Partition
		case "MD":
			kind = inventory.Memory
		}
		if v.Kind != kind {
			return errDashboardWitness
		}
		if kind != inventory.DiskCandidate {
			if v.Identity != inventory.IdentityNotApplicable {
				return errDashboardWitness
			}
		} else if v.Identity != inventory.IdentityUnique && v.Identity != inventory.IdentityMissing && v.Identity != inventory.IdentityAmbiguous || (len(v.Identifiers) == 0) != (v.Identity == inventory.IdentityMissing) {
			return errDashboardWitness
		}
		for _, i := range v.Identifiers {
			key := "ident"
			if i.Kind == "lunid" {
				key = "lunid"
			}
			if i.Kind != "serial" && i.Kind != "lunid" || i.Value == "" || i.Source != "geom.config."+key && i.Source != "provider.config."+key {
				return errDashboardWitness
			}
		}
	}
	for _, v := range s.GEOM.Data.Geoms {
		for _, c := range v.Consumers {
			if c.ProviderID != "" && !providers[c.ProviderID] {
				return errDashboardWitness
			}
		}
	}
	for _, isPool := range []bool{true, false} {
		seenName, seenGUID := map[string]bool{}, map[string]bool{}
		check := func(name, guid string) bool {
			n, e := witnessUint(guid)
			if e != nil || n == 0 || name == "" || seenName[name] || seenGUID[guid] {
				return false
			}
			seenName[name], seenGUID[guid] = true, true
			return true
		}
		if isPool {
			for _, v := range s.Pools.Data {
				if !check(v.Name, v.GUID) || v.Allocated > v.Size || v.Free > v.Size || !witnessOneOf(v.Health, "ONLINE", "DEGRADED", "FAULTED", "OFFLINE", "UNAVAIL", "REMOVED", "SUSPENDED") {
					return errDashboardWitness
				}
			}
		} else {
			for _, v := range s.Datasets.Data {
				if !check(v.Name, v.GUID) {
					return errDashboardWitness
				}
			}
		}
	}
	return nil
}
