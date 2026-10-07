// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

// Only a new owned private root directory and the parent's declared mirror.
// All publication changes are metadata; no ZFS mutation is performed here.
func nativeEnrollmentPublicationFixture(t *testing.T, ctx context.Context, check func(string, bool), p storage.Policy, parent string) {
	t.Helper()
	path, e := os.MkdirTemp(parent, "enrollment-publication-")
	if e != nil {
		t.Fatal(e)
	}
	var state *storage.StateDirectory
	cleanup := func() bool {
		if state != nil {
			if state.Close() != nil {
				return false
			}
			state = nil
		}
		if os.RemoveAll(path) != nil {
			return false
		}
		_, e := os.Lstat(path)
		return errors.Is(e, os.ErrNotExist)
	}
	t.Cleanup(func() {
		if !cleanup() {
			t.Error("native enrollment publication cleanup")
		}
	})
	state, e = storage.OpenStateDirectory(path)
	if e != nil {
		t.Fatal(e)
	}
	source, e := NewPropertyEnrollmentSource(state)
	if e != nil {
		t.Fatal(e)
	}
	var fs unix.Statfs_t
	if e = unix.Statfs(path, &fs); e != nil {
		t.Fatal(e)
	}
	unsupported := unix.ByteSliceToString(fs.Fstypename[:]) == "ufs" && fs.Flags&unix.MNT_SOFTDEP != 0
	p.Revision = 1
	var previous storage.Policy
	for _, kind := range []string{"first", "successor"} {
		bound, e := source.Review(ctx, "root-publish-fixture", p, previous)
		if unsupported {
			check("enrollment_publish_native_"+kind, errors.Is(e, storage.ErrStateDirectory) && errors.Is(bound.Publish(ctx, "root-publish-fixture", storage.Observation{}, nil, storage.PoolSystemUse{}), storage.ErrInvalid))
		} else {
			if e != nil {
				t.Fatal(e)
			}
			o, d, u, e := source.collect(ctx, p)
			if e != nil {
				t.Fatal(e)
			}
			check("enrollment_publish_native_"+kind, bound.Publish(ctx, "root-publish-fixture", o, d, u) == nil && !o.Eligible)
			if kind == "first" {
				check("enrollment_publish_native_repeat_denied", errors.Is(bound.Publish(ctx, "root-publish-fixture", o, d, u), storage.ErrDenied))
			}
		}
		previous = p
		p.Revision++
	}
	if unsupported {
		check("enrollment_publish_native_repeat_denied", errors.Is((storage.PropertyEnrollmentState{}).Publish(ctx, "root-publish-fixture", storage.Observation{}, nil, storage.PoolSystemUse{}), storage.ErrInvalid))
	}
	got, e := os.ReadFile(filepath.Join(path, "ownership-"+p.PoolGUID+".json"))
	expected, encodeErr := storage.EncodeOwnershipRecord(previous)
	preserved := unsupported && errors.Is(e, os.ErrNotExist) || !unsupported && e == nil && encodeErr == nil && bytes.Equal(got, expected)
	check("enrollment_publish_native_preserved", preserved && cleanup())
}
