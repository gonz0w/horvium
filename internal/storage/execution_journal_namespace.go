// SPDX-License-Identifier: BSD-3-Clause
package storage

import "strings"

const executionJournalName = "execution.json"
const executionJournalTemporaryPrefix = ".execution-"

// Private journal namespace, separate from ledger records and their exports.
// Only the native publisher may exclude its independently verified temporary.
func executionJournalNames(names []string, ownTemporary string) (bool, error) {
	// Permit one extra name to identify a retained orphan explicitly. Larger
	// directories are never scanned or accumulated without a fixed bound.
	if len(names) > 3 || len(ownTemporary) > maxFieldBytes {
		return false, ErrLimit
	}
	if ownTemporary != "" && !executionJournalTemporary(ownTemporary) {
		return false, ErrStateRecord
	}
	seen := make(map[string]bool, len(names))
	present, own := false, false
	for _, name := range names {
		if len(name) > maxFieldBytes {
			return false, ErrLimit
		}
		if seen[name] {
			return false, ErrDrift
		}
		seen[name] = true
		switch {
		case name == "writer.lock":
		case name == executionJournalName:
			present = true
		case ownTemporary != "" && name == ownTemporary:
			own = true
		case executionJournalTemporary(name):
			return false, ErrStateUncertain
		default:
			return false, ErrStateRecord
		}
	}
	if !seen["writer.lock"] || ownTemporary != "" && !own {
		return false, ErrDrift
	}
	return present, nil
}

func executionJournalTemporary(name string) bool {
	return len(name) == len(executionJournalTemporaryPrefix)+32+4 && strings.HasPrefix(name, executionJournalTemporaryPrefix) && strings.HasSuffix(name, ".tmp") && validJournalIdentity(name[len(executionJournalTemporaryPrefix):len(name)-4])
}

func checkInitialExecutionJournal(j executionJournal) error {
	if err := validateExecutionJournal(j); err != nil {
		return err
	}
	if j.Generation != 1 || j.Owner.State != "active" || j.Metadata.Pending || j.Native.Pending || j.Disarmed {
		return ErrDenied
	}
	return nil
}
