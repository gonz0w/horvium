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
	"time"
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
	var deniedPrevious []byte
	for _, kind := range []string{"first", "successor"} {
		bound, e := source.Review(ctx, "root-publish-fixture", p, previous)
		if unsupported {
			check("enrollment_publish_native_"+kind, errors.Is(e, storage.ErrStateDirectory) && errors.Is(source.Publish(ctx, "root-publish-fixture", bound), storage.ErrInvalid))
		} else {
			if e != nil {
				t.Fatal(e)
			}
			if kind == "first" {
				// Same sealed binding remains usable after pre-attempt denials.
				// All altered output is a read-projection fixture, never a ZFS write.
				originalRead, originalClock := source.pool.read, source.now
				for _, mode := range []string{"cancelled", "expired", "backwards", "native-failure", "context", "property-drift"} {
					t.Run("enrollment-source-publication-"+mode, func(t *testing.T) {
						request, cancel := context.WithCancel(ctx)
						defer cancel()
						root := "root-publish-fixture"
						expected := storage.ErrExpired
						switch mode {
						case "cancelled":
							cancel()
							expected = context.Canceled
						case "expired":
							source.now = func() time.Time { return bound.Review().ExpiresAt }
						case "backwards":
							source.now = func() time.Time { return bound.Review().CreatedAt.Add(-time.Second) }
						case "native-failure":
							expected = ErrUnavailable
							source.pool.read = func(context.Context, poolProbe, string, int) (readOutput, error) { return readOutput{}, ErrUnavailable }
						case "context":
							root = "different-context"
							expected = storage.ErrDenied
						case "property-drift":
							expected = storage.ErrDrift
							changed := false
							source.pool.read = func(ctx context.Context, probe poolProbe, name string, budget int) (readOutput, error) {
								out, err := originalRead(ctx, probe, name, budget)
								if err == nil && probe == poolProperties {
									old := string(out.data)
									out = enrollmentPropertyDriftOutput(out)
									changed = changed || old != string(out.data)
								}
								return out, err
							}
							defer func() {
								if !changed {
									t.Error("property drift fixture changed nothing")
								}
							}()
						}
						err := source.Publish(request, root, bound)
						source.pool.read, source.now = originalRead, originalClock
						if !errors.Is(err, expected) {
							t.Fatal("publication denial", err, expected)
						}
						if _, err := os.Lstat(filepath.Join(path, "ownership-"+p.PoolGUID+".json")); !errors.Is(err, os.ErrNotExist) {
							t.Fatal("denial published metadata", err)
						}
					})
				}
			}
			check("enrollment_publish_native_"+kind, source.Publish(ctx, "root-publish-fixture", bound) == nil && !bound.Review().Observation.Eligible)
			if kind == "first" {
				check("enrollment_publish_native_repeat_denied", errors.Is(source.Publish(ctx, "root-publish-fixture", bound), storage.ErrDenied))
			}
		}
		if unsupported && kind == "first" {
			// First publication was refused: prove absence before independently
			// provisioning the explicit predecessor for the successor control.
			if _, e := os.Lstat(filepath.Join(path, "ownership-"+p.PoolGUID+".json")); !errors.Is(e, os.ErrNotExist) {
				t.Fatal("denied first enrollment created a record", e)
			}
			deniedPrevious, e = storage.EncodeOwnershipRecord(p)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(path, "ownership-"+p.PoolGUID+".json"), deniedPrevious, 0600); e != nil {
				t.Fatal(e)
			}
		}
		previous = p
		p.Revision++
	}
	if unsupported {
		check("enrollment_publish_native_repeat_denied", errors.Is(source.Publish(ctx, "root-publish-fixture", storage.PropertyEnrollmentState{}), storage.ErrInvalid))
	}
	got, e := os.ReadFile(filepath.Join(path, "ownership-"+p.PoolGUID+".json"))
	expected, encodeErr := storage.EncodeOwnershipRecord(previous)
	preserved := unsupported && e == nil && bytes.Equal(got, deniedPrevious) || !unsupported && e == nil && encodeErr == nil && bytes.Equal(got, expected)
	check("enrollment_publish_native_preserved", preserved && cleanup())
}
