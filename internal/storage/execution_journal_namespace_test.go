// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"strings"
	"testing"
)

func TestExecutionJournalNamespace(t *testing.T) {
	temporary := executionJournalTemporaryPrefix + strings.Repeat("a", 32) + ".tmp"
	for _, tc := range []struct {
		name    string
		names   []string
		own     string
		present bool
		err     error
	}{
		{"empty-domain", []string{"writer.lock"}, "", false, nil},
		{"existing", []string{executionJournalName, "writer.lock"}, "", true, nil},
		{"own-create-temp", []string{temporary, "writer.lock"}, temporary, false, nil},
		{"own-replace-temp", []string{executionJournalName, temporary, "writer.lock"}, temporary, true, nil},
		{"missing-lock", []string{executionJournalName}, "", false, ErrDrift},
		{"duplicate-lock", []string{"writer.lock", "writer.lock"}, "", false, ErrDrift},
		{"duplicate-record", []string{"writer.lock", executionJournalName, executionJournalName}, "", false, ErrDrift},
		{"unknown", []string{"writer.lock", "unknown"}, "", false, ErrStateRecord},
		{"ledger-record", []string{"writer.lock", "ownership-100.json"}, "", false, ErrStateRecord},
		{"alias", []string{"writer.lock", "../execution.json"}, "", false, ErrStateRecord},
		{"orphan", []string{"writer.lock", temporary}, "", false, ErrStateUncertain},
		{"foreign-temp", []string{"writer.lock", temporary, executionJournalTemporaryPrefix + strings.Repeat("b", 32) + ".tmp"}, temporary, false, ErrStateUncertain},
		{"absent-own-temp", []string{"writer.lock", executionJournalName}, temporary, false, ErrDrift},
		{"bad-own-temp", []string{"writer.lock", "unknown"}, "unknown", false, ErrStateRecord},
		{"count", []string{"writer.lock", executionJournalName, temporary, "extra"}, temporary, false, ErrLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			present, e := executionJournalNames(tc.names, tc.own)
			if !errors.Is(e, tc.err) || present != tc.present {
				t.Fatal("journal namespace", present, e)
			}
		})
	}
}

func TestExecutionJournalInitialBoundary(t *testing.T) {
	j := executionJournalFixture()
	if checkInitialExecutionJournal(j) != nil {
		t.Fatal("initial bounded metadata")
	}
	for _, name := range []string{"generation", "closed", "metadata", "native", "disarmed", "invalid"} {
		t.Run(name, func(t *testing.T) {
			other := j
			switch name {
			case "generation":
				other.Generation = 2
			case "closed":
				other.Owner.State = "closed"
			case "metadata":
				other.Metadata = journalMetadataFixture(j)
			case "native":
				other.Native = journalNativeFixture(j)
			case "disarmed":
				other.Disarmed = true
			case "invalid":
				other.Binding.Lock = ""
			}
			if checkInitialExecutionJournal(other) == nil {
				t.Fatal("noninitial metadata accepted")
			}
		})
	}
}
