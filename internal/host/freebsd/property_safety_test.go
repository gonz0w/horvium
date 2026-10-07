// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"reflect"
	"testing"

	"github.com/gonz0w/horvium/internal/storage"
)

func safetyCaptures() map[systemProbe]string {
	rows := systemCaptures()
	rows[systemGEOM] = `<mesh><class id="d"><name>DISK</name>`
	for _, entry := range []struct{ id, name string }{{"0", "boot0"}, {"1", "ada0"}, {"2", "ada1"}} {
		rows[systemGEOM] += `<geom id="g` + entry.id + `"><name>` + entry.name + `</name><provider id="p` + entry.id + `"><name>` + entry.name + `</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom>`
	}
	rows[systemGEOM] += `</class></mesh>` + "\n"
	rows[systemPools] = "tank\t100\t67108864\t0\t67108864\tONLINE\n"
	rows[systemPoolGUID] = "100\n"
	properties := propertyCaptures()
	rows[systemPoolTopologyGUID] = properties[poolTopologyGUID]
	rows[systemPoolTopologyPath] = properties[poolTopologyPath]
	return rows
}

func TestPropertySafetyCollectionJoinsExclusionsWithoutAuthority(t *testing.T) {
	rows := safetyCaptures()
	s := &PropertySafetySource{policy: policySource(func(string) (storage.Policy, error) { return collectionPolicy(), nil }), system: useSource(func(_ context.Context, p systemProbe, name string, _ int) (readOutput, error) {
		if (p == systemPoolGUID || p == systemPoolTopologyGUID || p == systemPoolTopologyPath) && name != "tank" {
			t.Fatal("unvalidated pool")
		}
		return testOutput(rows[p]), nil
	})}
	p, o, err := s.Collect(context.Background(), "100")
	if err != nil || !o.Complete || o.Eligible || p.Revision != 1 {
		t.Fatal("incomplete or authoritative join", err)
	}
	for _, mode := range []string{"root", "system-mount", "swap", "native-failure", "topology-drift"} {
		t.Run(mode, func(t *testing.T) {
			rows = safetyCaptures()
			switch mode {
			case "root":
				rows[systemMounts] = "/dev/ada0 / ufs rw 1 1\n"
			case "system-mount":
				rows[systemMounts] += "tank /root/state zfs rw 0 0\n"
			case "swap":
				rows[systemSwap] += "/dev/ada1 65532 0 65532 0%\n"
			case "topology-drift":
				rows[systemPoolTopologyGUID] = propertyCaptures()[poolTopologyGUID] + "unknown\n"
			case "native-failure":
				rows[systemPools] = "invalid\n"
			}
			p, o, err = s.Collect(context.Background(), "100")
			if err == nil || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("partial safety result", err)
			}
		})
	}
}
