// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// Only called inside the opted-in owned five-disk snapshot fixture. Candidates
// were independently checked against declared serial/geometry and root/use
// backing before entry. These intentionally bad identities never enroll.
func nativePropertyProviderDenials(t *testing.T, ctx context.Context, prefix string, duplicates []inventory.Provider, missing, unique inventory.Provider, check func(string, bool)) {
	t.Helper()
	command := func(ctx context.Context, path string, args ...string) (string, error) {
		out, err := execute(ctx, path, args, 1<<20)
		return string(out.data), err
	}
	observer, err := NewPoolPropertySource()
	if err != nil {
		t.Fatal(err)
	}
	providers, err := NewPoolProviderSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"duplicate", "missing"} {
		name := prefix + "_" + mode
		if storage.ValidatePoolSelection("1", name) != nil {
			t.Fatal("fixture name")
		}
		if _, err := command(ctx, "/sbin/zpool", "list", "-H", name); err == nil {
			t.Fatal("fixture exists")
		}
		guid := ""
		created := false
		cleanup := func() bool {
			if !created {
				return true
			}
			clean, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			actual, err := command(clean, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", name)
			if err != nil || guid == "" || strings.TrimSpace(actual) != guid {
				t.Error("owned provider pool identity lost")
				return false
			}
			if _, err = command(clean, "/sbin/zpool", "destroy", name); err != nil {
				t.Error("provider pool teardown", err)
				return false
			}
			created = false
			if _, err = command(clean, "/sbin/zpool", "list", "-H", name); err == nil {
				t.Error("provider pool remains")
				return false
			}
			return true
		}
		t.Cleanup(func() { cleanup() })
		args := []string{"create", "-m", "none", "-o", "cachefile=none", name}
		if mode == "duplicate" {
			args = append(args, "mirror", "/dev/"+duplicates[0].Name, "/dev/"+duplicates[1].Name)
		} else {
			args = append(args, "mirror", "/dev/"+unique.Name, "/dev/"+missing.Name)
		}
		if _, err = command(ctx, "/sbin/zpool", args...); err != nil {
			t.Fatal("owned provider fixture create", err)
		}
		created = true
		value, err := command(ctx, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", name)
		if err != nil {
			t.Fatal(err)
		}
		guid = strings.TrimSpace(value)
		o, err := observer.Collect(ctx, guid, name)
		if err != nil {
			t.Fatal("independent provider observation", err)
		}
		d, err := providers.Collect(ctx, o)
		check("providers_"+mode+"_denied", errors.Is(err, inventory.ErrDiskCorrelation) && d == nil && !o.Eligible)
		if !cleanup() {
			t.Fatal("provider fixture cleanup failed")
		}
	}
	check("provider_fixtures_teardown", true)
}
