// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"strings"
	"testing"
)

const poolRows = "tank\tguid\t100\t-\n" +
	"tank\tload_guid\t200\t-\n" +
	"tank\thealth\tONLINE\t-\n" +
	"tank\treadonly\toff\tdefault\n" +
	"tank\tbootfs\t-\tdefault\n" +
	"tank\taltroot\t-\tdefault\n" +
	"tank\tcachefile\tnone\tlocal\n" +
	"tank\tcompatibility\toff\tdefault\n" + "tank\tmultihost\toff\tdefault\n"

func poolFixture() PoolState {
	return PoolState{GUID: "100", LoadGUID: "200", Name: "tank", GUIDSource: NoSource, LoadGUIDSource: NoSource,
		Health: PoolValue{"ONLINE", NoSource}, Readonly: PoolValue{"off", Default}, BootFS: PoolValue{"-", Default},
		AltRoot: PoolValue{"-", Default}, CacheFile: PoolValue{"none", Local}, Compatibility: PoolValue{"off", Default}, MultiHost: PoolValue{"off", Default}}
}

func TestPoolProjectionAndReview(t *testing.T) {
	pool, err := ParsePoolState([]byte(poolRows), "tank")
	if err != nil {
		t.Fatal(err)
	}
	if pool != poolFixture() {
		t.Fatal("native values/sources not retained")
	}
	p, o, i := fixture()
	o.Pool = pool
	if err := VerifyPoolState("100", "tank", pool); err != nil {
		t.Fatal(err)
	}
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Review().Pool != pool {
		t.Fatal("pool absent from review")
	}
	view := plan.Review()
	view.Pool.CacheFile.Effective = "/owned/cache"
	if view.Pool == plan.Review().Pool {
		t.Fatal("review shares pool data")
	}
	o.Pool.Compatibility = PoolValue{"future-profile,/operator/profile", Local}
	if plan.Review().Pool != pool {
		t.Fatal("input mutation changed review")
	}
	if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
		t.Fatal("compatibility policy drift ignored", err)
	}
}

func TestPoolProjectionDenials(t *testing.T) {
	for _, data := range []string{"", strings.TrimSuffix(poolRows, "\n"), poolRows + "tank\tguid\t100\t-\n",
		strings.Replace(poolRows, "\tload_guid\t200", "\tload_guid\t0", 1),
		strings.Replace(poolRows, "\tguid\t100", "\tguid\t0100", 1),
		strings.Replace(poolRows, "\tguid\t100", "\tguid\t18446744073709551616", 1),
		strings.Replace(poolRows, "\thealth\tONLINE", "\thealth\tONLINE-ish", 1),
		strings.Replace(poolRows, "\treadonly\toff", "\treadonly\tunknown", 1),
		strings.Replace(poolRows, "\treadonly\toff\tdefault", "\treadonly\toff\treceived", 1),
		strings.Replace(poolRows, "\tbootfs\t-", "\tbootfs\tother/root", 1),
		strings.Replace(poolRows, "\tcachefile\tnone", "\tcachefile\trelative", 1),
		strings.Replace(poolRows, "\taltroot\t-", "\taltroot\trelative", 1),
		strings.Replace(poolRows, "\tcompatibility\toff", "\tcompatibility\ta,,b", 1),
		strings.Replace(poolRows, "\tload_guid\t200\t-", "\tunknown\t200\t-", 1),
		strings.Replace(poolRows, "tank\thealth", "foreign\thealth", 1),
		strings.Replace(poolRows, "\treadonly\toff", "\treadonly\to\rff", 1),
		strings.Replace(poolRows, "\tguid\t100\t-", "\tguid\t100\t-\t", 1),
		poolRows + string([]byte{255}),
	} {
		if pool, err := ParsePoolState([]byte(data), "tank"); err == nil || pool != (PoolState{}) {
			t.Fatal("partial/invalid pool retained", err)
		}
	}
	for _, data := range [][]byte{make([]byte, maxPlanBytes+1), []byte(strings.Repeat("a", maxFieldBytes+1) + "\tguid\t100\t-\n")} {
		if pool, err := ParsePoolState(data, "tank"); !errors.Is(err, ErrLimit) || pool != (PoolState{}) {
			t.Fatal(err)
		}
	}
}

func TestPoolAdverseStatesRemainUnavailable(t *testing.T) {
	for _, tc := range []struct{ name, from, to string }{
		{"read only", "\treadonly\toff", "\treadonly\ton"},
		{"boot pool", "\tbootfs\t-", "\tbootfs\ttank/data"},
		{"degraded", "\thealth\tONLINE", "\thealth\tDEGRADED"},
		{"suspended", "\thealth\tONLINE", "\thealth\tSUSPENDED"},
		{"faulted", "\thealth\tONLINE", "\thealth\tFAULTED"},
		{"unknown", "\thealth\tONLINE", "\thealth\tUNKNOWN"},
		{"multihost", "\tmultihost\toff", "\tmultihost\ton"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := ParsePoolState([]byte(strings.Replace(poolRows, tc.from, tc.to, 1)), "tank")
			if err != nil {
				t.Fatal(err)
			}
			if err := VerifyPoolState("100", "tank", pool); !errors.Is(err, ErrUnavailable) {
				t.Fatal("adverse pool qualified", err)
			}
			p, o, i := fixture()
			o.Pool = pool
			if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrUnavailable) {
				t.Fatal(err)
			}
		})
	}
}

func TestPoolObservationIdentityAndDrift(t *testing.T) {
	for _, change := range []func(*Observation){
		func(o *Observation) { o.Pool = PoolState{} },
		func(o *Observation) { o.Pool.GUID = "999" },
		func(o *Observation) { o.Pool.Name = "other" },
		func(o *Observation) { o.Pool.LoadGUID = "201" },
		func(o *Observation) { o.Pool.AltRoot = PoolValue{"/owned/alt", Local} },
		func(o *Observation) { o.Pool.CacheFile = PoolValue{"/owned/cache", Local} },
		func(o *Observation) { o.Pool.Readonly.Source = Local },
	} {
		p, o, i := fixture()
		plan, err := Build(instant, "session-a", p, o, i)
		if err != nil {
			t.Fatal(err)
		}
		change(&o)
		if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
			t.Fatal("pool drift ignored", err)
		}
	}
	p, o, i := fixture()
	o.Pool = PoolState{}
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing pool observation accepted", err)
	}
	_, o, _ = fixture()
	o.Pool.GUID = "999"
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrDrift) {
		t.Fatal("caller pool GUID accepted", err)
	}
	_, o, _ = fixture()
	o.Eligible = false
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrUnavailable) {
		t.Fatal("pool table supplied write eligibility", err)
	}
}

func TestPoolProjectionOrderingAndLimits(t *testing.T) {
	lines := strings.Split(strings.TrimSuffix(poolRows, "\n"), "\n")
	for a, b := 0, len(lines)-1; a < b; a, b = a+1, b-1 {
		lines[a], lines[b] = lines[b], lines[a]
	}
	pool, err := ParsePoolState([]byte(strings.Join(lines, "\n")+"\n"), "tank")
	if err != nil || pool != poolFixture() {
		t.Fatal("row order matters", err)
	}
	p, o, i := fixture()
	o.Pool.Compatibility.Effective = strings.Repeat("a", maxFieldBytes+1)
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
}

func FuzzPoolState(f *testing.F) {
	f.Add([]byte(poolRows))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		pool, err := ParsePoolState(data, "tank")
		if err != nil && pool != (PoolState{}) {
			t.Fatal("partial pool retained")
		}
	})
}
