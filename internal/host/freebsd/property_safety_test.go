// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/storage"
)

func safetyProvider() *PoolProviderSource {
	return providersForTest(providerTestReader(providerCaptures()))
}

func safetyCaptures() map[systemProbe]string {
	rows := systemCaptures()
	rows[systemGEOM] = `<mesh><class id="d"><name>DISK</name>`
	for _, entry := range []struct{ id, name string }{{"0", "boot0"}, {"1", "ada0"}, {"2", "ada1"}} {
		serial := ""
		if entry.name == "ada0" {
			serial = "SERIAL-A"
		}
		if entry.name == "ada1" {
			serial = "SERIAL-B"
		}
		rows[systemGEOM] += `<geom id="g` + entry.id + `"><name>` + entry.name + `</name><config><ident>` + serial + `</ident></config><provider id="p` + entry.id + `"><name>` + entry.name + `</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom>`
	}
	rows[systemGEOM] += `</class></mesh>` + "\n"
	rows[systemPools] = "tank\t100\t67108864\t0\t67108864\tONLINE\n"
	rows[systemPoolGUID] = "100\n"
	properties := propertyCaptures()
	rows[systemPoolTopologyGUID] = properties[poolTopologyGUID]
	rows[systemPoolTopologyPath] = properties[poolTopologyPath]
	return rows
}

func TestPropertySafetyCollectionRechecksPolicyAfterSystemReads(t *testing.T) {
	for _, mode := range []string{"revision", "missing", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			rows := safetyCaptures()
			changed := false
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := &PropertySafetySource{policy: policySource(func(string) (storage.Policy, error) {
				p := collectionPolicy()
				if changed {
					switch mode {
					case "revision":
						p.Revision++
					case "missing":
						return storage.Policy{}, storage.ErrStateRecord
					case "canceled":
						cancel()
					}
				}
				return p, nil
			}), providers: safetyProvider(), system: useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
				changed = true
				return testOutput(rows[p]), nil
			})}
			p, o, err := s.Collect(ctx, "100")
			if err == nil || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("policy changed during system reads accepted", err)
			}
			if mode == "revision" && !errors.Is(err, storage.ErrDrift) {
				t.Fatal("policy drift classification", err)
			}
		})
	}
}

func TestPropertySafetyCollectionJoinsExclusionsWithoutAuthority(t *testing.T) {
	rows := safetyCaptures()
	s := &PropertySafetySource{policy: policySource(func(string) (storage.Policy, error) { return collectionPolicy(), nil }), providers: safetyProvider(), system: useSource(func(_ context.Context, p systemProbe, name string, _ int) (readOutput, error) {
		if (p == systemPoolGUID || p == systemPoolTopologyGUID || p == systemPoolTopologyPath) && name != "tank" {
			t.Fatal("unvalidated pool")
		}
		return testOutput(rows[p]), nil
	})}
	p, o, err := s.Collect(context.Background(), "100")
	if err != nil || !o.Complete || o.Eligible || p.Revision != 1 {
		t.Fatal("incomplete or authoritative join", err)
	}
	for _, mode := range []string{"root", "system-mount", "swap", "native-failure", "topology-drift", "system-identity-drift"} {
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
			case "system-identity-drift":
				rows[systemGEOM] = strings.ReplaceAll(rows[systemGEOM], "SERIAL-A", "OTHER")
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

func TestPropertySafetyRequiresProviderProofAndFinalPolicy(t *testing.T) {
	for _, mode := range []string{"missing", "failed", "policy-drift"} {
		t.Run(mode, func(t *testing.T) {
			changed := false
			s := &PropertySafetySource{policy: policySource(func(string) (storage.Policy, error) {
				p := collectionPolicy()
				if changed {
					p.Revision++
				}
				return p, nil
			}), providers: safetyProvider(), system: useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
				return testOutput(safetyCaptures()[p]), nil
			})}
			read := s.providers.source.read
			s.providers.source.read = func(ctx context.Context, p systemProbe, n string, b int) (readOutput, error) {
				if mode == "failed" {
					return readOutput{}, ErrExit
				}
				if mode == "policy-drift" {
					changed = true
				}
				return read(ctx, p, n, b)
			}
			if mode == "missing" {
				s.providers = nil
			}
			p, o, err := s.Collect(context.Background(), "100")
			if err == nil || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("provider/policy gate bypass", err)
			}
			if mode == "policy-drift" && !errors.Is(err, storage.ErrDrift) {
				t.Fatal("policy drift", err)
			}
		})
	}
}
