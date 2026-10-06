// SPDX-License-Identifier: BSD-3-Clause
package storage

// Requires a fresh complete locked native census. The entire new file must fit
// alongside every old record; rename has not reclaimed destination bytes yet.
func statePublicationCapacity(c StateCatalog, size uint64, limits stateCatalogLimits) error {
	if !limits.valid() || size == 0 || size > maxStateRecordBytes || c.Bytes > limits.bytes || size > limits.bytes-c.Bytes || len(c.Ownership) > limits.ownership || c.Nonterminal < 0 || c.Terminal < 0 || c.Nonterminal > limits.nonterminal || c.Terminal > limits.terminal || len(c.Jobs) != c.Nonterminal+c.Terminal {
		return ErrLimit
	}
	return nil
}

func allowMixedOwnershipPublication(c StateCatalog, next Policy, expected, size uint64, limits stateCatalogLimits) error {
	if err := statePublicationCapacity(c, size, limits); err != nil {
		return err
	}
	return allowOwnershipPublication(OwnershipCatalog{Records: c.Ownership, Bytes: c.Bytes}, next, expected, size, ownershipPublicationLimits{limits.ownership, limits.bytes})
}

// Capacity and submission consistency only: the native caller must additionally
// read the current predecessor and apply ValidateJobSuccessor under the lease.
func allowJobPublication(c StateCatalog, next Job, expected, size uint64, limits stateCatalogLimits) error {
	if err := validateJob(next); err != nil {
		return err
	}
	if !validOwnershipPublicationRevision(uint64(len(next.History)), expected) {
		return ErrDrift
	}
	if err := statePublicationCapacity(c, size, limits); err != nil {
		return err
	}
	found := false
	nonterminal, terminal := c.Nonterminal, c.Terminal
	for _, record := range c.Jobs {
		if record.ID != next.ID {
			if record.SubmissionID == next.Intent.SubmissionID {
				return ErrDrift
			}
			continue
		}
		if found || expected == 0 || record.Revision != expected || record.SubmissionID != next.Intent.SubmissionID || next.History[expected-1].State != record.State {
			return ErrDrift
		}
		found = true
		if record.State == JobSucceeded || record.State == JobFailed || record.State == JobCancelled {
			terminal--
		} else {
			nonterminal--
		}
	}
	if !found && expected != 0 {
		return ErrDrift
	}
	if next.Terminal() {
		terminal++
	} else {
		nonterminal++
	}
	if nonterminal > limits.nonterminal || terminal > limits.terminal {
		return ErrLimit
	}
	return nil
}
