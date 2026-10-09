// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestNativeJointExecutionJournal(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_STATE_VM") != "1" {
		t.Skip("disposable journal pair required")
	}
	anchor, e := nativeUFSStateAnchor(t, "horvium-joint-journal-", false)
	if e != nil {
		t.Fatal(e)
	}
	jp, lp := filepath.Join(anchor, "journal"), filepath.Join(anchor, "ledger")
	for _, name := range []string{jp, lp} {
		if os.Mkdir(name, 0700) != nil || os.WriteFile(filepath.Join(name, "writer.lock"), nil, 0600) != nil {
			t.Fatal("owned existing pair")
		}
	}
	ctx := context.Background()
	installation := strings.Repeat("1", 32)
	p, e := OpenExecutionDomainLeases(ctx, jp, lp)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = p.Close() })
	var markers []string
	check := func(name string, ok bool) {
		t.Helper()
		if !ok {
			t.Fatal("joint journal assertion", name)
		}
		markers = append(markers, name+"=pass")
	}
	// Supplied backing metadata exercises the storage contract; the separate host
	// suite qualifies actual guarded native identities. Actual objects/ACLs,
	// namespace reads, journal bytes and census here always come from owned UFS.
	read := func(context.Context, int) (ExecutionBackingObservation, error) {
		return ExecutionBackingObservation{Kind: "ufs", UFS: [2]uint32{11, 22}}, nil
	}
	first, e := p.Observe(ctx, installation, read)
	check("empty_pair", e == nil && first.origin == p && !first.journalPresent && first.journal == (executionJournal{}) && first.journalDigest == "" && validJobDigest(first.census))
	check("original_receipt_absent", !p.native.journal.acknowledged)
	filename := filepath.Join(jp, executionJournalName)
	j := executionJournalFixture()
	j.Binding = first.binding
	j.Census = first.census
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
		data, e := encodeExecutionJournal(jj)
		if e != nil || os.WriteFile(filename, data, 0600) != nil {
			t.Fatal("owned journal history")
		}
		got, e := p.Observe(ctx, installation, read)
		check("observed_"+name, e == nil && got.journalPresent && got.journal == jj && got.binding == first.binding && got.census == first.census && validJobDigest(got.journalDigest) && !p.native.journal.acknowledged)
	}
	got, e := p.Observe(ctx, installation, read)
	check("repeat_history", e == nil && got.journal.Owner.State == "closed")
	ownerPolicy, _, _ := fixture()
	data, e := EncodeOwnershipRecord(ownerPolicy)
	if e != nil || os.WriteFile(filepath.Join(lp, "ownership-100.json"), data, 0600) != nil {
		t.Fatal("owned retained ledger metadata")
	}
	got, e = p.Observe(ctx, installation, read)
	check("census_changed_fact", e == nil && got.census != first.census && got.journal.Census == first.census && got.journal.Owner.State == "closed")
	check("census_mismatch_no_authority", !p.native.journal.acknowledged && !p.native.journal.initializable)
	// No automatic clearance, journal rewrite or manufactured durable receipt.
	original, e := os.ReadFile(filename)
	if e != nil {
		t.Fatal(e)
	}
	check("same_record_preserved", string(original) == string(journalFixtureBytes(t, got.journal)))
	// Same semantic journal, actual replacement inode during the cross-domain read.
	step := 0
	changedRead := func(c context.Context, fd int) (ExecutionBackingObservation, error) {
		step++
		if step == 3 {
			stage := filepath.Join(jp, "owned-replacement")
			if os.WriteFile(stage, original, 0600) != nil || os.Rename(stage, filename) != nil {
				t.Fatal("owned journal replacement")
			}
		}
		return read(c, fd)
	}
	zero, e := p.Observe(ctx, installation, changedRead)
	check("journal_inode_drift", e != nil && zero == (ExecutionDomainsObservation{}))
	zero, e = p.Observe(ctx, installation, read)
	check("drift_retired", errors.Is(e, ErrStateUncertain) && zero == (ExecutionDomainsObservation{}))
	if p.Close() != nil {
		t.Fatal("retired owner close")
	}
	p, e = OpenExecutionDomainLeases(ctx, jp, lp)
	if e != nil {
		t.Fatal(e)
	}
	again, e := p.Observe(ctx, installation, read)
	check("reopen_observation", e == nil && again.origin == p && again.origin != first.origin && again.journal.Owner.State == "closed" && !p.native.journal.acknowledged)
	check("reopen_no_publisher", errors.Is(p.native.journal.publish(again.journal, nil), ErrDenied))
	// Persisted native binding mismatch cannot be converted into a new pair.
	foreign := again.journal
	foreign.Binding.Lock = strings.Repeat("f", 64)
	b, e := encodeExecutionJournal(foreign)
	if e != nil || os.WriteFile(filename, b, 0600) != nil {
		t.Fatal("owned foreign binding")
	}
	zero, e = p.Observe(ctx, installation, read)
	check("foreign_binding_denial", errors.Is(e, ErrDrift) && zero == (ExecutionDomainsObservation{}))
	if p.Close() != nil {
		t.Fatal("foreign owner close")
	}
	if os.WriteFile(filename, original, 0600) != nil {
		t.Fatal("owned original history restore")
	}
	p, e = OpenExecutionDomainLeases(ctx, jp, lp)
	if e != nil {
		t.Fatal(e)
	}
	_, e = p.Observe(ctx, installation, read)
	check("restored_observation", e == nil)
	// A different valid installation must retire the original successful binding.
	zero, e = p.Observe(ctx, strings.Repeat("2", 32), read)
	check("installation_drift", errors.Is(e, ErrDrift) && zero == (ExecutionDomainsObservation{}))
	check("final_close", p.Close() == nil)
	var jst, lst unix.Stat_t
	check("separate_original_objects", unix.Stat(jp, &jst) == nil && unix.Stat(lp, &lst) == nil && jst.Ino != lst.Ino)
	t.Log("JOINT_EXECUTION_JOURNAL_RESULT " + strings.Join(markers, " "))
}
