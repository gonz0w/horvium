// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func postconditionIntent(t testing.TB, p Policy, o Observation, change Intent) PropertyIntent {
	t.Helper()
	plan, err := Build(instant, "postcondition-fixture", p, o, change)
	if err != nil {
		t.Fatal(err)
	}
	i, err := plan.CapturePropertyIntent(instant.Add(time.Second), "postcondition-fixture", p, o)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func TestPropertyPostconditionExactLocalValues(t *testing.T) {
	for _, property := range []Property{Quota, RefQuota, Compression} {
		for _, value := range []string{"none", "18446744073709551615", "off", "lz4"} {
			change := Intent{FilesystemGUID: "101", Property: property, Value: value}
			if !validIntent(change) {
				continue
			}
			t.Run(string(property)+"/"+value, func(t *testing.T) {
				p, o, _ := fixture()
				// Isolate the selected property: compression's child overrides it.
				o.Filesystems[1].Properties.Compression = Value{"off", Local, "102", ""}
				i := postconditionIntent(t, p, o, change)
				o.Eligible = false
				if !(property == Compression && value == "lz4") && i.ComparePostcondition(p, o) == nil {
					t.Fatal("unchanged source accepted")
				}
				switch property {
				case Quota:
					o.Filesystems[0].Properties.Quota = Value{value, Local, "101", ""}
				case RefQuota:
					o.Filesystems[0].Properties.RefQuota = Value{value, Local, "101", ""}
				case Compression:
					o.Filesystems[0].Properties.Compression = Value{value, Local, "101", ""}
				}
				if err := i.ComparePostcondition(p, o); err != nil {
					t.Fatal(err)
				}
				if o.Eligible {
					t.Fatal("comparison granted eligibility")
				}
			})
		}
	}
}

func TestPropertyPostconditionCompressionInheritanceAndOverrides(t *testing.T) {
	for _, source := range []Source{Local, Received} {
		t.Run(string(source), func(t *testing.T) {
			p, o, change := fixture()
			change.Property, change.Value = Compression, "off"
			o.Filesystems[1].Properties.Compression = Value{"lz4", source, "102", ""}
			if source == Received {
				o.Filesystems[1].Properties.Compression.ReceivedValue = "lz4"
			}
			protected := o.Filesystems[1]
			protected.GUID, protected.Name, protected.ParentGUID = "103", "tank/data/protected", "102"
			protected.Properties.Compression = Value{"lz4", Inherited, "102", ""}
			inheritor := o.Filesystems[1]
			inheritor.GUID, inheritor.Name, inheritor.ParentGUID = "104", "tank/inheritor", "101"
			inheritor.Properties.Compression = Value{"lz4", Inherited, "101", ""}
			for _, f := range []Filesystem{protected, inheritor} {
				o.Filesystems = append(o.Filesystems, f)
				o.States = append(o.States, stateFixture(f))
				o.Identities = append(o.Identities, DatasetIdentity{GUID: f.GUID, Name: f.Name, Type: FilesystemType})
			}
			p.Scopes = append(p.Scopes, Scope{GUID: "104", Name: "tank/inheritor", ParentGUID: "101", Properties: []Property{Compression}})
			i := postconditionIntent(t, p, o, change)
			o.Eligible = false
			o.Filesystems[0].Properties.Compression = Value{"off", Local, "101", ""}
			if i.ComparePostcondition(p, o) == nil {
				t.Fatal("stale inheritor accepted")
			}
			o.Filesystems[3].Properties.Compression = Value{"off", Inherited, "101", ""}
			if err := i.ComparePostcondition(p, o); err != nil {
				t.Fatal(err)
			}
			o.Filesystems[2].Properties.Compression.Effective = "off"
			if i.ComparePostcondition(p, o) == nil {
				t.Fatal("protected branch changed")
			}
		})
	}
}

func TestPropertyPostconditionRetainsShadowedReceivedValues(t *testing.T) {
	for _, property := range []Property{Quota, RefQuota, Compression} {
		p, o, change := fixture()
		change.Property, change.Value = property, "4194304"
		if property == Compression {
			change.Value = "off"
			o.Filesystems[1].Properties.Compression = Value{"lz4", Local, "102", ""}
		}
		before := Value{"none", Received, "101", "none"}
		if property == Compression {
			before = Value{"lz4", Received, "101", "lz4"}
		}
		switch property {
		case Quota:
			o.Filesystems[0].Properties.Quota = before
		case RefQuota:
			o.Filesystems[0].Properties.RefQuota = before
		case Compression:
			o.Filesystems[0].Properties.Compression = before
		}
		i := postconditionIntent(t, p, o, change)
		after := Value{change.Value, Local, "101", before.ReceivedValue}
		switch property {
		case Quota:
			o.Filesystems[0].Properties.Quota = after
		case RefQuota:
			o.Filesystems[0].Properties.RefQuota = after
		case Compression:
			o.Filesystems[0].Properties.Compression = after
		}
		o.Eligible = false
		if err := i.ComparePostcondition(p, o); err != nil {
			t.Fatal(property, err)
		}
		after.ReceivedValue = ""
		switch property {
		case Quota:
			o.Filesystems[0].Properties.Quota = after
		case RefQuota:
			o.Filesystems[0].Properties.RefQuota = after
		case Compression:
			o.Filesystems[0].Properties.Compression = after
		}
		if i.ComparePostcondition(p, o) == nil {
			t.Fatal("lost received value accepted", property)
		}
	}
}

func TestPropertyPostconditionRebindsInheritedTargetSupplier(t *testing.T) {
	p, o, change := fixture()
	change.FilesystemGUID, change.Property, change.Value = "102", Compression, "off"
	f := o.Filesystems[1]
	f.GUID, f.Name, f.ParentGUID = "103", "tank/data/child", "102"
	o.Filesystems = append(o.Filesystems, f)
	o.States = append(o.States, stateFixture(f))
	o.Identities = append(o.Identities, DatasetIdentity{GUID: f.GUID, Name: f.Name, Type: FilesystemType})
	p.Scopes = append(p.Scopes, Scope{GUID: f.GUID, Name: f.Name, ParentGUID: f.ParentGUID, Properties: []Property{Compression}})
	i := postconditionIntent(t, p, o, change)
	o.Eligible = false
	o.Filesystems[1].Properties.Compression = Value{"off", Local, "102", ""}
	o.Filesystems[2].Properties.Compression = Value{"off", Inherited, "102", ""}
	if err := i.ComparePostcondition(p, o); err != nil {
		t.Fatal(err)
	}
	o.Filesystems[2].Properties.Compression.SourceGUID = "101"
	if i.ComparePostcondition(p, o) == nil {
		t.Fatal("old supplying ancestor accepted")
	}
}

func TestPropertyPostconditionRejectsUnrelatedContextAndPolicyDrift(t *testing.T) {
	for _, mode := range []string{"value", "source", "received", "other-property", "descendant", "identity", "version", "state", "pool", "features", "topology", "policy", "members", "incomplete", "missing", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			i, _, p, o := propertyIntentFixture(t)
			o.Filesystems[0].Properties.Quota = Value{i.Review().Intent.Value, Local, "101", ""}
			o.Eligible = false
			if err := i.ComparePostcondition(p, o); err != nil {
				t.Fatal("control", err)
			}
			switch mode {
			case "value":
				o.Filesystems[0].Properties.Quota.Effective = "4194304"
			case "source":
				o.Filesystems[0].Properties.Quota.Source = Default
				o.Filesystems[0].Properties.Quota.SourceGUID = ""
			case "received":
				o.Filesystems[0].Properties.Quota.ReceivedValue = "1048576"
			case "other-property":
				o.Filesystems[0].Properties.RefQuota = Value{"1048576", Local, "101", ""}
			case "descendant":
				o.Filesystems[1].Properties.Quota = Value{"1048576", Local, "102", ""}
			case "identity":
				o.Identities[0].GUID = "999"
			case "version":
				o.Identities[2].Name = "tank/data@changed"
			case "state":
				o.States[0].CanMount = Value{"off", Local, "101", ""}
			case "pool":
				o.Pool.LoadGUID = "999"
			case "features":
				o.Features.Format = "unqualified"
			case "topology":
				o.Topology.Nodes[1].GUID = "999"
			case "policy":
				p.Revision++
			case "members":
				p.Members = enrolledPolicy().Members
			case "incomplete":
				o.Complete = false
			case "missing":
				o.Filesystems = nil
			case "oversized":
				o.Filesystems[0].Name = strings.Repeat("x", maxFieldBytes+1)
			}
			if i.ComparePostcondition(p, o) == nil {
				t.Fatal("drift accepted")
			}
		})
	}
}

func TestPropertyPostconditionImmutableDecodedAndOrdered(t *testing.T) {
	i, _, p, o := propertyIntentFixture(t)
	encoded, _ := EncodePropertyIntent(i)
	decoded, err := DecodePropertyIntent(encoded)
	if err != nil {
		t.Fatal(err)
	}
	o.Filesystems[0].Properties.Quota = Value{i.Review().Intent.Value, Local, "101", ""}
	o.Eligible = false
	o.Filesystems[0], o.Filesystems[1] = o.Filesystems[1], o.Filesystems[0]
	o.States[0], o.States[1] = o.States[1], o.States[0]
	p.Scopes[0], p.Scopes[1] = p.Scopes[1], p.Scopes[0]
	before, _ := json.Marshal(struct {
		P Policy
		O Observation
	}{p, o})
	if err := decoded.ComparePostcondition(p, o); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(struct {
		P Policy
		O Observation
	}{p, o})
	again, _ := EncodePropertyIntent(decoded)
	if !bytes.Equal(encoded, again) || !bytes.Equal(before, after) {
		t.Fatal("caller or immutable baseline changed")
	}
	if decoded.CompareBaseline(p, o) == nil {
		t.Fatal("baseline validation relaxed")
	}
	if err := (PropertyIntent{}).ComparePostcondition(p, o); err == nil {
		t.Fatal("zero baseline accepted")
	}
	if !errors.Is(i.ComparePostcondition(p, Observation{}), ErrUnavailable) {
		t.Fatal("incomplete observation accepted")
	}
}

func FuzzPropertyPostcondition(f *testing.F) {
	i, _, p, o := propertyIntentFixture(f)
	f.Add(i.Review().Intent.Value)
	f.Add("none")
	f.Add("\xff")
	f.Fuzz(func(t *testing.T, value string) {
		observed := o
		observed.Filesystems = append([]Filesystem(nil), o.Filesystems...)
		observed.Eligible = false
		observed.Filesystems[0].Properties.Quota = Value{value, Local, "101", ""}
		err := i.ComparePostcondition(p, observed)
		if (err == nil) != (value == i.Review().Intent.Value) {
			t.Fatal("unexpected postcondition result")
		}
		if !reflect.DeepEqual(o.Filesystems[0].Properties.Quota, Value{"none", Default, "", ""}) {
			t.Fatal("shared baseline modified")
		}
	})
}
