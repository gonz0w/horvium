// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
)

const executionJournalGoldenContent = `{"version":1,"kind":"execution-journal","generation":1,"binding":{"installation":"11111111111111111111111111111111","directory":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","lock":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","ledger_backing":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","journal_backing":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},"owner":{"id":"22222222222222222222222222222222","boot":"00000000000000000000000000000000","state":"active"},"census":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","metadata":{"pending":false,"before":"","after":"","name":"","predecessor":"","successor":""},"native":{"pending":false,"job":"","submission":"","pool_guid":"","filesystem_guid":"","intent":"","owner":"","boot":"","supervisor":""},"disarmed":false}`

func executionJournalFixture() executionJournal {
	return executionJournal{Version: 1, Kind: "execution-journal", Generation: 1,
		Binding: executionJournalBinding{strings.Repeat("1", 32), strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64), strings.Repeat("d", 64)},
		Owner:   executionJournalOwner{strings.Repeat("2", 32), strings.Repeat("00", 16), "active"}, Census: strings.Repeat("e", 64)}
}
func journalMetadataFixture(j executionJournal) executionJournalMetadata {
	return executionJournalMetadata{true, j.Census, strings.Repeat("f", 64), "ownership-100.json", "", strings.Repeat("a", 64)}
}
func journalNativeFixture(j executionJournal) executionJournalNative {
	return executionJournalNative{true, strings.Repeat("a", 64), strings.Repeat("b", 64), "100", "102", strings.Repeat("c", 64), j.Owner.ID, j.Owner.Boot, strings.Repeat("3", 32)}
}
func uncheckedExecutionJournal(content string) []byte {
	h := sha256.Sum256([]byte("horvium:execution-journal:v1\n" + content))
	return []byte(strings.TrimSuffix(content, "}") + `,"checksum":"` + hex.EncodeToString(h[:]) + `"}`)
}
func journalFixtureBytes(t testing.TB, j executionJournal) []byte {
	t.Helper()
	b, e := json.Marshal(j)
	if e != nil {
		t.Fatal(e)
	}
	return uncheckedExecutionJournal(string(b))
}
func TestExecutionJournalCanonicalAndIndependent(t *testing.T) {
	j := executionJournalFixture()
	original := j
	b, e := encodeExecutionJournal(j)
	if e != nil || !bytes.Equal(b, uncheckedExecutionJournal(executionJournalGoldenContent)) {
		t.Fatal("canonical bounded journal", e)
	}
	if j != original {
		t.Fatal("encoder changed original")
	}
	got, e := decodeExecutionJournal(b)
	if e != nil || got != j {
		t.Fatal("complete independent record", got, e)
	}
	got.Binding.Installation = "changed"
	b[0] = 'x'
	again, e := decodeExecutionJournal(uncheckedExecutionJournal(executionJournalGoldenContent))
	if e != nil || again != j || original != j {
		t.Fatal("decoded or encoded bytes alias source")
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, uncheckedExecutionJournal(executionJournalGoldenContent), "", "  ") != nil {
		t.Fatal("fixture indent")
	}
	if out, e := decodeExecutionJournal(pretty.Bytes()); e != nil || out != j {
		t.Fatal("presentation changed semantics", e)
	}
}
func TestExecutionJournalPendingDomainsAndClosedState(t *testing.T) {
	j := executionJournalFixture()
	for _, name := range []string{"idle", "metadata", "native", "both", "disarmed", "disarmed-pending", "closed", "historical-native"} {
		t.Run(name, func(t *testing.T) {
			want := j
			switch name {
			case "metadata":
				want.Metadata = journalMetadataFixture(j)
			case "native":
				want.Native = journalNativeFixture(j)
			case "both":
				want.Metadata = journalMetadataFixture(j)
				want.Native = journalNativeFixture(j)
			case "disarmed":
				want.Disarmed = true
			case "disarmed-pending":
				want.Disarmed = true
				want.Metadata = journalMetadataFixture(j)
				want.Native = journalNativeFixture(j)
			case "closed":
				want.Owner.State = "closed"
			case "historical-native":
				want.Native = journalNativeFixture(j)
				want.Owner.ID = strings.Repeat("4", 32)
				want.Owner.Boot = strings.Repeat("ff", 16)
			}
			b, e := encodeExecutionJournal(want)
			if e != nil {
				t.Fatal(e)
			}
			got, e := decodeExecutionJournal(b)
			if e != nil || got != want {
				t.Fatal("lost pending history", e)
			}
		})
	}
	for _, name := range []string{"pending-metadata", "pending-native", "disarmed"} {
		t.Run("closed-"+name, func(t *testing.T) {
			bad := j
			bad.Owner.State = "closed"
			switch name {
			case "pending-metadata":
				bad.Metadata = journalMetadataFixture(j)
			case "pending-native":
				bad.Native = journalNativeFixture(j)
			case "disarmed":
				bad.Disarmed = true
			}
			assertJournalDenied(t, bad)
		})
	}
}
func assertJournalDenied(t testing.TB, j executionJournal) {
	t.Helper()
	if b, e := encodeExecutionJournal(j); e == nil || b != nil {
		t.Fatal("invalid journal encoded")
	}
	if out, e := decodeExecutionJournal(journalFixtureBytes(t, j)); e == nil || out != (executionJournal{}) {
		t.Fatal("invalid rechecksummed journal decoded", e)
	}
}
func TestExecutionJournalSemanticDenials(t *testing.T) {
	base := executionJournalFixture()
	changes := map[string]func(*executionJournal){
		"version": func(j *executionJournal) { j.Version = 2 }, "kind": func(j *executionJournal) { j.Kind = "job" }, "generation": func(j *executionJournal) { j.Generation = 0 },
		"installation": func(j *executionJournal) { j.Binding.Installation = strings.Repeat("A", 32) }, "directory": func(j *executionJournal) { j.Binding.Directory = "" }, "lock": func(j *executionJournal) { j.Binding.Lock = "invalid" }, "ledger-backing": func(j *executionJournal) { j.Binding.LedgerBacking = "" }, "journal-backing": func(j *executionJournal) { j.Binding.JournalBacking = "" }, "owner": func(j *executionJournal) { j.Owner.ID = "" }, "boot-short": func(j *executionJournal) { j.Owner.Boot = "00" }, "boot-uuid": func(j *executionJournal) { j.Owner.Boot = "00000000-0000-0000-0000-000000000000" }, "lifecycle": func(j *executionJournal) { j.Owner.State = "idle" }, "census": func(j *executionJournal) { j.Census = "" },
		"idle-metadata-residue": func(j *executionJournal) { j.Metadata.After = j.Census }, "idle-native-residue": func(j *executionJournal) { j.Native.Supervisor = j.Owner.ID },
		"metadata-before": func(j *executionJournal) {
			j.Metadata = journalMetadataFixture(*j)
			j.Metadata.Before = strings.Repeat("a", 64)
		}, "metadata-after": func(j *executionJournal) { j.Metadata = journalMetadataFixture(*j); j.Metadata.After = j.Census }, "metadata-target": func(j *executionJournal) { j.Metadata = journalMetadataFixture(*j); j.Metadata.Name = "writer.lock" }, "metadata-path": func(j *executionJournal) {
			j.Metadata = journalMetadataFixture(*j)
			j.Metadata.Name = "../ownership-100.json"
		}, "metadata-successor": func(j *executionJournal) { j.Metadata = journalMetadataFixture(*j); j.Metadata.Successor = "" }, "metadata-no-change": func(j *executionJournal) {
			j.Metadata = journalMetadataFixture(*j)
			j.Metadata.Predecessor = j.Metadata.Successor
		},
		"native-job": func(j *executionJournal) { j.Native = journalNativeFixture(*j); j.Native.Job = "" }, "native-submission": func(j *executionJournal) { j.Native = journalNativeFixture(*j); j.Native.Submission = "" }, "native-pool": func(j *executionJournal) { j.Native = journalNativeFixture(*j); j.Native.PoolGUID = "0100" }, "native-filesystem": func(j *executionJournal) { j.Native = journalNativeFixture(*j); j.Native.FilesystemGUID = "0" }, "native-intent": func(j *executionJournal) { j.Native = journalNativeFixture(*j); j.Native.Intent = "" }, "native-owner": func(j *executionJournal) { j.Native = journalNativeFixture(*j); j.Native.Owner = "" }, "native-boot": func(j *executionJournal) { j.Native = journalNativeFixture(*j); j.Native.Boot = "" }, "native-supervisor": func(j *executionJournal) { j.Native = journalNativeFixture(*j); j.Native.Supervisor = "" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) { bad := base; change(&bad); assertJournalDenied(t, bad) })
	}
}
func TestExecutionJournalStrictShapeAndDamage(t *testing.T) {
	good := string(uncheckedExecutionJournal(executionJournalGoldenContent))
	cases := map[string][]byte{"empty": nil, "truncated": []byte(good[:len(good)-1]), "trailing": []byte(good + "{}"), "null": []byte("null"), "deep": []byte(strings.Repeat("[", 1000)), "utf8": append([]byte(good), 255), "unknown": []byte(strings.Replace(good, `"version":1`, `"version":1,"extra":0`, 1)), "duplicate": []byte(strings.Replace(good, `"version":1`, `"version":1,"version":1`, 1)), "case": []byte(strings.Replace(good, `"census":`, `"Census":`, 1)), "missing": []byte(strings.Replace(good, `"disarmed":false,`, "", 1)), "nullable": []byte(strings.Replace(good, `"metadata":{`, `"metadata":null,"removed":{`, 1)), "integer": []byte(strings.Replace(good, `"generation":1`, `"generation":1.0`, 1)), "overflow": []byte(strings.Replace(good, `"generation":1`, `"generation":18446744073709551616`, 1)), "checksum": []byte(strings.Replace(good, `"generation":1`, `"generation":2`, 1)), "wrong-domain": []byte(strings.Replace(good, `"checksum":"`, `"checksum":"x`, 1))}
	wrongDomain := sha256.Sum256([]byte("horvium:job:v1\n" + executionJournalGoldenContent))
	var changed map[string]json.RawMessage
	json.Unmarshal([]byte(good), &changed)
	changed["checksum"], _ = json.Marshal(hex.EncodeToString(wrongDomain[:]))
	cases["foreign-checksum-domain"], _ = json.Marshal(changed)
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			original := append([]byte(nil), data...)
			out, e := decodeExecutionJournal(data)
			if e == nil || out != (executionJournal{}) || !bytes.Equal(data, original) {
				t.Fatal("unsafe bytes accepted or changed", e)
			}
		})
	}
	// Every required top-level and nested field is checked independently, including
	// pending flags and empty values. A missing/null shape is not repaired by defaults.
	var wire map[string]json.RawMessage
	if json.Unmarshal([]byte(good), &wire) != nil {
		t.Fatal("fixture")
	}
	for _, parent := range []string{"", "binding", "owner", "metadata", "native"} {
		object := wire
		if parent != "" {
			object = nil
			if json.Unmarshal(wire[parent], &object) != nil {
				t.Fatal(parent)
			}
		}
		for field := range object {
			for _, action := range []string{"missing", "null", "duplicate", "case"} {
				t.Run(parent+"/"+field+"/"+action, func(t *testing.T) {
					var candidate map[string]json.RawMessage
					json.Unmarshal([]byte(good), &candidate)
					var nested map[string]json.RawMessage
					target := candidate
					if parent != "" {
						json.Unmarshal(candidate[parent], &nested)
						target = nested
					}
					value := target[field]
					delete(target, field)
					switch action {
					case "null":
						target[field] = json.RawMessage("null")
					case "case":
						target[strings.ToUpper(field)] = value
					case "duplicate":
						target[field] = value
					}
					local, _ := json.Marshal(target)
					if action == "duplicate" {
						needle := `"` + field + `":` + string(value)
						local = []byte(strings.Replace(string(local), needle, needle+","+needle, 1))
					}
					data := local
					if parent != "" {
						candidate[parent] = local
						data, _ = json.Marshal(candidate)
					}
					if out, e := decodeExecutionJournal(data); e == nil || out != (executionJournal{}) {
						t.Fatal("required field repaired", e)
					}
				})
			}
		}
	}
}
func TestExecutionJournalLimitsAndRawBoot(t *testing.T) {
	j := executionJournalFixture()
	good := uncheckedExecutionJournal(executionJournalGoldenContent)
	exact := append(append([]byte(nil), good...), bytes.Repeat([]byte(" "), maxStateRecordBytes-len(good))...)
	if out, e := decodeExecutionJournal(exact); e != nil || out != j {
		t.Fatal("exact 64KiB record", e)
	}
	if out, e := decodeExecutionJournal(append(exact, ' ')); !errors.Is(e, ErrLimit) || out != (executionJournal{}) {
		t.Fatal("oversize record", e)
	}
	for _, boot := range []string{strings.Repeat("00", 16), "0102030405060708090a0b0c0d0e0f00", strings.Repeat("ff", 16)} {
		j.Owner.Boot = boot
		b, e := encodeExecutionJournal(j)
		if e != nil {
			t.Fatal(e)
		}
		out, e := decodeExecutionJournal(b)
		if e != nil || out.Owner.Boot != boot {
			t.Fatal("raw 16-byte boot changed", e)
		}
	}
	j = executionJournalFixture()
	j.Generation = math.MaxUint64
	b, e := encodeExecutionJournal(j)
	if e != nil {
		t.Fatal("highest representable generation", e)
	}
	if out, e := decodeExecutionJournal(b); e != nil || out != j {
		t.Fatal("highest generation round trip", e)
	}
	j = executionJournalFixture()
	j.Metadata = journalMetadataFixture(j)
	j.Metadata.Name = strings.Repeat("a", maxStateRecordBytes+1)
	if b, e := encodeExecutionJournal(j); !errors.Is(e, ErrLimit) || b != nil {
		t.Fatal("unbounded input field", e)
	}
}
func journalAdvance(t testing.TB, before, after executionJournal) executionJournal {
	t.Helper()
	after.Generation = before.Generation + 1
	if e := checkExecutionJournalSuccessor(before, after); e != nil {
		t.Fatal("legal metadata successor", e)
	}
	b, e := encodeExecutionJournal(after)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := decodeExecutionJournal(b)
	if e != nil || decoded != after {
		t.Fatal("successor record", e)
	}
	return after
}
func TestExecutionJournalOrderedTransitions(t *testing.T) {
	idle := executionJournalFixture()
	launch := idle
	launch.Native = journalNativeFixture(idle)
	launch = journalAdvance(t, idle, launch)
	applying := launch
	applying.Metadata = journalMetadataFixture(launch)
	applying = journalAdvance(t, launch, applying)
	confirmed := applying
	confirmed.Census = applying.Metadata.After
	confirmed.Metadata = executionJournalMetadata{}
	confirmed = journalAdvance(t, applying, confirmed)
	if confirmed.Native != launch.Native {
		t.Fatal("metadata confirmation erased native uncertainty")
	}
	outcome := confirmed
	outcome.Metadata = journalMetadataFixture(confirmed)
	outcome.Metadata.After = strings.Repeat("b", 64)
	outcome.Metadata.Name = "job-" + launch.Native.Job + ".json"
	outcome.Metadata.Predecessor = strings.Repeat("d", 64)
	outcome = journalAdvance(t, confirmed, outcome)
	final := outcome
	final.Census = outcome.Metadata.After
	final.Metadata = executionJournalMetadata{}
	final = journalAdvance(t, outcome, final)
	retired := final
	retired.Native = executionJournalNative{}
	retired = journalAdvance(t, final, retired)
	closed := retired
	closed.Owner.State = "closed"
	closed = journalAdvance(t, retired, closed)
	if closed.Generation != 8 || closed.Binding != idle.Binding || closed.Census != outcome.Metadata.After {
		t.Fatal("original binding/order changed")
	}
	// Metadata-only publications and the restore barrier use the same successor
	// checks; the barrier is one-way here. Recovery/startup require native gates.
	pending := idle
	pending.Metadata = journalMetadataFixture(idle)
	pending = journalAdvance(t, idle, pending)
	disarmed := pending
	disarmed.Disarmed = true
	journalAdvance(t, pending, disarmed)
}
func TestExecutionJournalTransitionDenials(t *testing.T) {
	idle := executionJournalFixture()
	meta := idle
	meta.Metadata = journalMetadataFixture(idle)
	meta.Generation = 2
	native := idle
	native.Native = journalNativeFixture(idle)
	native.Generation = 2
	both := native
	both.Metadata = journalMetadataFixture(native)
	both.Generation = 3
	cases := map[string]struct{ before, after executionJournal }{}
	add := func(name string, before executionJournal, edit func(*executionJournal)) {
		after := before
		after.Generation++
		edit(&after)
		cases[name] = struct{ before, after executionJournal }{before, after}
	}
	add("noop", idle, func(*executionJournal) {})
	add("generation-gap", idle, func(j *executionJournal) { j.Generation++; j.Disarmed = true })
	add("generation-repeat", idle, func(j *executionJournal) { j.Generation--; j.Disarmed = true })
	add("binding", idle, func(j *executionJournal) { j.Binding.Lock = strings.Repeat("a", 64); j.Disarmed = true })
	add("owner", idle, func(j *executionJournal) { j.Owner.ID = strings.Repeat("3", 32); j.Disarmed = true })
	add("boot", idle, func(j *executionJournal) { j.Owner.Boot = strings.Repeat("ff", 16); j.Disarmed = true })
	add("unconfirmed-census", idle, func(j *executionJournal) { j.Census = strings.Repeat("f", 64) })
	add("confirm-wrong-census", meta, func(j *executionJournal) { j.Metadata = executionJournalMetadata{}; j.Census = strings.Repeat("b", 64) })
	add("confirm-keeps-pending", meta, func(j *executionJournal) { j.Census = j.Metadata.After })
	add("replace-pending-metadata", meta, func(j *executionJournal) { j.Metadata.Successor = strings.Repeat("b", 64) })
	add("discard-pending-metadata", meta, func(j *executionJournal) { j.Metadata = executionJournalMetadata{} })
	add("begin-native-during-metadata", meta, func(j *executionJournal) { j.Native = journalNativeFixture(*j) })
	add("foreign-native-owner", idle, func(j *executionJournal) {
		j.Native = journalNativeFixture(*j)
		j.Native.Owner = strings.Repeat("3", 32)
	})
	add("foreign-native-boot", idle, func(j *executionJournal) {
		j.Native = journalNativeFixture(*j)
		j.Native.Boot = strings.Repeat("ff", 16)
	})
	add("replace-native", native, func(j *executionJournal) { j.Native.Supervisor = strings.Repeat("4", 32) })
	add("clear-native-during-metadata", both, func(j *executionJournal) { j.Native = executionJournalNative{} })
	add("confirm-and-clear-native", both, func(j *executionJournal) {
		j.Census = j.Metadata.After
		j.Metadata = executionJournalMetadata{}
		j.Native = executionJournalNative{}
	})
	add("launch-and-metadata", idle, func(j *executionJournal) {
		j.Native = journalNativeFixture(*j)
		j.Metadata = journalMetadataFixture(*j)
	})
	add("close-during-native", native, func(j *executionJournal) { j.Owner.State = "closed" })
	add("close-during-metadata", meta, func(j *executionJournal) { j.Owner.State = "closed" })
	disarmed := idle
	disarmed.Disarmed = true
	add("rearm", disarmed, func(j *executionJournal) { j.Disarmed = false })
	add("publish-after-disarm", disarmed, func(j *executionJournal) { j.Metadata = journalMetadataFixture(*j) })
	closed := idle
	closed.Owner.State = "closed"
	add("reopen-closed", closed, func(j *executionJournal) { j.Owner.State = "active" })
	add("disarm-with-effect", idle, func(j *executionJournal) { j.Disarmed = true; j.Native = journalNativeFixture(*j) })
	historical := native
	historical.Owner.ID = strings.Repeat("4", 32)
	add("clear-historical-native", historical, func(j *executionJournal) { j.Native = executionJournalNative{} })
	badBefore := idle
	badBefore.Version = 2
	add("invalid-before", badBefore, func(j *executionJournal) { j.Version = 1; j.Disarmed = true })
	add("invalid-after", idle, func(j *executionJournal) { j.Version = 2; j.Disarmed = true })
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			before, after := c.before, c.after
			if e := checkExecutionJournalSuccessor(before, after); e == nil {
				t.Fatal("invalid transition accepted")
			}
			if before != c.before || after != c.after {
				t.Fatal("validator changed inputs")
			}
		})
	}
	idle.Generation = math.MaxUint64
	after := idle
	after.Generation = 0
	after.Disarmed = true
	if e := checkExecutionJournalSuccessor(idle, after); !errors.Is(e, ErrLimit) {
		t.Fatal("generation overflow", e)
	}
}
func FuzzExecutionJournal(f *testing.F) {
	f.Add(uncheckedExecutionJournal(executionJournalGoldenContent))
	f.Add([]byte("{}"))
	f.Add([]byte("null"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxStateRecordBytes+1 {
			t.Skip()
		}
		original := append([]byte(nil), data...)
		j, e := decodeExecutionJournal(data)
		if !bytes.Equal(data, original) {
			t.Fatal("decoder modified bytes")
		}
		if e != nil {
			if j != (executionJournal{}) {
				t.Fatal("partial record on failure")
			}
			return
		}
		canonical, e := encodeExecutionJournal(j)
		if e != nil || len(canonical) > maxStateRecordBytes {
			t.Fatal("valid record cannot encode", e)
		}
		again, e := decodeExecutionJournal(canonical)
		if e != nil || again != j {
			t.Fatal("canonical round trip", e)
		}
	})
}
