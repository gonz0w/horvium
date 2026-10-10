// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"strings"
	"testing"
)

func TestExecutionDomainJournalCorrespondence(t *testing.T) {
	j := executionJournalFixture()
	b := j.Binding
	if e := checkExecutionDomainJournal(b, false, executionJournal{}); e != nil {
		t.Fatal("absent journal observation", e)
	}
	for _, name := range []string{"active", "metadata", "native", "both", "disarmed", "closed"} {
		jj := j
		switch name {
		case "metadata":
			jj.Metadata = journalMetadataFixture(j)
		case "native":
			jj.Native = journalNativeFixture(j)
		case "both":
			jj.Metadata = journalMetadataFixture(j)
			jj.Native = journalNativeFixture(j)
		case "disarmed":
			jj.Disarmed = true
		case "closed":
			jj.Owner.State = "closed"
		}
		if e := checkExecutionDomainJournal(b, true, jj); e != nil {
			t.Fatal("valid observed history lost", name, e)
		}
	}
	for _, name := range []string{"invalid-binding", "missing-record", "stale-absent", "newer", "installation", "directory", "lock", "ledger", "journal"} {
		bb, jj, present := b, j, true
		switch name {
		case "invalid-binding":
			bb.Installation = ""
		case "missing-record":
			jj = executionJournal{}
		case "stale-absent":
			present = false
		case "newer":
			jj.Version++
		case "installation":
			jj.Binding.Installation = strings.Repeat("3", 32)
		case "directory":
			jj.Binding.Directory = strings.Repeat("f", 64)
		case "lock":
			jj.Binding.Lock = strings.Repeat("f", 64)
		case "ledger":
			jj.Binding.LedgerBacking = strings.Repeat("f", 64)
		case "journal":
			jj.Binding.JournalBacking = strings.Repeat("f", 64)
		}
		if e := checkExecutionDomainJournal(bb, present, jj); e == nil {
			t.Fatal("foreign or partial journal accepted", name)
		}
	}
}
