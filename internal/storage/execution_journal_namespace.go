// SPDX-License-Identifier: BSD-3-Clause
package storage

const executionJournalName = "execution.json"
const executionJournalTemporaryPrefix = ".execution-"

// Private journal namespace, separate from ledger records and their exports.
// Only the native publisher may exclude its independently verified temporary.
func executionJournalNames(names []string, ownTemporary string) (bool, error) {
	return false, ErrDenied
}

func checkInitialExecutionJournal(j executionJournal) error {
	return ErrDenied
}
