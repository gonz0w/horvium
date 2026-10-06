// SPDX-License-Identifier: BSD-3-Clause
package storage

// ValidateJobSuccessor refuses identity/intent rewrites, stale or skipped
// updates, and alteration/truncation of earlier events. The predecessor must
// later come from a trusted current durable record. This pure guard performs
// no publication, reservation, admission or native execution. Callers own both
// inputs during the call; neither input is modified.
func ValidateJobSuccessor(previous, next Job) error {
	if err := validateJob(previous); err != nil {
		return err
	}
	if err := validateJob(next); err != nil {
		return err
	}
	if previous.ID != next.ID || previous.Intent != next.Intent || len(next.History) != len(previous.History)+1 {
		return ErrDrift
	}
	for i, event := range previous.History {
		if event != next.History[i] {
			return ErrDrift
		}
	}
	return nil
}
