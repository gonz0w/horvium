// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"strings"
	"testing"
)

func bindingDomains() (executionDomainIdentity, executionDomainIdentity) {
	l := executionDomainIdentity{executionBackingIdentity{kind: "zfs", pool: 123, dataset: 456}, executionObjectIdentity{10, 11, 1700000000, 12}, executionObjectIdentity{20, 21, 1700000001, 22}}
	j := l
	j.backing.dataset = 789
	j.directory.inode = 30
	j.lock.inode = 40
	return l, j
}

func TestExecutionBindingIdentity(t *testing.T) {
	l, j := bindingDomains()
	id := strings.Repeat("a", 32)
	b, e := makeExecutionBinding(id, l, j)
	if e != nil || !validExecutionJournalBinding(b) {
		t.Fatal("valid private metadata", e)
	}
	again, e := makeExecutionBinding(id, l, j)
	if e != nil || again != b {
		t.Fatal("unstable binding", e)
	}
	if b.Directory == b.Lock || b.LedgerBacking == b.JournalBacking {
		t.Fatal("roles/backings conflated")
	}
	// Every independently persistent component must change the corresponding
	// object or backing binding, including inode reuse with a new generation.
	for _, field := range []string{"ledger-inode", "ledger-generation", "ledger-birth", "ledger-nanos", "ledger-lock", "ledger-pool", "ledger-dataset", "journal-directory", "journal-lock", "journal-pool", "journal-dataset", "installation", "roles"} {
		t.Run(field, func(t *testing.T) {
			ll, jj, ii := l, j, id
			switch field {
			case "ledger-inode":
				ll.directory.inode++
			case "ledger-generation":
				ll.directory.generation++
			case "ledger-birth":
				ll.directory.birthSeconds++
			case "ledger-nanos":
				ll.directory.birthNanos++
			case "ledger-lock":
				ll.lock.generation++
			case "ledger-pool":
				ll.backing.pool++
			case "ledger-dataset":
				ll.backing.dataset++
			case "journal-directory":
				jj.directory.generation++
			case "journal-lock":
				jj.lock.inode++
			case "journal-pool":
				jj.backing.pool++
			case "journal-dataset":
				jj.backing.dataset++
			case "installation":
				ii = strings.Repeat("b", 32)
			case "roles":
				ll, jj = jj, ll
			}
			got, e := makeExecutionBinding(ii, ll, jj)
			if e != nil || got == b {
				t.Fatal("substitution not bound", e)
			}
		})
	}
	// One UFS filesystem can contain distinct protected directories; only a
	// separately qualified file-level restore procedure can make this usable.
	l.backing = executionBackingIdentity{kind: "ufs", ufs: [2]uint32{123, 456}}
	j.backing = l.backing
	u, e := makeExecutionBinding(id, l, j)
	if e != nil || u.LedgerBacking != u.JournalBacking || u == b {
		t.Fatal("UFS representation", e)
	}
	l.backing.ufs[1]++
	v, e := makeExecutionBinding(id, l, j)
	if e != nil || v.LedgerBacking == u.LedgerBacking || v.Directory == u.Directory || v.Lock == u.Lock {
		t.Fatal("UFS identity substitution", e)
	}
}

func TestExecutionBindingDenials(t *testing.T) {
	l, j := bindingDomains()
	for _, field := range []string{"empty-installation", "long-installation", "uppercase", "unknown-fs", "zero-pool", "zero-dataset", "foreign-ufs", "zero-ufs", "partial-ufs", "foreign-zfs", "same-zfs-dataset", "same-directory", "same-lock", "same-role-object", "zero-inode", "zero-generation", "missing-birth", "negative-birth", "invalid-nanos", "journal-invalid"} {
		t.Run(field, func(t *testing.T) {
			ll, jj, id := l, j, strings.Repeat("a", 32)
			switch field {
			case "empty-installation":
				id = ""
			case "long-installation":
				id = strings.Repeat("a", 4097)
			case "uppercase":
				id = strings.Repeat("A", 32)
			case "unknown-fs":
				ll.backing.kind = "nfs"
			case "zero-pool":
				ll.backing.pool = 0
			case "zero-dataset":
				ll.backing.dataset = 0
			case "foreign-ufs":
				ll.backing.ufs[0] = 1
			case "zero-ufs":
				ll.backing = executionBackingIdentity{kind: "ufs"}
			case "partial-ufs":
				ll.backing = executionBackingIdentity{kind: "ufs", ufs: [2]uint32{1, 0}}
			case "foreign-zfs":
				ll.backing.kind = "ufs"
				ll.backing.ufs = [2]uint32{1, 2}
			case "same-zfs-dataset":
				jj.backing = ll.backing
			case "same-directory":
				jj = ll
				jj.lock.inode++
			case "same-lock":
				jj = ll
				jj.directory.inode++
			case "same-role-object":
				ll.lock = ll.directory
			case "zero-inode":
				ll.directory.inode = 0
			case "zero-generation":
				ll.lock.generation = 0
			case "missing-birth":
				ll.directory.birthSeconds = 0
			case "negative-birth":
				ll.lock.birthSeconds = -1
			case "invalid-nanos":
				ll.lock.birthNanos = 1000000000
			case "journal-invalid":
				jj.lock = executionObjectIdentity{}
			}
			b, e := makeExecutionBinding(id, ll, jj)
			if e == nil || b != (executionJournalBinding{}) {
				t.Fatal("invalid binding or partial output", e)
			}
		})
	}
}
