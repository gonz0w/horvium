// SPDX-License-Identifier: BSD-3-Clause
package storage

import "testing"

func TestControlRestoreScopeLiveFacts(t *testing.T) {
	config := propertyStateBacking{kind: "zfs", source: "system/config", target: "/etc/control", fsid: [2]int32{1, 2}}
	ledger := propertyStateBacking{kind: "zfs", source: "system/ledger", target: "/var/db/ledger", fsid: [2]int32{3, 4}}
	if checkControlRestoreScope(config, ledger) != nil {
		t.Fatal("independent live scopes rejected")
	}
	for _, change := range []func(*propertyStateBacking){
		func(c *propertyStateBacking) { c.fsid = ledger.fsid },
		func(c *propertyStateBacking) { c.source = ledger.source },
		func(c *propertyStateBacking) { c.source = ledger.source + "/config" },
		func(c *propertyStateBacking) { c.kind = "nfs" },
		func(c *propertyStateBacking) { c.fsid = [2]int32{} },
		func(c *propertyStateBacking) { c.readonly = true },
		func(c *propertyStateBacking) { c.softupdates = true },
	} {
		got := config
		change(&got)
		if checkControlRestoreScope(got, ledger) == nil {
			t.Fatal("unsafe/unknown restore scope admitted")
		}
	}
	config.source = "system/ledger-sibling"
	if checkControlRestoreScope(config, ledger) != nil {
		t.Fatal("component sibling treated as descendant")
	}
	config.source = "system" // A precisely ledger-only rollback does not affect parent.
	if checkControlRestoreScope(config, ledger) != nil {
		t.Fatal("config ancestor rejected for ledger-only scope")
	}
	config.kind = "ufs"
	ledger.kind = "ufs"
	config.fsid = ledger.fsid
	config.source = "/dev/ada0p2"
	ledger.source = config.source
	if checkControlRestoreScope(config, ledger) != nil {
		t.Fatal("UFS file-only restoration incorrectly requires another FS")
	}
}

func TestExecutionInspectionCompleteClosedMetadata(t *testing.T) {
	j := executionJournalFixture()
	j.Owner.State = "closed"
	o := ExecutionDomainsObservation{origin: &ExecutionDomainLeases{}, binding: j.Binding, census: j.Census, journalPresent: true, journal: j, systemObserved: true}
	if state, e := o.controlInspectionState(); e != nil || state != "closed" {
		t.Fatal("definite closed metadata", state, e)
	}
	for _, change := range []func(*ExecutionDomainsObservation){
		func(o *ExecutionDomainsObservation) { o.jobsPresent = true },
		func(o *ExecutionDomainsObservation) { o.journal.Owner.State = "active" },
		func(o *ExecutionDomainsObservation) { o.journal.Disarmed = true },
		func(o *ExecutionDomainsObservation) { o.journal.Metadata.Pending = true },
		func(o *ExecutionDomainsObservation) { o.journal.Native.Pending = true },
		func(o *ExecutionDomainsObservation) { o.census = "changed" },
	} {
		got := o
		change(&got)
		if state, e := got.controlInspectionState(); e != nil || state != "recovery-required" {
			t.Fatal("unsafe complete metadata reported closed", state, e)
		}
	}
	for _, change := range []func(*ExecutionDomainsObservation){
		func(o *ExecutionDomainsObservation) { o.origin = nil },
		func(o *ExecutionDomainsObservation) { o.journalPresent = false },
		func(o *ExecutionDomainsObservation) { o.systemObserved = false },
	} {
		got := o
		change(&got)
		if state, e := got.controlInspectionState(); e == nil || state != "" {
			t.Fatal("incomplete metadata fabricated result")
		}
	}
}
