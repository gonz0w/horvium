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

// Synthetic qualified-format outputs; no synthetic eligibility authority.
func propertyCaptures() map[poolProbe]string {
	pool := "tank\tguid\t100\t-\ntank\tload_guid\t200\t-\ntank\thealth\tONLINE\t-\ntank\treadonly\toff\tdefault\ntank\tbootfs\t-\tdefault\ntank\taltroot\t-\tdefault\ntank\tcachefile\tnone\tlocal\ntank\tcompatibility\toff\tdefault\ntank\tmultihost\toff\tdefault\n"
	features := "tank\tguid\t100\t-\n"
	for _, name := range strings.Fields("allocation_classes async_destroy blake3 block_cloning bookmark_v2 bookmark_written bookmarks device_rebuild device_removal draid edonr embedded_data empty_bpobj enabled_txg encryption extensible_dataset filesystem_limits head_errlog hole_birth large_blocks large_dnode livelist log_spacemap lz4_compress multi_vdev_crash_dump obsolete_counts project_quota redacted_datasets redaction_bookmarks resilver_defer sha512 skein spacemap_histogram spacemap_v2 userobj_accounting vdev_zaps_v2 zilsaxattr zpool_checkpoint zstd_compress") {
		features += "tank\tfeature@" + name + "\tenabled\tlocal\n"
	}
	preamble := "  pool: tank\n state: ONLINE\n  scan: none requested\nconfig:\n\n\tNAME STATE READ WRITE CKSUM\n\ttank ONLINE 0 0 0\n"
	end := "\nerrors: No known data errors\n"
	return map[poolProbe]string{
		poolVersions:        "14.5-RELEASE\n14.5-RELEASE\n14.5-RELEASE\n",
		poolState:           pool,
		poolIdentities:      "tank\t101\tfilesystem\n",
		poolProperties:      "tank\tguid\t101\t-\t-\ntank\tquota\t0\t-\tdefault\ntank\trefquota\t0\t-\tdefault\ntank\tcompression\tlz4\t-\tlocal\n",
		poolFilesystemState: "tank\tguid\t101\t-\t-\ntank\tmountpoint\tnone\t-\tlocal\ntank\tcanmount\ton\t-\tdefault\ntank\treadonly\toff\t-\tdefault\ntank\tmounted\tno\t-\t-\ntank\tencryption\toff\t-\tdefault\ntank\tencryptionroot\t-\t-\t-\ntank\tkeystatus\tnone\t-\t-\n",
		poolFeatures:        features,
		poolTopologyGUID:    preamble + "\t  900 ONLINE 0 0 0\n\t    901 ONLINE 0 0 0\n\t    902 ONLINE 0 0 0\n" + end,
		poolTopologyPath:    preamble + "\t  mirror-0 ONLINE 0 0 0\n\t    /dev/ada0 ONLINE 0 0 0\n\t    /dev/ada1 ONLINE 0 0 0\n" + end,
	}
}

func propertySource(read poolReader) *PoolPropertySource {
	return &PoolPropertySource{read: read, bytes: 64 << 20, records: 100000, commandTimeout: time.Second, totalTimeout: time.Second}
}

func TestPoolPropertyCollectionIsCompleteWithoutAuthority(t *testing.T) {
	rows := propertyCaptures()
	calls := map[poolProbe]int{}
	s := propertySource(func(ctx context.Context, p poolProbe, name string, limit int) (readOutput, error) {
		if name != "tank" || limit <= 0 {
			t.Fatal("invalid native selection/budget")
		}
		calls[p]++
		return testOutput(rows[p]), nil
	})
	o, err := s.Collect(context.Background(), "100", "tank")
	if err != nil || !o.Complete || o.Eligible || o.Pool.LoadGUID != "200" || len(o.Filesystems) != 1 || len(o.Features.Values) != 39 || len(o.Topology.Nodes) != 3 {
		t.Fatal("incomplete or authoritative result", err)
	}
	for _, p := range poolProbes {
		if calls[p] != 2 {
			t.Fatal("missing independent second pass", p, calls[p])
		}
	}
	policy := storage.Policy{Revision: 1, PoolGUID: "100", PoolName: "tank", Scopes: []storage.Scope{{GUID: "101", Name: "tank", Properties: []storage.Property{storage.Quota}}}}
	if _, err := storage.Build(time.Now(), "session", policy, o, storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1"}); !errors.Is(err, storage.ErrUnavailable) {
		t.Fatal("collector granted native eligibility", err)
	}
	o.Filesystems[0].Name = "changed"
	again, err := s.Collect(context.Background(), "100", "tank")
	if err != nil || again.Filesystems[0].Name != "tank" {
		t.Fatal("result aliases reader/cached state", err)
	}
}

func TestPoolPropertyCollectionFailuresExposeNothing(t *testing.T) {
	for _, p := range poolProbes {
		for _, failure := range []error{ErrExit, ErrPermission, ErrUnavailable, ErrLimit, context.DeadlineExceeded, context.Canceled, nil} {
			t.Run(string(p)+"/"+map[bool]string{true: "malformed", false: "failed"}[failure == nil], func(t *testing.T) {
				rows := propertyCaptures()
				s := propertySource(func(_ context.Context, item poolProbe, _ string, _ int) (readOutput, error) {
					if item == p {
						return testOutput("private native output"), failure
					}
					return testOutput(rows[item]), nil
				})
				result, err := s.Collect(context.Background(), "100", "tank")
				if err == nil || !reflect.DeepEqual(result, storage.Observation{}) || strings.Contains(err.Error(), "private native output") {
					t.Fatal("failure exposed partial/native data", err)
				}
			})
		}
	}
}

func TestPoolPropertyCollectionDriftDeniesWholeResult(t *testing.T) {
	changes := map[poolProbe][2]string{
		poolVersions: {"14.5-RELEASE", "15.1-RELEASE"}, poolState: {"load_guid\t200", "load_guid\t201"},
		poolIdentities: {"101", "103"}, poolProperties: {"quota\t0", "quota\t1"}, poolFilesystemState: {"canmount\ton", "canmount\toff"},
		poolFeatures: {"enabled", "disabled"}, poolTopologyGUID: {"901 ONLINE", "903 ONLINE"}, poolTopologyPath: {"/dev/ada0", "/dev/ada2"},
	}
	for p, change := range changes {
		t.Run(string(p), func(t *testing.T) {
			rows := propertyCaptures()
			calls := map[poolProbe]int{}
			s := propertySource(func(_ context.Context, item poolProbe, _ string, _ int) (readOutput, error) {
				calls[item]++
				v := rows[item]
				if item == p && calls[item] == 2 {
					v = strings.ReplaceAll(v, change[0], change[1])
				}
				return testOutput(v), nil
			})
			result, err := s.Collect(context.Background(), "100", "tank")
			if err == nil || !reflect.DeepEqual(result, storage.Observation{}) {
				t.Fatal("mixed/drifting native result accepted", err)
			}
		})
	}
}

func TestPoolPropertySelectionAndBounds(t *testing.T) {
	rows := propertyCaptures()
	calls := 0
	s := propertySource(func(_ context.Context, p poolProbe, _ string, _ int) (readOutput, error) {
		calls++
		return testOutput(rows[p]), nil
	})
	for _, selection := range [][2]string{{"0", "tank"}, {"0100", "tank"}, {"100", "-tank"}, {"100", "tank/data"}, {"100", "tank;id"}, {"100", "mirror0"}, {"100", "tank\n"}} {
		if _, err := s.Collect(context.Background(), selection[0], selection[1]); err == nil || calls != 0 {
			t.Fatal("unsafe selection reached native reader", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Collect(ctx, "100", "tank"); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("cancelled read launched")
	}
	for _, kind := range []string{"byte exhaustion", "consumption mismatch", "record exhaustion", "unsupported release", "unknown pool", "nil context", "nil source"} {
		t.Run(kind, func(t *testing.T) {
			candidate := propertySource(s.read)
			guid := "100"
			ctx := context.Background()
			switch kind {
			case "byte exhaustion":
				candidate.bytes = 10
			case "consumption mismatch":
				candidate.read = func(context.Context, poolProbe, string, int) (readOutput, error) {
					return readOutput{data: []byte("x"), consumed: 0}, nil
				}
			case "record exhaustion":
				candidate.records = 1
			case "unsupported release":
				candidate.read = func(context.Context, poolProbe, string, int) (readOutput, error) {
					return testOutput("16.0-RELEASE\n16.0-RELEASE\n16.0-RELEASE\n"), nil
				}
			case "unknown pool":
				guid = "999"
			case "nil context":
				ctx = nil
			case "nil source":
				candidate = nil
			}
			result, err := candidate.Collect(ctx, guid, "tank")
			if err == nil || !reflect.DeepEqual(result, storage.Observation{}) {
				t.Fatal("invalid inputs/budget exposed observation", err)
			}
		})
	}
}

func TestPoolPropertyCancellationDuringCollection(t *testing.T) {
	rows := propertyCaptures()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := propertySource(func(_ context.Context, p poolProbe, _ string, _ int) (readOutput, error) {
		if p == poolProperties {
			cancel()
		}
		return testOutput(rows[p]), nil
	})
	if result, err := s.Collect(ctx, "100", "tank"); !errors.Is(err, context.Canceled) || !reflect.DeepEqual(result, storage.Observation{}) {
		t.Fatal("cancellation returned partial/success", err)
	}
}

func TestPoolPropertyTopologyOrder(t *testing.T) {
	rows := propertyCaptures()
	rows[poolTopologyGUID] = strings.ReplaceAll(rows[poolTopologyGUID], "900 ONLINE", "9900 ONLINE")
	s := propertySource(func(_ context.Context, p poolProbe, _ string, _ int) (readOutput, error) {
		return testOutput(rows[p]), nil
	})
	o, err := s.Collect(context.Background(), "100", "tank")
	if err != nil || o.Topology.Nodes[0].Kind != "leaf" || !matchesCollectedMirror(o.Topology, []string{"/dev/ada0", "/dev/ada1"}) {
		t.Fatal("fixture assumed positional topology", err)
	}
	if matchesCollectedMirror(o.Topology, []string{"/dev/ada0", "/dev/ada2"}) {
		t.Fatal("fixture accepted wrong member")
	}
}

// Native topology is GUID-sorted, not traversal-ordered. Compare declared
// members and their parent identity without assigning meaning to slice offsets.
func matchesCollectedMirror(topology storage.PoolTopology, leaves []string) bool {
	if len(topology.Nodes) != 3 || len(leaves) != 2 || leaves[0] == leaves[1] {
		return false
	}
	expected := map[string]bool{leaves[0]: true, leaves[1]: true}
	parent := ""
	for _, n := range topology.Nodes {
		if n.Kind == "mirror" {
			if parent != "" || n.ParentGUID != "" {
				return false
			}
			parent = n.GUID
		}
	}
	if parent == "" {
		return false
	}
	for _, n := range topology.Nodes {
		if n.GUID == parent {
			continue
		}
		if n.Kind != "leaf" || n.ParentGUID != parent || !expected[n.Address] {
			return false
		}
		delete(expected, n.Address)
	}
	return len(expected) == 0
}

func FuzzPoolPropertyCollection(f *testing.F) {
	f.Add([]byte("hostile output"), "100", "tank", uint8(2))
	f.Add([]byte{}, "0100", "-tank", uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, guid, name string, probe uint8) {
		if len(data) > 4096 || len(guid) > 4096 || len(name) > 4096 {
			return
		}
		rows := propertyCaptures()
		s := propertySource(func(_ context.Context, p poolProbe, _ string, _ int) (readOutput, error) {
			if p == poolProbes[int(probe)%len(poolProbes)] {
				return readOutput{data: data, consumed: len(data)}, nil
			}
			return testOutput(rows[p]), nil
		})
		result, err := s.Collect(context.Background(), guid, name)
		if err != nil && !reflect.DeepEqual(result, storage.Observation{}) {
			t.Fatal("partial result")
		}
		if err == nil && (!result.Complete || result.Eligible) {
			t.Fatal("incomplete/authoritative success")
		}
	})
}
