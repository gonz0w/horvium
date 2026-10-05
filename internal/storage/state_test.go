// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func stateFixture(f Filesystem) FilesystemState {
	return FilesystemState{GUID: f.GUID, Name: f.Name,
		Mountpoint:     Value{Effective: "none", Source: Local, SourceGUID: f.GUID},
		CanMount:       Value{Effective: "on", Source: Default},
		Readonly:       Value{Effective: "off", Source: Default},
		Mounted:        Value{Effective: "no", Source: NoSource},
		Encryption:     Value{Effective: "off", Source: Default},
		EncryptionRoot: Value{Effective: "-", Source: NoSource},
		KeyStatus:      Value{Effective: "none", Source: NoSource},
	}
}

const stateRootRows = "tank\tguid\t101\t-\t-\n" +
	"tank\tmountpoint\tnone\t-\tlocal\n" +
	"tank\tcanmount\ton\t-\tdefault\n" +
	"tank\treadonly\toff\t-\tlocal\n" +
	"tank\tmounted\tno\t-\t-\n" +
	"tank\tencryption\toff\t-\tdefault\n" +
	"tank\tencryptionroot\t-\t-\t-\n" +
	"tank\tkeystatus\tnone\t-\t-\n"
const stateChildRows = "tank/data\tguid\t102\t-\t-\n" +
	"tank/data\tmountpoint\tnone\t-\tlocal\n" +
	"tank/data\tcanmount\tnoauto\t-\tlocal\n" +
	"tank/data\treadonly\toff\t-\tinherited from tank\n" +
	"tank/data\tmounted\tno\t-\t-\n" +
	"tank/data\tencryption\toff\t-\tdefault\n" +
	"tank/data\tencryptionroot\t-\t-\t-\n" +
	"tank/data\tkeystatus\t-\t-\t-\n"

func TestFilesystemStateProjection(t *testing.T) {
	states, err := ParseFilesystemStates([]byte(stateChildRows+stateRootRows), "tank")
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 || states[1].Readonly.SourceGUID != "101" || states[1].CanMount.Effective != "noauto" || states[1].KeyStatus.Effective != "-" {
		t.Fatal("native state/source not retained")
	}
	p, o, i := fixture()
	o.States = states
	if err := VerifyFilesystemStates("tank", o.Identities, states); err != nil {
		t.Fatal(err)
	}
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Review().States) != 2 {
		t.Fatal("review omitted state")
	}
	view := plan.Review()
	view.States[0].Mountpoint.Effective = "legacy"
	if reflect.DeepEqual(view.States, plan.Review().States) {
		t.Fatal("review shares state slice")
	}
	o.States[0].Mountpoint.Effective = "legacy"
	if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
		t.Fatal("mount policy drift ignored", err)
	}
}

func TestFilesystemStateDenials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Observation)
	}{
		{"missing table", func(o *Observation) { o.States = nil }},
		{"whole filesystem omitted", func(o *Observation) { o.States = o.States[:1] }},
		{"GUID reused", func(o *Observation) { o.States[1].GUID = "999"; o.States[1].Mountpoint.SourceGUID = "999" }},
		{"readonly foreign child", func(o *Observation) { o.States[1].Readonly = Value{Effective: "on", Source: Local, SourceGUID: "102"} }},
		{"temporary override", func(o *Observation) {
			o.States[1].Readonly = Value{Effective: "off", Source: Temporary, SourceGUID: "102"}
		}},
		{"encrypted unlocked", func(o *Observation) {
			o.States[1].Encryption = Value{Effective: "aes-256-gcm", Source: Local, SourceGUID: "102"}
			o.States[1].EncryptionRoot = Value{Effective: "tank/data", Source: NoSource}
			o.States[1].EncryptionRootGUID = "102"
			o.States[1].KeyStatus = Value{Effective: "available", Source: NoSource}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, o, i := fixture()
			tc.change(&o)
			if _, err := Build(instant, "session-a", p, o, i); err == nil {
				t.Fatal("unsafe or incomplete state accepted")
			}
		})
	}
}

func TestFilesystemStateMalformedProjection(t *testing.T) {
	rows := stateRootRows + stateChildRows
	for _, data := range []string{"", strings.TrimSuffix(rows, "\n"), rows + "tank\tguid\t101\t-\t-\n",
		strings.Replace(rows, "\tcanmount\ton", "\tcanmount\tunknown", 1),
		strings.Replace(rows, "\tmounted\tno", "\tmounted\tmaybe", 1),
		strings.Replace(rows, "\tencryption\toff", "\tencryption\tunknown", 1),
		strings.Replace(rows, "\tmountpoint\tnone", "\tmountpoint\trelative", 1),
		strings.Replace(rows, "\treadonly\toff\t-\tinherited from tank", "\treadonly\ton\t-\tinherited from tank", 1),
		strings.Replace(rows, "\treadonly\toff\t-\tinherited from tank", "\treadonly\toff\t-\tinherited from tank/missing", 1),
		strings.Replace(rows, "tank/data\tguid\t102", "tank/data\tguid\t101", 1),
		strings.Replace(rows, "\tcanmount\tnoauto\t-\tlocal", "\tcanmount\tnoauto\t-\tinherited from tank", 1),
		strings.Replace(rows, "\tencryptionroot\t-", "\tencryptionroot\ttank", 1),
		strings.Replace(rows, "tank/data\tguid", "foreign/data\tguid", 1),
		strings.Replace(rows, "\treadonly\toff", "\treadonly\to\rff", 1),
		rows + string([]byte{255}),
		strings.Replace(rows, "\tguid\t101\t-\t-", "\tguid\t101\t-\t-\t", 1),
	} {
		if got, err := ParseFilesystemStates([]byte(data), "tank"); err == nil || got != nil {
			t.Fatal("partial or invalid state retained", err)
		}
	}
	for _, data := range [][]byte{make([]byte, maxObservationBytes+1), []byte(strings.Repeat("a", maxFieldBytes+1) + "\tguid\t101\t-\t-\n"), []byte(strings.Repeat("a\tguid\t1\t-\t-\n", 8*maxRecords+1))} {
		if got, err := ParseFilesystemStates(data, "tank"); !errors.Is(err, ErrLimit) || got != nil {
			t.Fatal(err)
		}
	}
}

func TestFilesystemStateDriftAndOrdering(t *testing.T) {
	for _, change := range []func(*Observation){
		func(o *Observation) { o.States[1].CanMount = Value{Effective: "off", Source: Local, SourceGUID: "102"} },
		func(o *Observation) { o.States[1].Mounted.Effective = "yes" },
		func(o *Observation) { o.States[1].Readonly = Value{Effective: "off", Source: Local, SourceGUID: "102"} },
		func(o *Observation) { o.States[1].Mountpoint.ReceivedValue = "legacy" },
	} {
		p, o, i := fixture()
		plan, err := Build(instant, "session-a", p, o, i)
		if err != nil {
			t.Fatal(err)
		}
		change(&o)
		if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
			t.Fatal(err)
		}
	}
	p, o, i := fixture()
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	o.States[0], o.States[1] = o.States[1], o.States[0]
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal(err)
	}
}

func FuzzFilesystemStates(f *testing.F) {
	f.Add([]byte(stateRootRows + stateChildRows))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		states, err := ParseFilesystemStates(data, "tank")
		if err != nil && states != nil {
			t.Fatal("partial result")
		}
	})
}

func TestEncryptedAndReadonlyStatesRemainObservable(t *testing.T) {
	_, o, _ := fixture()
	for _, keyStatus := range []string{"available", "unavailable"} {
		child := strings.Replace(stateChildRows, "\tencryption\toff\t-\tdefault", "\tencryption\taes-256-gcm\t-\tlocal", 1)
		child = strings.Replace(child, "\tencryptionroot\t-\t-\t-", "\tencryptionroot\ttank/data\t-\t-", 1)
		child = strings.Replace(child, "\tkeystatus\t-\t-\t-", "\tkeystatus\t"+keyStatus+"\t-\t-", 1)
		states, err := ParseFilesystemStates([]byte(stateRootRows+child), "tank")
		if err != nil {
			t.Fatal(err)
		}
		if states[1].EncryptionRootGUID != "102" || states[1].KeyStatus.Effective != keyStatus {
			t.Fatal("crypto constraints omitted")
		}
		if err := VerifyFilesystemStates("tank", o.Identities, states); !errors.Is(err, ErrUnavailable) {
			t.Fatal("encrypted state granted qualification", err)
		}
	}
	rows := strings.ReplaceAll(stateRootRows+stateChildRows, "\treadonly\toff", "\treadonly\ton")
	states, err := ParseFilesystemStates([]byte(rows), "tank")
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyFilesystemStates("tank", o.Identities, states); !errors.Is(err, ErrUnavailable) {
		t.Fatal("readonly state granted qualification", err)
	}
}

func TestFilesystemStateInputBudgets(t *testing.T) {
	p, o, i := fixture()
	o.States = make([]FilesystemState, maxRecords+1)
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	_, o, _ = fixture()
	o.States[0].Mountpoint.Effective = strings.Repeat("x", maxFieldBytes+1)
	if err := VerifyFilesystemStates("tank", o.Identities, o.States); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
}
