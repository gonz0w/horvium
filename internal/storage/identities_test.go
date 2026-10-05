// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

const identityRows = "tank\t101\tfilesystem\n" +
	"tank/data\t102\tfilesystem\n" +
	"tank@seed\t201\tsnapshot\n" +
	"tank#seed\t201\tbookmark\n" +
	"tank/data@replica\t201\tsnapshot\n"

func TestIdentityProjectionBindsVersionsToOwners(t *testing.T) {
	ids, err := ParseIdentities([]byte(identityRows), "tank")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 5 {
		t.Fatal("version membership omitted")
	}
	for _, id := range ids {
		if id.Type == SnapshotType || id.Type == BookmarkType {
			expected := "101"
			if strings.HasPrefix(id.Name, "tank/data@") {
				expected = "102"
			}
			if id.OwnerGUID != expected || id.GUID != "201" {
				t.Fatal("version GUID/owner binding incorrect")
			}
		}
	}
	p, o, i := fixture()
	o.Identities = ids
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Review().Versions) != 3 {
		t.Fatal("versions omitted from review")
	}
	view := plan.Review()
	view.Versions[0].Name = "altered"
	if reflect.DeepEqual(view.Versions, plan.Review().Versions) {
		t.Fatal("review version slice is shared")
	}
}

func TestIdentityProjectionMalformedAndLimits(t *testing.T) {
	for _, tc := range []struct{ name, data string }{
		{"empty", ""},
		{"truncated", strings.TrimSuffix(identityRows, "\n")},
		{"unknown type", strings.Replace(identityRows, "\tbookmark\n", "\tunknown\n", 1)},
		{"duplicate name", identityRows + "tank\t103\tfilesystem\n"},
		{"duplicate filesystem guid", strings.Replace(identityRows, "\t102\tfilesystem", "\t101\tfilesystem", 1)},
		{"guid overflow", strings.Replace(identityRows, "\t102\tfilesystem", "\t18446744073709551616\tfilesystem", 1)},
		{"guid noncanonical", strings.Replace(identityRows, "\t102\tfilesystem", "\t0102\tfilesystem", 1)},
		{"missing owner", strings.Replace(identityRows, "tank/data@replica", "tank/missing@replica", 1)},
		{"missing parent", strings.ReplaceAll(identityRows, "tank/data", "tank/missing/data")},
		{"type delimiter mismatch", strings.Replace(identityRows, "tank#seed", "tank@other", 1)},
		{"foreign pool", strings.ReplaceAll(identityRows, "tank/data", "foreign/data")},
		{"control", strings.Replace(identityRows, "tank/data", "tank/da\rta", 1)},
		{"delimiter flood", strings.Replace(identityRows, "\t102\t", "\t102\t\t", 1)},
		{"utf8", identityRows + string([]byte{255})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ids, err := ParseIdentities([]byte(tc.data), "tank")
			if err == nil || ids != nil {
				t.Fatal("partial or invalid identity table retained", err)
			}
		})
	}
	for _, data := range [][]byte{make([]byte, maxObservationBytes+1), []byte(strings.Repeat("a", maxFieldBytes+1) + "\t101\tfilesystem\n"), []byte(strings.Repeat("a\t1\tfilesystem\n", maxRecords+1))} {
		if ids, err := ParseIdentities(data, "tank"); !errors.Is(err, ErrLimit) || ids != nil {
			t.Fatal(err)
		}
	}
}

func TestPropertyIdentityCoverageCannotOmitWholeFilesystem(t *testing.T) {
	_, o, _ := fixture()
	if err := VerifyFilesystemIdentities("tank", o.Filesystems, o.Identities); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Observation)
	}{
		{"missing property filesystem", func(o *Observation) { o.Filesystems = o.Filesystems[:1] }},
		{"missing identity filesystem", func(o *Observation) { o.Identities = o.Identities[:1] }},
		{"reused name different guid", func(o *Observation) { o.Identities[1].GUID = "999" }},
		{"missing entire identity table", func(o *Observation) { o.Identities = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, o, i := fixture()
			tc.change(&o)
			if _, err := Build(instant, "session-a", p, o, i); err == nil {
				t.Fatal("incomplete identity binding accepted")
			}
		})
	}
}

func TestVolumePresenceIsUnavailableForCurrentPlanning(t *testing.T) {
	ids, err := ParseIdentities([]byte(identityRows+"tank/volume\t301\tvolume\n"), "tank")
	if err != nil {
		t.Fatal(err)
	}
	p, o, i := fixture()
	o.Identities = ids
	if err := VerifyFilesystemIdentities("tank", o.Filesystems, ids); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestIdentityStabilityIgnoresOrderButRefusesChanges(t *testing.T) {
	ids, err := ParseIdentities([]byte(identityRows), "tank")
	if err != nil {
		t.Fatal(err)
	}
	reverse := append([]DatasetIdentity(nil), ids...)
	for left, right := 0, len(reverse)-1; left < right; left, right = left+1, right-1 {
		reverse[left], reverse[right] = reverse[right], reverse[left]
	}
	if err := VerifyIdentityStability("tank", ids, reverse); err != nil {
		t.Fatal(err)
	}
	reverse[0].GUID = "999"
	if err := VerifyIdentityStability("tank", ids, reverse); !errors.Is(err, ErrDrift) {
		t.Fatal(err)
	}
	if err := VerifyIdentityStability("tank", ids, ids[:len(ids)-1]); !errors.Is(err, ErrDrift) {
		t.Fatal(err)
	}
}

func TestVersionMembershipChangesRequireNewReview(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Observation)
	}{
		{"added snapshot", func(o *Observation) {
			o.Identities = append(o.Identities, DatasetIdentity{GUID: "202", Name: "tank/data@new", Type: SnapshotType, OwnerGUID: "102"})
		}},
		{"removed snapshot", func(o *Observation) { o.Identities = o.Identities[:len(o.Identities)-1] }},
		{"snapshot rename", func(o *Observation) {
			for n := range o.Identities {
				if o.Identities[n].Name == "tank@seed" {
					o.Identities[n].Name = "tank@renamed"
				}
			}
		}},
		{"snapshot name reused", func(o *Observation) {
			for n := range o.Identities {
				if o.Identities[n].Name == "tank@seed" {
					o.Identities[n].GUID = "999"
				}
			}
		}},
		{"wrong owner", func(o *Observation) { o.Identities[len(o.Identities)-1].OwnerGUID = "999" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, o, i := fixture()
			ids, err := ParseIdentities([]byte(identityRows), "tank")
			if err != nil {
				t.Fatal(err)
			}
			o.Identities = ids
			plan, err := Build(instant, "session-a", p, o, i)
			if err != nil {
				t.Fatal(err)
			}
			tc.change(&o)
			if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
				t.Fatal("version drift ignored", err)
			}
		})
	}
}

func TestUnrelatedVersionChangeDoesNotExpandScope(t *testing.T) {
	p, o, i := fixture()
	i.FilesystemGUID = "102"
	i.Property = RefQuota
	p.Scopes[1].Properties = append(p.Scopes[1].Properties, RefQuota)
	sibling := o.Filesystems[1]
	sibling.GUID = "103"
	sibling.Name = "tank/sibling"
	o.Filesystems = append(o.Filesystems, sibling)
	o.States = append(o.States, stateFixture(sibling))
	o.Identities = append(o.Identities, DatasetIdentity{GUID: "103", Name: "tank/sibling", Type: FilesystemType}, DatasetIdentity{GUID: "201", Name: "tank/sibling@seed", Type: SnapshotType, OwnerGUID: "103"})
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Review().Versions) != 0 {
		t.Fatal("unrelated version expanded review")
	}
	o.Identities[len(o.Identities)-1].GUID = "202"
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal(err)
	}
}

func FuzzIdentities(f *testing.F) {
	f.Add([]byte(identityRows))
	f.Add([]byte("tank\t101\tfilesystem\n"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		ids, err := ParseIdentities(data, "tank")
		if err != nil {
			if ids != nil {
				t.Fatal("partial rows")
			}
			return
		}
		if err := VerifyIdentityStability("tank", ids, ids); err != nil {
			t.Fatal("self-stability failed", err)
		}
	})
}

func TestVersionReviewBudgetAndOrdering(t *testing.T) {
	p, o, i := fixture()
	ids, err := ParseIdentities([]byte(identityRows), "tank")
	if err != nil {
		t.Fatal(err)
	}
	o.Identities = ids
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	for a, b := 0, len(ids)-1; a < b; a, b = a+1, b-1 {
		ids[a], ids[b] = ids[b], ids[a]
	}
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal("identity order altered review", err)
	}
	for n := 0; n < 1000; n++ {
		o.Identities = append(o.Identities, DatasetIdentity{GUID: "300", Name: "tank/data@version" + strconv.Itoa(n), Type: SnapshotType, OwnerGUID: "102"})
	}
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal("version impact silently truncated", err)
	}
}

func TestIdentityInputBudgets(t *testing.T) {
	p, o, i := fixture()
	o.Identities = make([]DatasetIdentity, maxRecords+1)
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	if err := VerifyIdentityStability("tank", o.Identities, o.Identities); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	_, o, _ = fixture()
	o.Identities[0].OwnerGUID = strings.Repeat("x", maxFieldBytes+1)
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
}
