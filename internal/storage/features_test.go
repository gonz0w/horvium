// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func featureFixture() PoolFeatures {
	f := PoolFeatures{PoolGUID: "100", PoolName: "tank", Format: "freebsd-14.5"}
	for _, name := range strings.Fields(commonFeatureNames) {
		f.Values = append(f.Values, PoolFeature{"feature@" + name, "enabled", Local})
	}
	return f
}
func featureRows(f PoolFeatures) string {
	s := f.PoolName + "\tguid\t" + f.PoolGUID + "\t-\n"
	for _, v := range f.Values {
		s += f.PoolName + "\t" + v.Name + "\t" + v.State + "\t" + string(v.Source) + "\n"
	}
	return s
}
func TestPoolFeaturesProjectionAndReview(t *testing.T) {
	want := featureFixture()
	f, err := ParsePoolFeatures([]byte(featureRows(want)), "tank", want.Format)
	if err != nil || !reflect.DeepEqual(f, want) {
		t.Fatal("native features lost", err)
	}
	if err := VerifyPoolFeatures("100", "tank", f); err != nil {
		t.Fatal(err)
	}
	p, o, i := fixture()
	o.Features = f
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Review().Features, f) {
		t.Fatal("features absent from review")
	}
	view := plan.Review()
	view.Features.Values[0].State = "disabled"
	if reflect.DeepEqual(view.Features, plan.Review().Features) {
		t.Fatal("review shares feature slice")
	}
	o.Features.Values[0].State = "active"
	if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
		t.Fatal("feature drift ignored", err)
	}
	if plan.Review().Features.Values[0].State != "enabled" {
		t.Fatal("input mutated review")
	}
}
func TestPoolFeaturesStrictEvidence(t *testing.T) {
	f := featureFixture()
	rows := featureRows(f)
	for _, data := range []string{"", strings.TrimSuffix(rows, "\n"), rows + "tank\tguid\t100\t-\n",
		strings.Replace(rows, "\tguid\t100", "\tguid\t0", 1),
		strings.Replace(rows, "\tguid\t100", "\tguid\t0100", 1),
		strings.Replace(rows, "\tenabled\tlocal", "\tunknown\tlocal", 1),
		strings.Replace(rows, "\tenabled\tlocal", "\tenabled\tdefault", 1),
		strings.Replace(rows, "feature@allocation_classes", "feature@bad-name", 1),
		strings.Replace(rows, "tank\tfeature@allocation_classes", "other\tfeature@allocation_classes", 1),
		strings.Replace(rows, "\tenabled", "\tenab\rled", 1), rows + string([]byte{255}),
		strings.Replace(rows, "tank\tfeature@allocation_classes\tenabled\tlocal\n", "", 1),
		rows + "tank\tfeature@allocation_classes\tenabled\tlocal\n"} {
		if got, err := ParsePoolFeatures([]byte(data), "tank", f.Format); err == nil || !reflect.DeepEqual(got, PoolFeatures{}) {
			t.Fatal("partial evidence", err)
		}
	}
	for _, data := range []string{strings.Repeat("a", maxPlanBytes+1), strings.Repeat("tank\tx\ty\t-\n", 257), "tank\tx\t" + strings.Repeat("a", maxFieldBytes+1) + "\t-\n"} {
		if _, err := ParsePoolFeatures([]byte(data), "tank", f.Format); !errors.Is(err, ErrLimit) {
			t.Fatal(err)
		}
	}
	if _, err := ParsePoolFeatures([]byte(rows), "tank", "unknown"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	// Non-feature fields are bounded shape-only observations, not eligibility proof.
	if _, err := ParsePoolFeatures([]byte(rows+"tank\tcomment\toperator text\tlocal\n"), "tank", f.Format); err != nil {
		t.Fatal(err)
	}
}
func TestUnknownAndUnsupportedFeaturesRemainUnavailable(t *testing.T) {
	for _, v := range []PoolFeature{{"feature@future_name", "disabled", Local}, {"unsupported@org.example:future", "inactive", Local}, {"unsupported@org.example:future", "readonly", Local}} {
		f := featureFixture()
		f.Values = append(f.Values, v)
		got, err := ParsePoolFeatures([]byte(featureRows(f)), "tank", f.Format)
		if err != nil {
			t.Fatal("observation discarded", err)
		}
		if err := VerifyPoolFeatures("100", "tank", got); !errors.Is(err, ErrUnavailable) {
			t.Fatal("unknown features qualified", err)
		}
		p, o, i := fixture()
		o.Features = got
		if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	}
}
func TestFeatureIdentityMissingAndOrdering(t *testing.T) {
	p, o, i := fixture()
	o.Features = featureFixture()
	plan, err := Build(instant, "session-a", p, o, i)
	if err != nil {
		t.Fatal(err)
	}
	for a, b := 0, len(o.Features.Values)-1; a < b; a, b = a+1, b-1 {
		o.Features.Values[a], o.Features.Values[b] = o.Features.Values[b], o.Features.Values[a]
	}
	if err := plan.Validate(instant, "session-a", p, o); err != nil {
		t.Fatal("ordering changes review", err)
	}
	for _, change := range []func(*Observation){func(o *Observation) { o.Features = PoolFeatures{} }, func(o *Observation) { o.Features.PoolGUID = "999" }, func(o *Observation) { o.Features.PoolName = "other" }, func(o *Observation) { o.Features.Values = o.Features.Values[:len(o.Features.Values)-1] }} {
		_, o, _ := fixture()
		o.Features = featureFixture()
		change(&o)
		if err := plan.Validate(instant, "session-a", p, o); !errors.Is(err, ErrDrift) {
			t.Fatal(err)
		}
	}
	_, o, _ = fixture()
	o.Features = PoolFeatures{}
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing features accepted", err)
	}
	_, o, _ = fixture()
	o.Features = featureFixture()
	o.Eligible = false
	if _, err := Build(instant, "session-a", p, o, i); !errors.Is(err, ErrUnavailable) {
		t.Fatal("features grant authority", err)
	}
}
func TestNewerFeatureFormat(t *testing.T) {
	f := featureFixture()
	f.Format = "freebsd-15.1"
	for _, name := range strings.Fields(newerFeatureNames) {
		f.Values = append(f.Values, PoolFeature{"feature@" + name, "disabled", Local})
	}
	got, err := ParsePoolFeatures([]byte(featureRows(f)), "tank", f.Format)
	if err != nil || len(got.Values) != 47 {
		t.Fatal(err)
	}
	if err := VerifyPoolFeatures("100", "tank", got); err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePoolFeatures([]byte(featureRows(featureFixture())), "tank", f.Format); err == nil {
		t.Fatal("incomplete newer table")
	}
}
func TestCommonProfileExactBytes(t *testing.T) {
	// Approved file reconstructed from factual names, including its pinned comment.
	data := []byte("# Features supported by OpenZFS 2.2 on Linux and FreeBSD\n" + strings.Join(strings.Fields(commonFeatureNames), "\n") + "\n")
	names, err := CommonFeatureProfile(data)
	if err != nil || len(names) != 39 {
		t.Fatal(err)
	}
	names[0] = "changed"
	again, err := CommonFeatureProfile(data)
	if err != nil || again[0] == "changed" {
		t.Fatal("shared profile data", err)
	}
	for _, bad := range [][]byte{nil, append(append([]byte(nil), data...), '\n'), []byte(strings.Replace(string(data), "blake3", "future", 1))} {
		if got, err := CommonFeatureProfile(bad); !errors.Is(err, ErrUnavailable) || got != nil {
			t.Fatal("profile fallback", err)
		}
	}
}
func FuzzPoolFeatures(f *testing.F) {
	f.Add([]byte(featureRows(featureFixture())))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParsePoolFeatures(data, "tank", "freebsd-14.5")
		if err != nil && !reflect.DeepEqual(got, PoolFeatures{}) {
			t.Fatal("partial features")
		}
	})
}
