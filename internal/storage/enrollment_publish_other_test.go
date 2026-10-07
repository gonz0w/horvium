// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEnrollmentPublicationUnavailableWithoutNativeWork(t *testing.T) {
	bound, o, d, u := enrollmentPublicationFixture(t)
	plan, e := BuildPropertyEnrollment(time.Now(), "root", bound.Review().Policy, o, d, u)
	if e != nil {
		t.Fatal(e)
	}
	bound.plan = plan
	if e := bound.Publish(context.Background(), "root", o, d, u); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
	if e := bound.Publish(context.Background(), "root", o, d, u); !errors.Is(e, ErrDenied) {
		t.Fatal("unsupported attempt replayed", e)
	}
}
