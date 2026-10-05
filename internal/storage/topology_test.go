// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func topologyFixture() PoolTopology {
	return PoolTopology{PoolGUID: "100", PoolName: "tank", Health: "ONLINE", Scan: "none requested", ScanReported: true, Nodes: []Vdev{
		{GUID: "900", Address: "mirror-0", Kind: "mirror", Health: "ONLINE"},
		{GUID: "901", ParentGUID: "900", Address: "/dev/ada0", Kind: "leaf", Health: "ONLINE"},
		{GUID: "902", ParentGUID: "900", Address: "/dev/ada1", Kind: "leaf", Health: "ONLINE", Ordinal: 1},
	}}
}

const topologyPreamble = "  pool: tank\n state: ONLINE\n  scan: none requested\nconfig:\n\n\tNAME STATE READ WRITE CKSUM\n\ttank ONLINE 0 0 0\n"
const topologyEnd = "\nerrors: No known data errors\n"
const topologyGUIDRows = topologyPreamble + "\t  900 ONLINE 0 0 0\n\t    901 ONLINE 0 0 0\n\t    902 ONLINE 0 0 0\n" + topologyEnd
const topologyPathRows = topologyPreamble + "\t  mirror-0 ONLINE 0 0 0\n\t    /dev/ada0 ONLINE 0 0 0\n\t    /dev/ada1 ONLINE 0 0 0\n" + topologyEnd

func TestTopologyProjectionAndReview(t *testing.T) {
	got, err := ParsePoolTopology([]byte(topologyGUIDRows), []byte(topologyPathRows), "tank", "100")
	if err != nil || !reflect.DeepEqual(got, topologyFixture()) {
		t.Fatal("native membership lost", err)
	}
	if err := VerifyPoolTopology("100", "tank", got); err != nil {
		t.Fatal(err)
	}
	p, o, i := fixture()
	o.Topology = got
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Review().Topology, got) {
		t.Fatal("topology missing")
	}
	copy := plan.Review()
	copy.Topology.Nodes[0].Address = "mirror-1"
	if copy.Topology.Nodes[0] == plan.Review().Topology.Nodes[0] {
		t.Fatal("shared review")
	}
	o.Topology.Nodes[1].Address = "/dev/ada2"
	if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
		t.Fatal("address drift ignored", err)
	}
	if plan.Review().Topology.Nodes[1].Address != "/dev/ada0" {
		t.Fatal("input mutated plan")
	}
}
func TestTopologyStrictEvidence(t *testing.T) {
	for _, data := range []string{"", strings.TrimSuffix(topologyGUIDRows, "\n"),
		strings.Replace(topologyGUIDRows, "NAME STATE READ WRITE CKSUM", "NAME STATE READ WRITE", 1),
		strings.Replace(topologyGUIDRows, "900 ONLINE", "0900 ONLINE", 1),
		strings.Replace(topologyGUIDRows, "900 ONLINE", "0 ONLINE", 1),
		strings.Replace(topologyGUIDRows, "902 ONLINE", "901 ONLINE", 1),
		strings.Replace(topologyGUIDRows, "900 ONLINE", "900 HEALTHY", 1),
		strings.Replace(topologyGUIDRows, "901 ONLINE 0", "901 ONLINE 0K", 1),
		strings.Replace(topologyGUIDRows, "901 ONLINE 0", "901 ONLINE 00", 1),
		strings.Replace(topologyGUIDRows, "901 ONLINE 0", "901 ONLINE 18446744073709551616", 1),
		strings.Replace(topologyGUIDRows, "\t    901", "\t     901", 1),
		strings.Replace(topologyGUIDRows, "\t    901", "\t      901", 1),
		strings.Replace(topologyGUIDRows, "errors: No known data errors", "", 1),
		strings.Replace(topologyGUIDRows, "state: ONLINE", "state: DEGRADED", 1),
		strings.Replace(topologyGUIDRows, "pool: tank", "pool: other", 1),
		topologyGUIDRows + topologyGUIDRows, topologyGUIDRows + "unknown: ignored\n", topologyGUIDRows + string([]byte{255}),
		strings.Replace(topologyGUIDRows, "900 ONLINE", "900 ON\rLINE", 1)} {
		if got, err := ParsePoolTopology([]byte(data), []byte(topologyPathRows), "tank", "100"); err == nil || !reflect.DeepEqual(got, PoolTopology{}) {
			t.Fatal("partial topology", err)
		}
	}
	for _, path := range []string{strings.Replace(topologyPathRows, "/dev/ada1", "/dev/ada0", 1), strings.Replace(topologyPathRows, "/dev/ada1", "/dev/../ada1", 1), strings.Replace(topologyPathRows, "902", "unknown", 1) + "\tlogs\n"} {
		if _, err := ParsePoolTopology([]byte(topologyGUIDRows), []byte(path), "tank", "100"); err == nil {
			t.Fatal("ambiguous path projection")
		}
	}
	for _, data := range [][]byte{make([]byte, (1<<20)+1), []byte(strings.Repeat("\n", 8193))} {
		if _, err := ParsePoolTopology(data, []byte(topologyPathRows), "tank", "100"); !errors.Is(err, ErrLimit) {
			t.Fatal(err)
		}
	}
}
func TestAdverseTopologyObservedButUnavailable(t *testing.T) {
	for _, change := range []func(*PoolTopology){
		func(x *PoolTopology) {
			x.Health = "DEGRADED"
			x.Nodes[0].Health = "DEGRADED"
			x.Nodes[1].Health = "OFFLINE"
		},
		func(x *PoolTopology) { x.Nodes[1].ChecksumErrors = 1 },
		func(x *PoolTopology) { x.ReadErrors = 1 },
		func(x *PoolTopology) { x.DataErrors = 1 },
		func(x *PoolTopology) { x.Scan = "resilvered 0B in 00:00:01 with 0 errors on fixture date" },
		func(x *PoolTopology) { x.Nodes[1].Annotation = "(resilvering)" },
		func(x *PoolTopology) { x.Nodes[0].Kind = "unsupported"; x.Nodes[0].Address = "raidz1-0" },
		func(x *PoolTopology) { x.Nodes = x.Nodes[:1] },
	} {
		x := topologyFixture()
		change(&x)
		if err := VerifyPoolTopology("100", "tank", x); err == nil {
			t.Fatal("adverse/unsupported topology qualified")
		}
		p, o, i := fixture()
		o.Topology = x
		if _, err := Build(instant, "session-a", p, o, i); err == nil {
			t.Fatal("adverse plan")
		}
	}
	guids := strings.ReplaceAll(topologyGUIDRows, "ONLINE", "DEGRADED")
	paths := strings.ReplaceAll(topologyPathRows, "ONLINE", "DEGRADED")
	paths = strings.Replace(paths, "/dev/ada0 DEGRADED", "/dev/ada0 OFFLINE", 1)
	guids = strings.Replace(guids, "901 DEGRADED", "901 OFFLINE", 1)
	got, err := ParsePoolTopology([]byte(guids), []byte(paths), "tank", "100")
	if err != nil || got.Nodes[1].Health != "OFFLINE" {
		t.Fatal("adverse observation discarded", err)
	}
	if err := VerifyPoolTopology("100", "tank", got); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
func TestTopologyMembershipAndOrderingDrift(t *testing.T) {
	p, o, i := fixture()
	o.Topology = topologyFixture()
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	for a, b := 0, len(o.Topology.Nodes)-1; a < b; a, b = a+1, b-1 {
		o.Topology.Nodes[a], o.Topology.Nodes[b] = o.Topology.Nodes[b], o.Topology.Nodes[a]
	}
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal("input ordering drift", err)
	}
	for _, change := range []func(*PoolTopology){
		func(x *PoolTopology) { x.Nodes[1].GUID = "903" },
		func(x *PoolTopology) { x.Nodes[1].Ordinal = 1; x.Nodes[2].Ordinal = 0 },
		func(x *PoolTopology) { x.PoolGUID = "999" },
		func(x *PoolTopology) { x.Diagnostics = "status: reviewed native warning" },
		func(x *PoolTopology) { *x = PoolTopology{} },
		func(x *PoolTopology) { x.Nodes = x.Nodes[:2] },
	} {
		_, o, _ := fixture()
		o.Topology = topologyFixture()
		change(&o.Topology)
		if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
			t.Fatal("membership drift ignored", err)
		}
	}
	if err := VerifyTopologyStability(topologyFixture(), topologyFixture()); err != nil {
		t.Fatal(err)
	}
	changed := topologyFixture()
	changed.Nodes[1].GUID = "903"
	if err := VerifyTopologyStability(topologyFixture(), changed); !errors.Is(err, ErrDrift) {
		t.Fatal(err)
	}
	_, o, _ = fixture()
	o.Topology = PoolTopology{}
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing topology accepted", err)
	}
}
func TestSupportedLayoutsAndTypedBounds(t *testing.T) {
	x := topologyFixture()
	x.Nodes[0].Kind = "raidz2"
	x.Nodes[0].Address = "raidz2-0"
	x.Nodes = append(x.Nodes, Vdev{GUID: "903", ParentGUID: "900", Address: "/dev/ada2", Kind: "leaf", Health: "ONLINE", Ordinal: 2}, Vdev{GUID: "904", ParentGUID: "900", Address: "/dev/ada3", Kind: "leaf", Health: "ONLINE", Ordinal: 3})
	if err := VerifyPoolTopology("100", "tank", x); err != nil {
		t.Fatal(err)
	}
	x.Nodes = x.Nodes[:4]
	if err := VerifyPoolTopology("100", "tank", x); !errors.Is(err, ErrUnavailable) {
		t.Fatal("short raidz2 accepted", err)
	}
	p, o, i := fixture()
	o.Topology = topologyFixture()
	o.Topology.Nodes = make([]Vdev, maxVdevs+1)
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
	o.Topology = topologyFixture()
	o.Topology.Scan = strings.Repeat("a", maxFieldBytes+1)
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrLimit) {
		t.Fatal(err)
	}
}
func FuzzPoolTopology(f *testing.F) {
	f.Add([]byte(topologyGUIDRows), []byte(topologyPathRows))
	f.Add([]byte{}, []byte{})
	f.Fuzz(func(t *testing.T, a, b []byte) {
		got, err := ParsePoolTopology(a, b, "tank", "100")
		if err != nil && !reflect.DeepEqual(got, PoolTopology{}) {
			t.Fatal("partial topology")
		}
	})
}

func TestTopologyGraphAndHomogeneousGroups(t *testing.T) {
	base := topologyFixture()
	base.Nodes = append(base.Nodes, Vdev{GUID: "910", Address: "mirror-1", Kind: "mirror", Health: "ONLINE", Ordinal: 1}, Vdev{GUID: "911", ParentGUID: "910", Address: "/dev/ada2", Kind: "leaf", Health: "ONLINE"}, Vdev{GUID: "912", ParentGUID: "910", Address: "/dev/ada3", Kind: "leaf", Health: "ONLINE", Ordinal: 1})
	if err := VerifyPoolTopology("100", "tank", base); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*PoolTopology){
		func(x *PoolTopology) { x.Nodes[1].ParentGUID = "absent" },
		func(x *PoolTopology) { x.Nodes[0].ParentGUID = "901" },
		func(x *PoolTopology) { x.Nodes[1].Ordinal = 1 },
		func(x *PoolTopology) { x.Nodes[0].GUID = "100" },
		func(x *PoolTopology) { x.Nodes[3].Kind = "raidz2"; x.Nodes[3].Address = "raidz2-1" },
		func(x *PoolTopology) {
			x.Nodes = append(x.Nodes, Vdev{GUID: "913", ParentGUID: "910", Address: "/dev/ada4", Kind: "leaf", Health: "ONLINE", Ordinal: 2})
		},
	} {
		x := base
		x.Nodes = append([]Vdev(nil), base.Nodes...)
		change(&x)
		if err := VerifyPoolTopology("100", "tank", x); err == nil {
			t.Fatal("invalid/mixed graph qualified")
		}
	}
}

func TestTopologyCountersAnnotationsAndUnavailableFormats(t *testing.T) {
	for _, pair := range [][2]string{
		{strings.Replace(topologyGUIDRows, "errors: No known data errors", "errors: 7 data errors, use '-v' for a list", 1), strings.Replace(topologyPathRows, "errors: No known data errors", "errors: 7 data errors, use '-v' for a list", 1)},
		{strings.Replace(topologyGUIDRows, "901 ONLINE 0 0 0", "901 ONLINE 0 0 19", 1), strings.Replace(topologyPathRows, "/dev/ada0 ONLINE 0 0 0", "/dev/ada0 ONLINE 0 0 19", 1)},
		{strings.Replace(topologyGUIDRows, "901 ONLINE 0 0 0", "901 ONLINE 0 0 0 (resilvering)", 1), strings.Replace(topologyPathRows, "/dev/ada0 ONLINE 0 0 0", "/dev/ada0 ONLINE 0 0 0 (resilvering)", 1)},
	} {
		got, err := ParsePoolTopology([]byte(pair[0]), []byte(pair[1]), "tank", "100")
		if err != nil {
			t.Fatal("adverse metadata lost", err)
		}
		if err := VerifyPoolTopology("100", "tank", got); !errors.Is(err, ErrUnavailable) {
			t.Fatal("errors/annotation qualified", err)
		}
	}
	for _, data := range []string{strings.Replace(topologyGUIDRows, "  scan:", "  expand:", 1), strings.Replace(topologyGUIDRows, "\t  900 ONLINE 0 0 0", "\tlogs", 1), strings.Replace(topologyGUIDRows, "901 ONLINE 0 0 0", "901 ONLINE 0 0 0 unknown-annotation", 1)} {
		if _, err := ParsePoolTopology([]byte(data), []byte(topologyPathRows), "tank", "100"); err == nil {
			t.Fatal("unqualified format accepted")
		}
	}
}

func TestQualifiedLegacyScanOmissionIsRetained(t *testing.T) {
	// Both inspected print_scan_status wrappers omit the line when there is no
	// scrub/error-scrub/resilver/rebuild history to print. Keep absence explicit.
	guids := strings.Replace(topologyGUIDRows, "  scan: none requested\n", "", 1)
	paths := strings.Replace(topologyPathRows, "  scan: none requested\n", "", 1)
	got, err := ParsePoolTopology([]byte(guids), []byte(paths), "tank", "100")
	if err != nil || got.Scan != "" || got.ScanReported {
		t.Fatal("qualified omission lost", err)
	}
	if err := VerifyPoolTopology("100", "tank", got); err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePoolTopology([]byte(guids), []byte(topologyPathRows), "tank", "100"); !errors.Is(err, ErrDrift) {
		t.Fatal("scan presence disagreement ignored", err)
	}
}
