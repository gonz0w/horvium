// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestExecutionCensusChangeExactSuccessor(t *testing.T) {
	binding := executionJournalFixture().Binding
	owner := ownershipGolden()
	policy, e := DecodeOwnershipRecord(owner)
	if e != nil {
		t.Fatal(e)
	}
	policy.Revision++
	successor, e := EncodeOwnershipRecord(policy)
	if e != nil {
		t.Fatal(e)
	}
	job := catalogJob(1, false)
	jobBytes, _ := EncodeJobRecord(job)
	jobName := "job-" + job.ID + ".json"
	for _, name := range []string{"empty-insert", "sorted-insert", "replace", "exact-bytes"} {
		t.Run(name, func(t *testing.T) {
			if name == "sorted-insert" {
				// Insert a job before an already present ownership record, rather
				// than only exercising the append-at-end completion path.
				b, e := newExecutionCensusChange(binding, []string{"ownership-100.json", "writer.lock"}, jobName, jobBytes, defaultStateCatalogLimits)
				if e != nil || b.add("ownership-100.json", owner) != nil {
					t.Fatal("sorted insertion stream", e)
				}
				before, after, previous, e := b.finish()
				if e != nil || previous != "" || before != expectedExecutionCensus(binding, []string{"ownership-100.json"}, map[string][]byte{"ownership-100.json": owner}) || after != expectedExecutionCensus(binding, []string{jobName, "ownership-100.json"}, map[string][]byte{jobName: jobBytes, "ownership-100.json": owner}) {
					t.Fatal("successor inserted before existing record", e)
				}
				if a, z, p, e := b.finish(); !errors.Is(e, ErrDenied) || a != "" || z != "" || p != "" {
					t.Fatal("single insertion completion")
				}
				return
			}
			names := []string{"writer.lock"}
			records := map[string][]byte{}
			if name != "empty-insert" {
				names = append(names, jobName)
				records[jobName] = jobBytes
			}
			replacement := owner
			if name == "replace" || name == "exact-bytes" {
				names = append(names, "ownership-100.json")
				records["ownership-100.json"] = owner
				replacement = successor
			}
			if name == "exact-bytes" {
				replacement = append([]byte(" \n"), successor...)
			}
			b, e := newExecutionCensusChange(binding, names, "ownership-100.json", replacement, defaultStateCatalogLimits)
			if e != nil {
				t.Fatal(e)
			}
			for _, entry := range b.before.entries {
				if b.add(entry.name, records[entry.name]) != nil {
					t.Fatal("validated stream")
				}
			}
			before, after, previous, e := b.finish()
			beforeNames := []string{}
			if len(records) > 0 {
				beforeNames = append(beforeNames, jobName)
			}
			if len(records) == 2 {
				beforeNames = append(beforeNames, "ownership-100.json")
			}
			if e != nil || before != expectedExecutionCensus(binding, beforeNames, records) {
				t.Fatal("complete predecessor", e)
			}
			expectedPrevious := ""
			if old, ok := records["ownership-100.json"]; ok {
				expectedPrevious = journalContentDigestPortable(old)
			}
			records["ownership-100.json"] = replacement
			afterNames := []string{}
			if name != "empty-insert" {
				afterNames = append(afterNames, jobName)
			}
			afterNames = append(afterNames, "ownership-100.json")
			if after != expectedExecutionCensus(binding, afterNames, records) || previous != expectedPrevious || before == after {
				t.Fatal("exact successor or predecessor bytes")
			}
			if a, b, p, e := b.finish(); !errors.Is(e, ErrDenied) || a != "" || b != "" || p != "" {
				t.Fatal("single completion")
			}
		})
	}
}

func TestExecutionCensusChangeDenials(t *testing.T) {
	binding := executionJournalFixture().Binding
	owner := ownershipGolden()
	for _, name := range []string{"nil", "partial", "copy", "wrong-order", "bad-record", "unknown-name", "missing-lock", "bad-binding", "byte-limit", "count-limit", "input-copy"} {
		t.Run(name, func(t *testing.T) {
			names := []string{"ownership-100.json", "writer.lock"}
			target := "ownership-100.json"
			data := owner
			limits := defaultStateCatalogLimits
			bind := binding
			switch name {
			case "unknown-name":
				target = "foreign.json"
			case "missing-lock":
				names = []string{"ownership-100.json"}
			case "bad-binding":
				bind.Lock = ""
			case "bad-record":
				data = []byte("{}")
			case "byte-limit":
				limits.bytes = uint64(len(owner) - 1)
			case "count-limit":
				limits.ownership = 0
			}
			b, e := newExecutionCensusChange(bind, names, target, data, limits)
			if name == "unknown-name" || name == "missing-lock" || name == "bad-binding" || name == "bad-record" || name == "byte-limit" || name == "count-limit" {
				if e == nil || b != nil {
					t.Fatal("invalid constructor accepted")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			switch name {
			case "nil":
				b = nil
			case "copy":
				copied := *b
				b = &copied
			case "wrong-order":
				if b.add("job-"+strings.Repeat("f", 64)+".json", owner) == nil {
					t.Fatal("wrong stream accepted")
				}
			case "input-copy":
				owned := bytes.Clone(owner)
				b, e = newExecutionCensusChange(bind, names, target, owned, limits)
				if e != nil {
					t.Fatal(e)
				}
				owned[0] = 'x'
				if b.add(target, owner) != nil {
					t.Fatal("original bytes")
				}
				a, z, p, e := b.finish()
				if e != nil || a != z || p != journalContentDigestPortable(owner) {
					t.Fatal("caller bytes alias")
				}
				return
			}
			a, z, p, e := b.finish()
			if e == nil || a != "" || z != "" || p != "" {
				t.Fatal("partial or foreign completion")
			}
		})
	}
}

func journalContentDigestPortable(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func FuzzExecutionCensusChange(f *testing.F) {
	f.Add(ownershipGolden())
	f.Add([]byte("{}"))
	f.Fuzz(func(t *testing.T, data []byte) {
		binding := executionJournalFixture().Binding
		b, e := newExecutionCensusChange(binding, []string{"ownership-100.json", "writer.lock"}, "ownership-100.json", data, defaultStateCatalogLimits)
		if e != nil {
			if b != nil {
				t.Fatal("partial constructor")
			}
			return
		}
		old := ownershipGolden()
		if b.add("ownership-100.json", old) != nil {
			t.Fatal("complete old stream")
		}
		before, after, previous, e := b.finish()
		if e != nil {
			t.Fatal(e)
		}
		if before != expectedExecutionCensus(binding, []string{"ownership-100.json"}, map[string][]byte{"ownership-100.json": old}) || after != expectedExecutionCensus(binding, []string{"ownership-100.json"}, map[string][]byte{"ownership-100.json": data}) || previous != journalContentDigestPortable(old) {
			t.Fatal("non-exact digest")
		}
	})
}
