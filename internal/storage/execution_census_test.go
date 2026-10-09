// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func expectedExecutionCensus(binding executionJournalBinding, names []string, records map[string][]byte) string {
	// Independent framing receipt: sorted records then the empty writer lock.
	var b bytes.Buffer
	b.WriteString("horvium:execution-census:v1\n")
	wire, _ := json.Marshal(binding)
	binary.Write(&b, binary.BigEndian, uint64(len(wire)))
	b.Write(wire)
	binary.Write(&b, binary.BigEndian, uint64(len(names)+1))
	for _, name := range append(append([]string(nil), names...), "writer.lock") {
		data := records[name]
		binary.Write(&b, binary.BigEndian, uint64(len(name)))
		b.WriteString(name)
		binary.Write(&b, binary.BigEndian, uint64(len(data)))
		h := sha256.Sum256(data)
		b.Write(h[:])
	}
	h := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(h[:])
}
func TestExecutionCensusExactBytesAndBinding(t *testing.T) {
	binding := executionJournalFixture().Binding
	owner := ownershipGolden()
	job := catalogJob(1, false)
	jobBytes, _ := EncodeJobRecord(job)
	jobName := "job-" + job.ID + ".json"
	names := []string{"writer.lock", "ownership-100.json", jobName}
	records := map[string][]byte{"ownership-100.json": owner, jobName: jobBytes}
	hash := func(bind executionJournalBinding, bytesByName map[string][]byte) string {
		t.Helper()
		b, e := newExecutionCensus(bind, names, defaultStateCatalogLimits)
		if e != nil {
			t.Fatal(e)
		}
		if b.add(jobName, bytesByName[jobName]) != nil || b.add("ownership-100.json", bytesByName["ownership-100.json"]) != nil {
			t.Fatal("validated bytes")
		}
		sum, e := b.finish()
		if e != nil {
			t.Fatal(e)
		}
		return sum
	}
	first := hash(binding, records)
	if first != expectedExecutionCensus(binding, []string{jobName, "ownership-100.json"}, records) {
		t.Fatal("domain/length/content/lock framing")
	}
	if hash(binding, records) != first || !bytes.Equal(owner, ownershipGolden()) {
		t.Fatal("repeat observation changed bytes")
	}
	changed := map[string][]byte{jobName: jobBytes, "ownership-100.json": append(append([]byte(nil), owner...), byte(' '))}
	if hash(binding, changed) == first {
		t.Fatal("exact record bytes discarded by canonicalization")
	}
	for _, name := range []string{"installation", "directory", "lock", "ledger-backing", "journal-backing"} {
		t.Run(name, func(t *testing.T) {
			other := binding
			switch name {
			case "installation":
				other.Installation = strings.Repeat("5", 32)
			case "directory":
				other.Directory = strings.Repeat("f", 64)
			case "lock":
				other.Lock = strings.Repeat("f", 64)
			case "ledger-backing":
				other.LedgerBacking = strings.Repeat("f", 64)
			case "journal-backing":
				other.JournalBacking = strings.Repeat("f", 64)
			}
			if hash(other, records) == first {
				t.Fatal("binding not hashed")
			}
		})
	}
	empty, e := newExecutionCensus(binding, []string{"writer.lock"}, defaultStateCatalogLimits)
	if e != nil {
		t.Fatal(e)
	}
	sum, e := empty.finish()
	if e != nil || sum == first || sum != expectedExecutionCensus(binding, nil, nil) {
		t.Fatal("complete empty namespace/lock binding", e)
	}
}
func TestExecutionCensusRefusesPartialOrUnsafeInputs(t *testing.T) {
	binding := executionJournalFixture().Binding
	for name, names := range map[string][]string{"missing-lock": nil, "duplicate": {"writer.lock", "writer.lock"}, "unknown": {"writer.lock", "foreign"}, "orphan": {"writer.lock", ".job-" + strings.Repeat("a", 32) + ".tmp"}, "alias": {"writer.lock", "../ownership-100.json"}} {
		t.Run(name, func(t *testing.T) {
			if b, e := newExecutionCensus(binding, names, defaultStateCatalogLimits); e == nil || b != nil {
				t.Fatal("unsafe namespace accepted")
			}
		})
	}
	newOne := func() *executionCensus {
		t.Helper()
		b, e := newExecutionCensus(binding, []string{"writer.lock", "ownership-100.json"}, defaultStateCatalogLimits)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	assertRetired := func(b *executionCensus) {
		t.Helper()
		if digest, e := b.finish(); e == nil || digest != "" {
			t.Fatal("partial/error digest returned")
		}
		if b.add("ownership-100.json", ownershipGolden()) == nil {
			t.Fatal("failed consumer resumed")
		}
	}
	var nilBuilder *executionCensus
	if nilBuilder.add("ownership-100.json", ownershipGolden()) == nil {
		t.Fatal("nil builder")
	}
	if sum, e := nilBuilder.finish(); e == nil || sum != "" {
		t.Fatal("nil result")
	}
	for _, name := range []string{"incomplete", "wrong-order", "corrupt", "wrong-guid", "extra", "duplicate", "copied"} {
		t.Run(name, func(t *testing.T) {
			b := newOne()
			switch name {
			case "incomplete":
				assertRetired(b)
				return
			case "wrong-order":
				if b.add("ownership-200.json", ownershipGolden()) == nil {
					t.Fatal("foreign entry accepted")
				}
			case "corrupt":
				if b.add("ownership-100.json", []byte("{}")) == nil {
					t.Fatal("corrupt record accepted")
				}
			case "wrong-guid":
				policy, err := DecodeOwnershipRecord(ownershipGolden())
				if err != nil {
					t.Fatal(err)
				}
				policy.PoolGUID = "200"
				data, err := EncodeOwnershipRecord(policy)
				if err != nil {
					t.Fatal(err)
				}
				if err = b.add("ownership-100.json", data); !errors.Is(err, ErrDrift) {
					t.Fatal("valid record selector mismatch", err)
				}
			case "extra", "duplicate":
				if b.add("ownership-100.json", ownershipGolden()) != nil {
					t.Fatal("fixture")
				}
				if b.add("ownership-100.json", ownershipGolden()) == nil {
					t.Fatal("record repeated")
				}
			case "copied":
				copied := *b
				if copied.add("ownership-100.json", ownershipGolden()) == nil {
					t.Fatal("copied consumer mutated original")
				}
				if sum, e := copied.finish(); e == nil || sum != "" {
					t.Fatal("copied result")
				}
				if b.add("ownership-100.json", ownershipGolden()) != nil {
					t.Fatal("copy retired original")
				}
				sum, e := b.finish()
				if e != nil || sum == "" {
					t.Fatal("original result", e)
				}
				return
			}
			assertRetired(b)
		})
	}
	b := newOne()
	if b.add("ownership-100.json", ownershipGolden()) != nil {
		t.Fatal("fixture")
	}
	if _, e := b.finish(); e != nil {
		t.Fatal(e)
	}
	assertRetired(b)
}
func TestExecutionCensusLimitsAndJournalConsumer(t *testing.T) {
	j := executionJournalFixture()
	owner := ownershipGolden()
	limits := stateCatalogLimits{1, 1, 1, uint64(len(owner))}
	b, e := newExecutionCensus(j.Binding, []string{"writer.lock", "ownership-100.json"}, limits)
	if e != nil || b.add("ownership-100.json", owner) != nil {
		t.Fatal("exact limits", e)
	}
	sum, e := b.finish()
	if e != nil {
		t.Fatal(e)
	}
	j.Census = sum
	encoded, e := encodeExecutionJournal(j)
	if e != nil {
		t.Fatal(e)
	}
	observed, e := decodeExecutionJournal(encoded)
	if e != nil || observed.Census != sum {
		t.Fatal("actual digest consumer", e)
	}
	limits.bytes--
	b, e = newExecutionCensus(j.Binding, []string{"writer.lock", "ownership-100.json"}, limits)
	if e != nil {
		t.Fatal(e)
	}
	if e = b.add("ownership-100.json", owner); !errors.Is(e, ErrLimit) {
		t.Fatal("byte cap", e)
	}
	if sum, e := b.finish(); e == nil || sum != "" {
		t.Fatal("over-limit returned partial hash")
	}
	for _, names := range [][]string{{"writer.lock", "ownership-100.json", "ownership-200.json"}, {"writer.lock", "job-" + fmt.Sprintf("%064x", 1) + ".json", "job-" + fmt.Sprintf("%064x", 2) + ".json", "job-" + fmt.Sprintf("%064x", 3) + ".json"}} {
		if b, e := newExecutionCensus(j.Binding, names, limits); !errors.Is(e, ErrLimit) || b != nil {
			t.Fatal("count limits", e)
		}
	}
	for _, mode := range []string{"nonterminal-cap", "terminal-cap", "duplicate-submission"} {
		t.Run(mode, func(t *testing.T) {
			first, second := catalogJob(1, mode == "terminal-cap"), catalogJob(2, mode == "terminal-cap")
			want := ErrLimit
			if mode == "duplicate-submission" {
				second.Intent.SubmissionID = first.Intent.SubmissionID
				want = ErrDrift
			}
			one, e := EncodeJobRecord(first)
			if e != nil {
				t.Fatal(e)
			}
			two, e := EncodeJobRecord(second)
			if e != nil {
				t.Fatal(e)
			}
			limits := stateCatalogLimits{1, 1, 1, uint64(len(one) + len(two))}
			oneName, twoName := "job-"+first.ID+".json", "job-"+second.ID+".json"
			b, e := newExecutionCensus(j.Binding, []string{twoName, "writer.lock", oneName}, limits)
			if e != nil || b.add(oneName, one) != nil {
				t.Fatal("first complete record", e)
			}
			if e = b.add(twoName, two); !errors.Is(e, want) {
				t.Fatal("bounded catalog refusal", e)
			}
			if sum, e := b.finish(); e == nil || sum != "" {
				t.Fatal("invalid catalog yielded digest")
			}
		})
	}
	bad := j.Binding
	bad.Lock = ""
	if b, e := newExecutionCensus(bad, []string{"writer.lock"}, limits); e == nil || b != nil {
		t.Fatal("missing lock binding")
	}
}
func FuzzExecutionCensus(f *testing.F) {
	f.Add("ownership-100.json", ownershipGolden())
	f.Add("ownership-100.json", []byte("{}"))
	f.Add("../writer.lock", []byte{})
	f.Fuzz(func(t *testing.T, name string, data []byte) {
		if len(name) > maxFieldBytes || len(data) > maxStateRecordBytes+1 {
			t.Skip()
		}
		original := append([]byte(nil), data...)
		binding := executionJournalFixture().Binding
		b, e := newExecutionCensus(binding, []string{"writer.lock", name}, defaultStateCatalogLimits)
		if e != nil {
			if b != nil {
				t.Fatal("partial builder")
			}
			return
		}
		if b.add(name, data) != nil {
			if sum, e := b.finish(); e == nil || sum != "" {
				t.Fatal("failed bytes granted hash")
			}
			return
		}
		sum, e := b.finish()
		if e != nil || !validJobDigest(sum) || !bytes.Equal(data, original) {
			t.Fatal("invalid hash/changed bytes", e)
		}
		if sum != expectedExecutionCensus(binding, []string{name}, map[string][]byte{name: data}) {
			t.Fatal("framing diverged")
		}
	})
}
