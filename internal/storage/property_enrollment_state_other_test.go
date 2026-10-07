// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestPropertyEnrollmentStateUnavailableWithoutNativeFilesystemWork(t *testing.T) {
	p, o, d, u := enrollmentFixture(t)
	plan, err := BuildPropertyEnrollment(instant, "root", p, o, d, u)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*StateDirectory{nil, {}} {
		bound, err := s.BindPropertyEnrollment(instant, "root", plan, Policy{}, o, d, u)
		if !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(bound, PropertyEnrollmentState{}) {
			t.Fatal("unsupported state binding", err)
		}
	}
}
