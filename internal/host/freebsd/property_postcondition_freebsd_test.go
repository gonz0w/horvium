// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

// All writes below are TEST setup on the parent's declared disposable mirror.
// Test-only eligibility permits baseline capture, never product admission. The
// actual guarded two-pass collector remains ineligible throughout comparisons.
func nativePropertyPostconditions(t *testing.T, ctx context.Context, source *PoolPropertySource, guid, pool string) {
	t.Helper()
	must := func(t testing.TB, path string, args ...string) string {
		t.Helper()
		out, err := execute(ctx, path, args, 1<<20)
		if err != nil {
			t.Fatal("owned postcondition fixture command", err)
		}
		return string(out.data)
	}
	target := pool + "/renamed"
	projection := func(t testing.TB) string {
		return must(t, "/sbin/zfs", "get", "-H", "-p", "-r", "-t", "filesystem", "-o", "name,property,value,received,source", "guid,quota,refquota,compression,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus", pool)
	}
	original := projection(t)
	owned := []string{target + "/post-inherit", target + "/post-override", pool + "/post-received"}
	must(t, "/sbin/zfs", "create", "-o", "mountpoint=none", owned[0])
	must(t, "/sbin/zfs", "create", "-o", "mountpoint=none", "-o", "compression=lz4", owned[1])
	must(t, "/sbin/zfs", "create", "-o", "mountpoint=none", owned[1]+"/protected")
	stream, err := execute(ctx, "/sbin/zfs", []string{"send", "-p", target + "@baseline"}, 1<<20)
	if err != nil {
		t.Fatal("owned received fixture send")
	}
	receive := exec.CommandContext(ctx, "/sbin/zfs", "receive", "-u", owned[2])
	receive.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	receive.Stdin = bytes.NewReader(stream.data)
	receive.Stdout, receive.Stderr = io.Discard, io.Discard
	if receive.Run() != nil {
		t.Fatal("owned received fixture receive")
	}
	collect := func(t testing.TB) storage.Observation {
		t.Helper()
		o, err := source.Collect(ctx, guid, pool)
		if err != nil || !o.Complete || o.Eligible {
			t.Fatal("fresh ineligible postcondition collection", err)
		}
		return o
	}
	baseline := collect(t)
	policy := storage.Policy{Revision: 1, PoolGUID: guid, PoolName: pool}
	for _, f := range baseline.Filesystems {
		policy.Scopes = append(policy.Scopes, storage.Scope{GUID: f.GUID, Name: f.Name, ParentGUID: f.ParentGUID, Properties: []storage.Property{storage.Quota, storage.RefQuota, storage.Compression}})
	}
	find := func(t testing.TB, o storage.Observation, name string) storage.Filesystem {
		t.Helper()
		for _, f := range o.Filesystems {
			if f.Name == name {
				return f
			}
		}
		t.Fatal("owned postcondition filesystem missing")
		return storage.Filesystem{}
	}
	capture := func(t testing.TB, name string, property storage.Property, value string) storage.PropertyIntent {
		t.Helper()
		o := collect(t)
		// Only this test baseline uses a synthetic eligibility label. Matching an
		// actual collected graph is not permission to construct a runtime plan.
		o.Eligible = true
		at := time.Now()
		plan, err := storage.Build(at, "postcondition-native-fixture", policy, o, storage.Intent{FilesystemGUID: find(t, o, name).GUID, Property: property, Value: value})
		if err != nil {
			t.Fatal("test baseline review", err)
		}
		i, err := plan.CapturePropertyIntent(at, "postcondition-native-fixture", policy, o)
		if err != nil {
			t.Fatal("test baseline capture", err)
		}
		return i
	}
	match := func(t testing.TB, i storage.PropertyIntent) storage.Observation {
		t.Helper()
		o := collect(t)
		if err := i.ComparePostcondition(policy, o); err != nil {
			t.Fatal("actual native postcondition mismatch", err)
		}
		if o.Eligible {
			t.Fatal("comparison enabled writes")
		}
		return o
	}
	for _, tc := range []struct {
		name     string
		property storage.Property
		value    string
	}{
		{"quota", storage.Quota, "2097152"}, {"refquota", storage.RefQuota, "1048576"}, {"no-limit", storage.Quota, "none"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			i := capture(t, target, tc.property, tc.value)
			must(t, "/sbin/zfs", "set", string(tc.property)+"="+tc.value, target)
			match(t, i)
			if got := strings.TrimSpace(must(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value,received,source", string(tc.property), target)); got != tc.value+"\t-\tlocal" {
				t.Fatal("independent selected value/source mismatch")
			}
			if tc.property == storage.Quota {
				must(t, "/sbin/zfs", "set", "quota=1048576", target)
			} else {
				_, denied := execute(ctx, "/sbin/zfs", []string{"inherit", "refquota", target}, 1<<20)
				if !errors.Is(denied, ErrExit) {
					t.Fatal("non-inheritable fixture reset was not denied")
				}
				match(t, i) // Refused reset preserves the actual local value/source.
				must(t, "/sbin/zfs", "inherit", "-S", "refquota", target)
				if strings.TrimSpace(must(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value,received,source", "refquota", target)) != "none\t-\tdefault" {
					t.Fatal("received/default fixture reset mismatch")
				}
			}
		})
	}
	i := capture(t, target, storage.Compression, "off")
	encoded, err := storage.EncodePropertyIntent(i)
	if err != nil {
		t.Fatal(err)
	}
	must(t, "/sbin/zfs", "set", "compression=off", target)
	after := match(t, i)
	t.Run("compression", func(t *testing.T) {
		v := find(t, after, target).Properties.Compression
		if v.Effective != "off" || v.Source != storage.Local {
			t.Fatal("actual target compression")
		}
	})
	t.Run("inherited", func(t *testing.T) {
		v := find(t, after, owned[0]).Properties.Compression
		got := strings.TrimSpace(must(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value,source", "compression", owned[0]))
		if v.Effective != "off" || v.Source != storage.Inherited || v.SourceGUID != find(t, after, target).GUID || got != "off\tinherited from "+target {
			t.Fatal("actual inherited effect")
		}
	})
	t.Run("overrides", func(t *testing.T) {
		for _, name := range []string{owned[1], owned[1] + "/protected"} {
			a, b := find(t, baseline, name), find(t, after, name)
			if !reflect.DeepEqual(a, b) {
				t.Fatal("protected branch changed")
			}
		}
	})
	t.Run("wrong-source", func(t *testing.T) {
		must(t, "/sbin/zfs", "set", "compression=off", owned[0])
		if i.ComparePostcondition(policy, collect(t)) == nil {
			t.Fatal("identical local child accepted instead of inheritance")
		}
		must(t, "/sbin/zfs", "inherit", "compression", owned[0])
		match(t, i)
	})
	t.Run("context-drift", func(t *testing.T) {
		must(t, "/sbin/zfs", "set", "refquota=1048576", target)
		if i.ComparePostcondition(policy, collect(t)) == nil {
			t.Fatal("unreviewed property drift accepted")
		}
		must(t, "/sbin/zfs", "inherit", "-S", "refquota", target)
		match(t, i)
	})
	t.Run("baseline-retained", func(t *testing.T) {
		again, err := storage.EncodePropertyIntent(i)
		if err != nil || !bytes.Equal(encoded, again) {
			t.Fatal("immutable intent changed")
		}
		if !errors.Is(i.CompareBaseline(policy, after), storage.ErrDrift) && !errors.Is(i.CompareBaseline(policy, after), storage.ErrUnavailable) {
			t.Fatal("precondition validation relaxed")
		}
		if _, err := storage.Build(time.Now(), "denied", policy, after, i.Review().Intent); !errors.Is(err, storage.ErrUnavailable) {
			t.Fatal("native eligibility bypass")
		}
	})
	must(t, "/sbin/zfs", "set", "compression=lz4", target)
	t.Run("received", func(t *testing.T) {
		i := capture(t, owned[2], storage.Quota, "2097152")
		before := find(t, collect(t), owned[2]).Properties.Quota
		if before.Source != storage.Received || before.ReceivedValue != "1048576" {
			t.Fatal("received fixture baseline")
		}
		must(t, "/sbin/zfs", "set", "quota=2097152", owned[2])
		after := find(t, match(t, i), owned[2]).Properties.Quota
		got := strings.TrimSpace(must(t, "/sbin/zfs", "get", "-H", "-p", "-o", "value,received,source", "quota", owned[2]))
		if after.Source != storage.Local || after.ReceivedValue != before.ReceivedValue || got != "2097152\t1048576\tlocal" {
			t.Fatal("shadowed received value changed")
		}
	})
	t.Run("restored", func(t *testing.T) {
		// These names were created by this fixture under its parent's owned pool.
		for _, name := range owned {
			must(t, "/sbin/zfs", "destroy", "-r", name)
		}
		if projection(t) != original {
			t.Fatal("original property/state/identity rows changed")
		}
	})
}
