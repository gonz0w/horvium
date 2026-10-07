// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func systemCaptures() map[systemProbe]string {
	return map[systemProbe]string{
		systemVersions: "14.5-RELEASE\n14.5-RELEASE\n14.5-RELEASE\n",
		systemGEOM:     `<mesh><class id="d"><name>DISK</name><geom id="g"><name>boot0</name><provider id="p"><name>boot0</name><mediasize>67108864</mediasize><sectorsize>512</sectorsize></provider></geom></class></mesh>` + "\n",
		systemMounts:   "/dev/boot0 / ufs rw 1 1\ndevfs /dev devfs rw 0 0\n",
		systemSwap:     "Device 1K-blocks Used Avail Capacity\n", systemPools: "",
	}
}
func useSource(read systemReader) *SystemUseSource {
	return &SystemUseSource{read: read, bytes: 64 << 20, records: 100000, commandTimeout: time.Second, totalTimeout: time.Second}
}

func TestSystemUseCollectionCompleteWithoutAuthority(t *testing.T) {
	rows := systemCaptures()
	calls := map[systemProbe]int{}
	s := useSource(func(_ context.Context, p systemProbe, name string, _ int) (readOutput, error) {
		if name != "" {
			t.Fatal("unexpected pool input")
		}
		calls[p]++
		return testOutput(rows[p]), nil
	})
	u, err := s.Collect(context.Background())
	if err != nil || len(u.System().Mounts) != 2 || u.System().Mounts[0].Disks[0].Name != "boot0" || len(u.Pools()) != 0 {
		t.Fatal("incomplete use", err)
	}
	for p := range rows {
		if calls[p] != 2 {
			t.Fatal("missing pass", p)
		}
	}
	u.System().Mounts[0].Disks[0].Name = "changed"
	if u.System().Mounts[0].Disks[0].Name != "boot0" {
		t.Fatal("aliased result")
	}
}

func TestSystemUseCollectionDeniesFailureDriftAndLimits(t *testing.T) {
	for _, probe := range []systemProbe{systemVersions, systemGEOM, systemMounts, systemSwap, systemPools} {
		for _, mode := range []string{"failure", "malformed", "drift"} {
			t.Run(string(probe)+"/"+mode, func(t *testing.T) {
				rows := systemCaptures()
				calls := 0
				s := useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
					if p == probe {
						calls++
						if mode == "failure" {
							return readOutput{}, ErrExit
						}
						if mode == "malformed" || calls == 2 {
							return testOutput("unsupported\n"), nil
						}
					}
					return testOutput(rows[p]), nil
				})
				u, err := s.Collect(context.Background())
				if err == nil || !reflect.DeepEqual(u, storage.PoolSystemUse{}) {
					t.Fatal("partial output", err)
				}
			})
		}
	}
	for _, mode := range []string{"bytes", "records", "accounting", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			rows := systemCaptures()
			s := useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
				return testOutput(rows[p]), nil
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "bytes":
				s.bytes = 8
			case "records":
				s.records = 4
			case "accounting":
				s.read = func(context.Context, systemProbe, string, int) (readOutput, error) {
					return readOutput{data: []byte("x"), consumed: 0}, nil
				}
			case "cancel":
				cancel()
			case "nil":
				s = nil
			}
			u, err := s.Collect(ctx)
			if err == nil || !reflect.DeepEqual(u, storage.PoolSystemUse{}) {
				t.Fatal("unsafe budget/result", err)
			}
		})
	}
	if _, err := useSource(nil).Collect(nil); !errors.Is(err, storage.ErrInvalid) {
		t.Fatal("nil context", err)
	}
}

func TestSystemUseCollectionValidatesObservedPoolNamesBeforeLaunch(t *testing.T) {
	for _, name := range []string{"-option", "unsafe/path", "logpool", strings.Repeat("x", 1025)} {
		rows := systemCaptures()
		rows[systemPools] = name + "\t100\t67108864\t0\t67108864\tONLINE\n"
		s := useSource(func(_ context.Context, p systemProbe, name string, _ int) (readOutput, error) {
			if name != "" {
				t.Fatal("unsafe name launched")
			}
			return testOutput(rows[p]), nil
		})
		if _, err := s.Collect(context.Background()); err == nil {
			t.Fatal("unsafe catalog name")
		}
	}
}

func TestSystemUseCollectionCompletePoolsAndValidDrift(t *testing.T) {
	for _, mode := range []string{"complete", "root-drift", "topology-drift", "missing-guid", "missing-status"} {
		t.Run(mode, func(t *testing.T) {
			rows := safetyCaptures()
			calls := map[systemProbe]int{}
			s := useSource(func(_ context.Context, p systemProbe, name string, _ int) (readOutput, error) {
				calls[p]++
				if p == systemPoolGUID || p == systemPoolTopologyGUID || p == systemPoolTopologyPath {
					if name != "tank" {
						t.Fatal("pool selection")
					}
				}
				if mode == "missing-guid" && p == systemPoolGUID || mode == "missing-status" && p == systemPoolTopologyPath {
					return readOutput{}, ErrExit
				}
				data := rows[p]
				if calls[p] == 2 {
					if mode == "root-drift" && p == systemMounts {
						data = strings.Replace(data, "/dev/boot0", "/dev/ada0", 1)
					}
					if mode == "topology-drift" && p == systemPoolTopologyGUID {
						data = strings.Replace(data, "901 ONLINE", "904 ONLINE", 1)
					}
				}
				return testOutput(data), nil
			})
			u, err := s.Collect(context.Background())
			if mode == "complete" {
				if err != nil || len(u.Pools()) != 1 || u.Pools()[0].GUID != "100" {
					t.Fatal("incomplete imported pool", err)
				}
				return
			}
			if err == nil || !reflect.DeepEqual(u, storage.PoolSystemUse{}) {
				t.Fatal("partial/drifting use", err)
			}
			if strings.HasSuffix(mode, "drift") && !errors.Is(err, storage.ErrDrift) {
				t.Fatal("valid drift not compared", err)
			}
		})
	}
}

func FuzzSystemUseCollection(f *testing.F) {
	f.Add([]byte("/dev/boot0 / ufs rw 1 1\n"), uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, mode uint8) {
		if len(data) > 4096 {
			t.Skip()
		}
		rows := systemCaptures()
		rows[[]systemProbe{systemMounts, systemSwap, systemPools, systemGEOM}[mode%4]] = string(data)
		s := useSource(func(_ context.Context, p systemProbe, _ string, _ int) (readOutput, error) {
			return testOutput(rows[p]), nil
		})
		u, err := s.Collect(context.Background())
		if err != nil && !reflect.DeepEqual(u, storage.PoolSystemUse{}) {
			t.Fatal("partial fuzz result")
		}
	})
}
