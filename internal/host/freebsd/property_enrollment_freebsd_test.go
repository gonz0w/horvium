// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// Only the opted-in parent supplies its owned pool/explicit root proposal.
// Native reads remain independent of the latched metadata-publication fixture.
func nativePropertyEnrollmentReviewFixture(t *testing.T, ctx context.Context, check func(string, bool), p storage.Policy, statePath, ordinary string, expected, sentinel []byte) {
	t.Helper()
	properties, err := NewPoolPropertySource()
	if err != nil {
		t.Fatal(err)
	}
	providers, err := NewPoolProviderSource()
	if err != nil {
		t.Fatal(err)
	}
	system, err := NewSystemUseSource()
	if err != nil {
		t.Fatal(err)
	}
	collect := func() (storage.Observation, []inventory.DiskObservation, storage.PoolSystemUse) {
		o, e := properties.Collect(ctx, p.PoolGUID, p.PoolName)
		if e != nil {
			t.Fatal(e)
		}
		d, e := providers.Collect(ctx, o)
		if e != nil {
			t.Fatal(e)
		}
		u, e := system.Collect(ctx)
		if e != nil {
			t.Fatal(e)
		}
		return o, d, u
	}
	o, d, u := collect()
	at := time.Now().UTC()
	plan, err := storage.BuildPropertyEnrollment(at, "root-fixture", p, o, d, u)
	if err != nil {
		t.Fatal("native enrollment review", err)
	}
	r := plan.Review()
	fresh, freshDisks, freshUse := collect()
	nativeEnrollmentStateFixture(t, ctx, check, p, fresh, freshDisks, freshUse, filepath.Dir(statePath))
	nativeEnrollmentPublicationFixture(t, ctx, check, p, filepath.Dir(statePath))
	check("enrollment_review_bound", plan.Validate(at, "root-fixture", p, fresh, freshDisks, freshUse) == nil && !r.Observation.Eligible && len(r.Policy.Scopes) == len(p.Scopes) && len(r.Policy.Members) == 2)
	check("enrollment_review_expired", errors.Is(plan.Validate(at.Add(5*time.Minute), "root-fixture", p, fresh, freshDisks, freshUse), storage.ErrExpired))
	check("enrollment_review_context_denied", errors.Is(plan.Validate(at, "other-fixture", p, fresh, freshDisks, freshUse), storage.ErrDenied))
	changed := plan.Review().Policy
	changed.Scopes[0].Properties = []storage.Property{storage.RefQuota}
	check("enrollment_review_policy_drift", errors.Is(plan.Validate(at, "root-fixture", changed, fresh, freshDisks, freshUse), storage.ErrDrift))
	changedDisks := append([]inventory.DiskObservation(nil), freshDisks...)
	changedDisks[0].StripeOffset += changedDisks[0].Provider.SectorSize
	check("enrollment_review_native_drift", errors.Is(plan.Validate(at, "root-fixture", p, fresh, changedDisks, freshUse), storage.ErrDrift))
	missing := p
	missing.Scopes = nil
	zero, e := storage.BuildPropertyEnrollment(at, "root-fixture", missing, fresh, freshDisks, freshUse)
	check("enrollment_review_missing_scope", e != nil && reflect.DeepEqual(zero, storage.PropertyEnrollmentPlan{}))
	r.Policy.Scopes[0].Properties[0] = storage.Compression
	r.Policy.Members[0].Serial = "changed"
	r.Observation.Filesystems[0].Name = "changed"
	r.Pools[0].Disks[0].Name = "changed"
	check("enrollment_review_immutable", plan.Validate(at, "root-fixture", p, fresh, freshDisks, freshUse) == nil && plan.Review().Policy.Members[0].Serial != "changed")
	record, e := os.ReadFile(filepath.Join(statePath, "ownership-"+p.PoolGUID+".json"))
	content, readErr := os.ReadFile(filepath.Join(ordinary, "sentinel"))
	check("enrollment_review_preserved", e == nil && readErr == nil && reflect.DeepEqual(record, expected) && reflect.DeepEqual(content, sentinel) && reflect.DeepEqual(o, fresh) && !fresh.Eligible)
}
