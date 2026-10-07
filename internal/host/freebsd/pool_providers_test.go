// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

func providerCaptures() map[systemProbe]string {
	r := safetyCaptures()
	r[systemGEOM] = `<mesh><class id="d"><name>DISK</name>`
	for _, pair := range []struct{ name, serial string }{{"ada0", "SERIAL-A"}, {"ada1", "SERIAL-B"}} {
		r[systemGEOM] += `<geom id="g` + pair.name + `"><name>` + pair.name + `</name><config><ident>` + pair.serial + `</ident></config><provider id="p` + pair.name + `"><name>` + pair.name + `</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom>`
	}
	r[systemGEOM] += `</class><class id="dev"><name>DEV</name>`
	for _, name := range []string{"ada0", "ada1"} {
		r[systemGEOM] += `<geom id="dev` + name + `"><name>` + name + `</name><consumer id="c` + name + `"><provider ref="p` + name + `"/></consumer></geom>`
	}
	r[systemGEOM] += `</class></mesh>` + "\n"
	return r
}
func providerTestReader(rows map[systemProbe]string) systemReader {
	return func(_ context.Context, p systemProbe, name string, _ int) (readOutput, error) {
		if p == systemDiskInfo {
			return testOutput("/dev/" + name + "\t512\t67108864\t131072\t0\t0\n"), nil
		}
		if p == systemDiskSerial {
			return testOutput(map[string]string{"ada0": "SERIAL-A\n", "ada1": "SERIAL-B\n"}[name]), nil
		}
		return testOutput(rows[p]), nil
	}
}
func providersForTest(read systemReader) *PoolProviderSource {
	return &PoolProviderSource{source: useSource(read)}
}
func providerObservation(t *testing.T) storage.Observation {
	t.Helper()
	r := propertyCaptures()
	o, err := propertySource(func(_ context.Context, p poolProbe, _ string, _ int) (readOutput, error) {
		return testOutput(r[p]), nil
	}).Collect(context.Background(), "100", "tank")
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func TestPoolProvidersBindEveryLeafWithoutAuthority(t *testing.T) {
	r := providerCaptures()
	calls := map[string]int{}
	read := providerTestReader(r)
	s := providersForTest(func(ctx context.Context, p systemProbe, name string, n int) (readOutput, error) {
		if p == systemDiskInfo || p == systemDiskSerial {
			if name != "ada0" && name != "ada1" {
				t.Fatal("untrusted address")
			}
			calls[string(p)+name]++
		}
		return read(ctx, p, name, n)
	})
	o := providerObservation(t)
	disks, err := s.Collect(context.Background(), o)
	if err != nil || len(disks) != 2 || disks[0].Provider.Name != "ada0" || disks[1].Provider.Name != "ada1" || o.Eligible {
		t.Fatal("incomplete/authoritative identity", err)
	}
	for _, n := range calls {
		if n != 2 {
			t.Fatal("missing independent pass")
		}
	}
	disks[0].Provider.Identifiers[0].Value = "changed"
	again, err := s.Collect(context.Background(), o)
	if err != nil || again[0].Provider.Identifiers[0].Value == "changed" {
		t.Fatal("aliased observations", err)
	}
}
func TestPoolProvidersRefusePartialIdentityAndDrift(t *testing.T) {
	for _, mode := range []string{"duplicate", "missing", "serial", "geometry", "partition", "attachment", "topology", "failure", "drift", "valid-drift", "accounting", "limit", "bytes", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			r := providerCaptures()
			o := providerObservation(t)
			switch mode {
			case "duplicate":
				r[systemGEOM] = strings.ReplaceAll(r[systemGEOM], "SERIAL-B", "SERIAL-A")
			case "missing":
				r[systemGEOM] = strings.ReplaceAll(r[systemGEOM], "<ident>SERIAL-A</ident>", "")
			case "attachment":
				r[systemGEOM] = strings.Replace(r[systemGEOM], "<name>DEV</name>", "<name>PART</name>", 1)
			case "partition":
				o.Topology.Nodes[1].Address = "/dev/ada0p1"
			case "topology":
				r[systemPoolTopologyGUID] = strings.Replace(r[systemPoolTopologyGUID], "901 ONLINE", "904 ONLINE", 1)
			}
			read := providerTestReader(r)
			serialCalls, infoCalls := 0, 0
			s := providersForTest(func(ctx context.Context, p systemProbe, name string, n int) (readOutput, error) {
				if p == systemDiskSerial && name == "ada0" {
					serialCalls++
					if mode == "serial" || mode == "drift" && serialCalls == 2 {
						return testOutput("OTHER\n"), nil
					}
				}
				if p == systemDiskInfo && mode == "geometry" {
					return testOutput("/dev/" + name + "\t512\t33554432\t65536\t0\t0\n"), nil
				}
				if p == systemDiskInfo && name == "ada0" {
					infoCalls++
					if mode == "valid-drift" && infoCalls == 2 {
						return testOutput("/dev/ada0\t512\t67108864\t131072\t4096\t0\n"), nil
					}
				}
				if p == systemGEOM && mode == "failure" {
					return readOutput{}, ErrExit
				}
				if mode == "accounting" {
					return readOutput{data: []byte("x")}, nil
				}
				return read(ctx, p, name, n)
			})
			if mode == "limit" {
				s.source.records = 8
			}
			if mode == "bytes" {
				s.source.bytes = 8
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancel" {
				cancel()
			}
			disks, err := s.Collect(ctx, o)
			if err == nil || disks != nil {
				t.Fatal("partial provider proof", err)
			}
			if (mode == "topology" || mode == "valid-drift") && !errors.Is(err, storage.ErrDrift) {
				t.Fatal("topology drift", err)
			}
		})
	}
}
func TestPoolProvidersValidateBeforeNativeReads(t *testing.T) {
	o := providerObservation(t)
	for _, mode := range []string{"nil", "incomplete", "selection", "path"} {
		t.Run(mode, func(t *testing.T) {
			copy := o
			copy.Topology.Nodes = append([]storage.Vdev(nil), o.Topology.Nodes...)
			ctx := context.Background()
			switch mode {
			case "nil":
				ctx = nil
			case "incomplete":
				copy.Complete = false
			case "selection":
				copy.PoolName = "-option"
			case "path":
				copy.Topology.Nodes[1].Address = "/dev/-option"
			}
			s := providersForTest(func(context.Context, systemProbe, string, int) (readOutput, error) {
				t.Fatal("unvalidated read")
				return readOutput{}, nil
			})
			if d, err := s.Collect(ctx, copy); err == nil || d != nil {
				t.Fatal("unsafe selection")
			}
		})
	}
}
func FuzzPoolProviderCollection(f *testing.F) {
	f.Add([]byte("SERIAL-A\n"), uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, mode uint8) {
		if len(data) > 4096 {
			t.Skip()
		}
		r := providerCaptures()
		read := providerTestReader(r)
		s := providersForTest(func(ctx context.Context, p systemProbe, name string, n int) (readOutput, error) {
			if p == []systemProbe{systemGEOM, systemDiskInfo, systemDiskSerial, systemPoolTopologyPath}[mode%4] {
				return testOutput(string(data)), nil
			}
			return read(ctx, p, name, n)
		})
		d, err := s.Collect(context.Background(), providerObservation(t))
		if err != nil && !reflect.DeepEqual(d, []inventory.DiskObservation(nil)) {
			t.Fatal("partial fuzz result")
		}
	})
}
