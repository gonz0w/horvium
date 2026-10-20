// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"strings"
	"testing"
)

const literalControl = `{"format":"horvium-storage-control","version":1,"journal_directory":"/var/db/horvium-journal","ledger_directory":"/var/db/horvium-ledger"}`

func TestControlConfigLiteralAndExactShape(t *testing.T) {
	got, e := ParseControlConfig(strings.NewReader(literalControl))
	if e != nil || got.JournalDirectory != "/var/db/horvium-journal" || got.LedgerDirectory != "/var/db/horvium-ledger" {
		t.Fatalf("literal selection: %#v %v", got, e)
	}
	bad := []string{
		strings.Replace(literalControl, `"version":1`, `"version":1,"ver\u0073ion":1`, 1),
		strings.Replace(literalControl, `"version":1`, `"Version":1`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":1,"secret":"sentinel"`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":null`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":"1"`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":1.0`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":1e0`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":+1`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":-1`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":true`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":{}`, 1),
		strings.Replace(literalControl, `"version":1`, `"version":[]`, 1),
		strings.Replace(literalControl, `"format":"horvium-storage-control",`, "", 1),
		strings.Replace(literalControl, `"journal_directory":"/var/db/horvium-journal"`, `"journal_directory":null`, 1),
		strings.Replace(literalControl, `/var/db/horvium-journal`, `/var/db/horvium-ledger/child`, 1),
		strings.Replace(literalControl, `/var/db/horvium-journal`, `/var/db/horvium-ledger`, 1),
		literalControl + ` {}`, "null", "[]", "", "\xef\xbb\xbf" + literalControl,
		strings.Replace(literalControl, "horvium-journal", `\ud800`, 1),
		strings.Replace(literalControl, "horvium-journal", `\udc00`, 1),
		strings.Replace(literalControl, "horvium-journal", `\u0000`, 1),
		string(append([]byte(literalControl[:10]), 0xff)),
	}
	for i, input := range bad {
		if _, e := ParseControlConfig(strings.NewReader(input)); e == nil {
			t.Errorf("invalid literal %d admitted", i)
		}
	}
	if _, e := ParseControlConfig(nil); e == nil {
		t.Fatal("nil reader admitted")
	}
	if _, e := ParseControlConfig(strings.NewReader(strings.Replace(literalControl, "horvium-journal", `\ud83d\ude00`, 1))); e != nil {
		t.Fatal("valid surrogate pair rejected", e)
	}
	for _, path := range []string{"", "/", "relative", "~/state", "/var//state", "/var/./state", "/var/../state", "/var/state/", "/var/\x7fstate", "/var/\nstate", "/" + strings.Repeat("x", 4096)} {
		if ValidControlPath(path) {
			t.Errorf("invalid path admitted %q", path)
		}
	}
	if !ValidControlPath("/" + strings.Repeat("x", 4095)) {
		t.Fatal("4096-byte path rejected")
	}
	padding := maxControlConfigBytes - len(literalControl)
	if _, e := ParseControlConfig(strings.NewReader(literalControl + strings.Repeat(" ", padding))); e != nil {
		t.Fatal("exact byte bound", e)
	}
	if _, e := ParseControlConfig(strings.NewReader(literalControl + strings.Repeat(" ", padding+1))); e == nil {
		t.Fatal("oversize selection admitted")
	}
}

func FuzzControlConfig(f *testing.F) {
	f.Add([]byte(literalControl))
	f.Fuzz(func(t *testing.T, data []byte) {
		got, e := ParseControlConfig(bytes.NewReader(data))
		if e == nil && (len(data) > maxControlConfigBytes || !ValidControlPath(got.JournalDirectory) || !ValidControlPath(got.LedgerDirectory) || controlPathsOverlap(got.JournalDirectory, got.LedgerDirectory)) {
			t.Fatal("invalid selection admitted")
		}
	})
}
