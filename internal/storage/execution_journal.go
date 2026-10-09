// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"
	"unicode/utf8"
)

// Internal journal metadata only. No decoded bytes or successful transition
// check grants a lease, publication acknowledgement, launch or restart authority.
// The native owner must independently establish every accepted ADR 0009 gate.
type executionJournalBinding struct {
	Installation   string `json:"installation"`
	Directory      string `json:"directory"`
	Lock           string `json:"lock"`
	LedgerBacking  string `json:"ledger_backing"`
	JournalBacking string `json:"journal_backing"`
}
type executionJournalOwner struct {
	ID    string `json:"id"`
	Boot  string `json:"boot"`
	State string `json:"state"`
}
type executionJournalMetadata struct {
	Pending     bool   `json:"pending"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Name        string `json:"name"`
	Predecessor string `json:"predecessor"`
	Successor   string `json:"successor"`
}
type executionJournalNative struct {
	Pending        bool   `json:"pending"`
	Job            string `json:"job"`
	Submission     string `json:"submission"`
	PoolGUID       string `json:"pool_guid"`
	FilesystemGUID string `json:"filesystem_guid"`
	Intent         string `json:"intent"`
	Owner          string `json:"owner"`
	Boot           string `json:"boot"`
	Supervisor     string `json:"supervisor"`
}
type executionJournal struct {
	Version    uint64                   `json:"version"`
	Kind       string                   `json:"kind"`
	Generation uint64                   `json:"generation"`
	Binding    executionJournalBinding  `json:"binding"`
	Owner      executionJournalOwner    `json:"owner"`
	Census     string                   `json:"census"`
	Metadata   executionJournalMetadata `json:"metadata"`
	Native     executionJournalNative   `json:"native"`
	Disarmed   bool                     `json:"disarmed"`
}

// A separate explicit wire shape keeps the checksum outside the domain-bound
// content and uses the existing fixed-depth strict JSON walker without changing
// ownership/job formats. Every field is required, including idle empty fields.
type executionJournalRecord struct {
	Version    uint64                   `json:"version"`
	Kind       string                   `json:"kind"`
	Generation uint64                   `json:"generation"`
	Binding    executionJournalBinding  `json:"binding"`
	Owner      executionJournalOwner    `json:"owner"`
	Census     string                   `json:"census"`
	Metadata   executionJournalMetadata `json:"metadata"`
	Native     executionJournalNative   `json:"native"`
	Disarmed   bool                     `json:"disarmed"`
	Checksum   string                   `json:"checksum"`
}

func encodeExecutionJournal(j executionJournal) ([]byte, error) {
	if err := validateExecutionJournal(j); err != nil {
		return nil, err
	}
	r := executionJournalRecord{j.Version, j.Kind, j.Generation, j.Binding, j.Owner, j.Census, j.Metadata, j.Native, j.Disarmed, executionJournalChecksum(j)}
	b, err := json.Marshal(r)
	if err != nil {
		return nil, ErrStateRecord
	}
	if len(b) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	return b, nil
}
func decodeExecutionJournal(data []byte) (executionJournal, error) {
	if len(data) > maxStateRecordBytes {
		return executionJournal{}, ErrLimit
	}
	if !utf8.Valid(data) || !ownershipShape(data, executionJournalRecord{}) {
		return executionJournal{}, ErrStateRecord
	}
	var r executionJournalRecord
	if json.Unmarshal(data, &r) != nil {
		return executionJournal{}, ErrStateRecord
	}
	j := executionJournal{r.Version, r.Kind, r.Generation, r.Binding, r.Owner, r.Census, r.Metadata, r.Native, r.Disarmed}
	if err := validateExecutionJournal(j); err != nil {
		return executionJournal{}, err
	}
	if !validJobDigest(r.Checksum) || r.Checksum != executionJournalChecksum(j) {
		return executionJournal{}, ErrStateRecord
	}
	return j, nil
}
func executionJournalChecksum(j executionJournal) string {
	b, _ := json.Marshal(j)
	h := sha256.New()
	_, _ = h.Write([]byte("horvium:execution-journal:v1\n"))
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}
func validJournalIdentity(s string) bool {
	// Raw sixteen bytes encoded as exact lowercase hex. No UUID/string trimming
	// or claim that caller-supplied identity is fresh or native boot provenance.
	return len(s) == 32 && strings.IndexFunc(s, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) < 0
}
func validExecutionJournalBinding(b executionJournalBinding) bool {
	return validJournalIdentity(b.Installation) && validJobDigest(b.Directory) && validJobDigest(b.Lock) && validJobDigest(b.LedgerBacking) && validJobDigest(b.JournalBacking)
}

func validateExecutionJournal(j executionJournal) error {
	// Refuse excessive caller-owned fields before copying or marshalling. Valid
	// records have a fixed field count, no inventory/history and no dynamic arrays.
	for _, s := range []string{j.Kind, j.Binding.Installation, j.Binding.Directory, j.Binding.Lock, j.Binding.LedgerBacking, j.Binding.JournalBacking, j.Owner.ID, j.Owner.Boot, j.Owner.State, j.Census, j.Metadata.Before, j.Metadata.After, j.Metadata.Name, j.Metadata.Predecessor, j.Metadata.Successor, j.Native.Job, j.Native.Submission, j.Native.PoolGUID, j.Native.FilesystemGUID, j.Native.Intent, j.Native.Owner, j.Native.Boot, j.Native.Supervisor} {
		if len(s) > maxFieldBytes {
			return ErrLimit
		}
	}
	if j.Version != 1 || j.Kind != "execution-journal" || j.Generation == 0 || !validExecutionJournalBinding(j.Binding) || !validJournalIdentity(j.Owner.ID) || !validJournalIdentity(j.Owner.Boot) || j.Owner.State != "active" && j.Owner.State != "closed" || !validJobDigest(j.Census) {
		return ErrStateRecord
	}
	m := j.Metadata
	if !m.Pending {
		if m != (executionJournalMetadata{}) {
			return ErrStateRecord
		}
	} else {
		entries, err := stateCatalogNames([]string{"writer.lock", m.Name}, defaultStateCatalogLimits)
		if err != nil || len(entries) != 1 || !validJobDigest(m.Before) || m.Before != j.Census || !validJobDigest(m.After) || m.After == m.Before || m.Predecessor != "" && !validJobDigest(m.Predecessor) || !validJobDigest(m.Successor) || m.Predecessor == m.Successor {
			return ErrStateRecord
		}
	}
	n := j.Native
	if !n.Pending {
		if n != (executionJournalNative{}) {
			return ErrStateRecord
		}
	} else if !validJobDigest(n.Job) || !validJobDigest(n.Submission) || len(n.PoolGUID) > 20 || !validGUID(n.PoolGUID) || len(n.FilesystemGUID) > 20 || !validGUID(n.FilesystemGUID) || !validJobDigest(n.Intent) || !validJournalIdentity(n.Owner) || !validJournalIdentity(n.Boot) || !validJournalIdentity(n.Supervisor) {
		return ErrStateRecord
	}
	if j.Owner.State == "closed" && (m.Pending || n.Pending || j.Disarmed) {
		return ErrStateRecord
	}
	return nil
}

// Pure single-successor validation, never publication or recovery authority.
// The future original native owner must still prove exact complete census,
// durable acknowledgements, original supervisor completion and observed outcome
// before publishing or clearing anything. In particular, accepting native-clear
// bytes here does NOT establish those facts. Startup/rebinding is not implemented.
func checkExecutionJournalSuccessor(before, after executionJournal) error {
	if err := validateExecutionJournal(before); err != nil {
		return err
	}
	if before.Generation == math.MaxUint64 {
		return ErrLimit
	}
	if err := validateExecutionJournal(after); err != nil {
		return err
	}
	if after.Generation != before.Generation+1 || before.Binding != after.Binding || before.Owner.State != "active" || before.Disarmed {
		return ErrDenied
	}
	// Exactly one logical transition candidate; ordinary successors cannot
	// replace original owner/boot/binding, rearm restoration or resurrect closed state.
	expected := before
	expected.Generation++
	switch {
	case !before.Disarmed && after.Disarmed:
		expected.Disarmed = true
	case before.Owner.State == "active" && after.Owner.State == "closed":
		if before.Metadata.Pending || before.Native.Pending {
			return ErrDenied
		}
		expected.Owner.State = "closed"
	case !before.Metadata.Pending && after.Metadata.Pending:
		expected.Metadata = after.Metadata
	case before.Metadata.Pending && !after.Metadata.Pending:
		expected.Census = before.Metadata.After
		expected.Metadata = executionJournalMetadata{}
	case !before.Native.Pending && after.Native.Pending:
		if before.Metadata.Pending || after.Native.Owner != before.Owner.ID || after.Native.Boot != before.Owner.Boot {
			return ErrDenied
		}
		expected.Native = after.Native
	case before.Native.Pending && !after.Native.Pending:
		if before.Metadata.Pending || before.Native.Owner != before.Owner.ID || before.Native.Boot != before.Owner.Boot {
			return ErrDenied
		}
		expected.Native = executionJournalNative{}
	default:
		return ErrDenied
	}
	if expected != after {
		return ErrDenied
	}
	return nil
}
